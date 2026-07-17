import { existsSync } from 'node:fs';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const defaultOutputDir = resolve(frontendRoot, '.artifacts/table-hardening-browser');
const viewportMatrix = [
  { name: 'mobile-360x640', width: 360, height: 640, key: '360x640' },
  { name: 'mobile-390x844', width: 390, height: 844, key: '390x844' },
  { name: 'mobile-430x932', width: 430, height: 932, key: '430x932' },
  { name: 'tablet-768x1024', width: 768, height: 1024, key: '768x1024' },
  { name: 'desktop-1280x720', width: 1280, height: 720, key: '1280x720' }
];
const cardReadabilityViewportKeys = new Set(['360x640', '1280x720']);
const cardReadabilitySamples = ['10h', 'ad', 'ks', 'qc'];
const rectangleTargets = [
  { name: 'screen', selector: '.room-mobile-screen', required: true },
  { name: 'stage', selector: '.room-stage', required: true },
  { name: 'topbar', selector: '.table-topbar-compact', required: true },
  { name: 'table', selector: '.table-area', required: true },
  { name: 'center', selector: '.table-center-stack-compact', required: false },
  { name: 'seatNodes', selector: '.seat-ring-casino .casino-seat-node', required: false },
  { name: 'hero', selector: '.hero-panel', required: false },
  { name: 'actions', selector: '.hero-actions-safe, .owner-action-row-dock', required: false },
  { name: 'toolbar', selector: '.mock-bottom-toolbar-safe', required: false },
  { name: 'drawer', selector: '.table-drawer', required: false }
];
const viewportBoundTargets = new Set(['screen', 'stage', 'topbar', 'center', 'seatNodes', 'hero', 'actions', 'toolbar', 'drawer']);
const blockingOverlapPairs = [
  ['topbar', 'center'],
  ['topbar', 'seatNodes'],
  ['topbar', 'hero'],
  ['topbar', 'actions'],
  ['topbar', 'toolbar'],
  ['center', 'seatNodes'],
  ['center', 'hero'],
  ['center', 'actions'],
  ['center', 'toolbar'],
  ['seatNodes', 'hero'],
  ['seatNodes', 'actions'],
  ['seatNodes', 'toolbar'],
  ['hero', 'actions'],
  ['hero', 'toolbar'],
  ['actions', 'toolbar']
];
const clickableSelector = [
  'button:not([disabled])',
  'a[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[role="button"]:not([aria-disabled="true"])'
].join(', ');

function usage() {
  return [
    '用法：npm run test:browser:table-hardening -- --url <牌桌 URL> [选项]',
    '',
    '选项：',
    '  --url <url>                 已登录用户可访问的正式牌桌 URL',
    '  --token <token>             注入 depu.auth.token sessionStorage',
    '  --storage-state <path>      Playwright storageState 文件',
    `  --output <path>             输出目录，默认 ${defaultOutputDir}`,
    '  --wait-ms <milliseconds>    页面稳定等待时间，默认 500',
    '  --headed                    使用有头浏览器',
    '  --help                      显示帮助',
    '',
    '也可使用 DEPU_TABLE_URL、DEPU_AUTH_TOKEN、DEPU_STORAGE_STATE 和 DEPU_BROWSER_ACCEPTANCE_OUTPUT。'
  ].join('\n');
}

function readOptionValue(args, index, optionName) {
  const value = args[index + 1];
  if (!value || value.startsWith('--')) throw new Error(`${optionName} 缺少参数`);
  return value;
}

function parseArgs(args) {
  const options = {
    url: process.env.DEPU_TABLE_URL || '',
    token: process.env.DEPU_AUTH_TOKEN || '',
    storageState: process.env.DEPU_STORAGE_STATE || '',
    outputDir: process.env.DEPU_BROWSER_ACCEPTANCE_OUTPUT || defaultOutputDir,
    waitMs: 500,
    headed: false,
    help: false
  };

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === '--url') {
      options.url = readOptionValue(args, index, arg);
      index += 1;
    } else if (arg === '--token') {
      options.token = readOptionValue(args, index, arg);
      index += 1;
    } else if (arg === '--storage-state') {
      options.storageState = readOptionValue(args, index, arg);
      index += 1;
    } else if (arg === '--output') {
      options.outputDir = readOptionValue(args, index, arg);
      index += 1;
    } else if (arg === '--wait-ms') {
      options.waitMs = Number(readOptionValue(args, index, arg));
      index += 1;
    } else if (arg === '--headed') {
      options.headed = true;
    } else if (arg === '--help') {
      options.help = true;
    } else {
      throw new Error(`未知参数：${arg}`);
    }
  }

  if (!Number.isInteger(options.waitMs) || options.waitMs < 0) {
    throw new Error('--wait-ms 必须是非负整数');
  }

  return options;
}

async function collectRectangles(page) {
  return page.evaluate((targets) => {
    const rounded = (rectangle) => ({
      x: Math.round(rectangle.x * 100) / 100,
      y: Math.round(rectangle.y * 100) / 100,
      width: Math.round(rectangle.width * 100) / 100,
      height: Math.round(rectangle.height * 100) / 100,
      top: Math.round(rectangle.top * 100) / 100,
      right: Math.round(rectangle.right * 100) / 100,
      bottom: Math.round(rectangle.bottom * 100) / 100,
      left: Math.round(rectangle.left * 100) / 100
    });

    return Object.fromEntries(targets.map(({ name, selector }) => {
      const rectangles = [...document.querySelectorAll(selector)]
        .map((element) => {
          const style = window.getComputedStyle(element);
          const rectangle = element.getBoundingClientRect();
          if (style.display === 'none' || style.visibility === 'hidden' || rectangle.width <= 0 || rectangle.height <= 0) {
            return null;
          }
          return rounded(rectangle);
        })
        .filter(Boolean);
      return [name, rectangles];
    }));
  }, rectangleTargets);
}

async function collectClickHits(page) {
  return page.evaluate((selector) => {
    const candidates = [...document.querySelectorAll(selector)];
    return candidates.flatMap((candidate, index) => {
      const style = window.getComputedStyle(candidate);
      const rectangle = candidate.getBoundingClientRect();
      const left = Math.max(0, rectangle.left);
      const right = Math.min(window.innerWidth, rectangle.right);
      const top = Math.max(0, rectangle.top);
      const bottom = Math.min(window.innerHeight, rectangle.bottom);
      if (
        style.display === 'none'
        || style.visibility === 'hidden'
        || style.pointerEvents === 'none'
        || right <= left
        || bottom <= top
      ) {
        return [];
      }

      const x = Math.round((left + right) / 2);
      const y = Math.round((top + bottom) / 2);
      const hit = document.elementFromPoint(x, y);
      const passed = Boolean(hit && (hit === candidate || candidate.contains(hit)));
      const text = (candidate.getAttribute('aria-label') || candidate.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80);

      return [{
        index,
        tag: candidate.tagName.toLowerCase(),
        testId: candidate.getAttribute('data-testid') || '',
        text,
        rectangle: {
          x: Math.round(rectangle.x * 100) / 100,
          y: Math.round(rectangle.y * 100) / 100,
          width: Math.round(rectangle.width * 100) / 100,
          height: Math.round(rectangle.height * 100) / 100
        },
        point: { x, y },
        hitTag: hit?.tagName.toLowerCase() || '',
        hitTestId: hit?.getAttribute('data-testid') || '',
        hitText: (hit?.getAttribute('aria-label') || hit?.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80),
        passed
      }];
    });
  }, clickableSelector);
}

async function collectCardReadability(page, viewport) {
  if (!cardReadabilityViewportKeys.has(viewport.key)) {
    return { required: false, status: 'NOT_REQUIRED', cards: [], failures: [] };
  }

  return page.evaluate((samples) => {
    const existingProbe = document.querySelector('[data-testid="cardFaceReadabilityProbe"]');
    existingProbe?.remove();

    const probe = document.createElement('section');
    probe.dataset.testid = 'cardFaceReadabilityProbe';
    probe.setAttribute('aria-hidden', 'true');
    Object.assign(probe.style, {
      position: 'fixed',
      top: '74px',
      left: '50%',
      zIndex: '10000',
      display: 'grid',
      gridTemplateColumns: 'repeat(4, minmax(0, 1fr))',
      gap: '8px',
      width: 'min(340px, calc(100vw - 16px))',
      padding: '8px',
      borderRadius: '10px',
      background: 'rgba(4, 15, 22, 0.94)',
      boxSizing: 'border-box',
      pointerEvents: 'none'
    });

    const rankLabels = { a: 'A', k: 'K', q: 'Q', j: 'J', '10': '10' };
    const suitSymbols = { s: '♠', h: '♥', d: '♦', c: '♣' };
    for (const size of ['community', 'hero']) {
      for (const code of samples) {
        const suit = code.slice(-1);
        const rank = code.slice(0, -1);
        const card = document.createElement('span');
        card.className = `${size === 'community' ? 'board-card' : 'hole-card'} table-card-face${size === 'hero' ? ' hero-card-face' : ''}${suit === 'h' || suit === 'd' ? ' red' : ''}`;
        card.dataset.cardCode = code;
        card.dataset.cardSize = size;
        card.style.justifySelf = 'center';

        const rankElement = document.createElement('span');
        rankElement.className = 'card-rank';
        rankElement.textContent = rankLabels[rank] || rank.toUpperCase();
        const suitElement = document.createElement('span');
        suitElement.className = 'card-suit';
        suitElement.textContent = suitSymbols[suit] || '';
        card.append(rankElement, suitElement);
        probe.append(card);
      }
    }
    (document.querySelector('.room-mobile-screen') || document.body).append(probe);

    const rounded = (value) => Math.round(value * 100) / 100;
    const parseColor = (value) => {
      const channels = value.match(/[\d.]+/g)?.map(Number) ?? [];
      if (channels.length < 3) return null;
      return { red: channels[0], green: channels[1], blue: channels[2], alpha: channels[3] ?? 1 };
    };
    const luminance = (color) => {
      const channels = [color.red, color.green, color.blue].map((channel) => {
        const normalized = channel / 255;
        return normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
      });
      return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
    };
    const contrastRatio = (foreground, background) => {
      const foregroundLuminance = luminance(foreground);
      const backgroundLuminance = luminance(background);
      return (Math.max(foregroundLuminance, backgroundLuminance) + 0.05)
        / (Math.min(foregroundLuminance, backgroundLuminance) + 0.05);
    };
    const fitsWithin = (child, parent) => (
      child.left >= parent.left - 0.5
      && child.top >= parent.top - 0.5
      && child.right <= parent.right + 0.5
      && child.bottom <= parent.bottom + 0.5
    );

    const cards = [...probe.querySelectorAll('.table-card-face')].map((card) => {
      const cardRectangle = card.getBoundingClientRect();
      const rankElement = card.querySelector('.card-rank');
      const suitElement = card.querySelector('.card-suit');
      const rankRectangle = rankElement.getBoundingClientRect();
      const suitRectangle = suitElement.getBoundingClientRect();
      const cardStyle = window.getComputedStyle(card);
      const rankStyle = window.getComputedStyle(rankElement);
      const suitStyle = window.getComputedStyle(suitElement);
      const foreground = parseColor(cardStyle.color);
      const background = parseColor(cardStyle.backgroundColor);
      const ratio = foreground && background && background.alpha === 1
        ? rounded(contrastRatio(foreground, background))
        : 0;
      const rankFits = fitsWithin(rankRectangle, cardRectangle);
      const suitFits = fitsWithin(suitRectangle, cardRectangle);
      const rankFontSize = Number.parseFloat(rankStyle.fontSize);
      const suitFontSize = Number.parseFloat(suitStyle.fontSize);
      const expectedRank = card.dataset.cardCode.slice(0, -1).toUpperCase();
      const expectedSuit = suitSymbols[card.dataset.cardCode.slice(-1)];
      const textMatches = rankElement.textContent === expectedRank && suitElement.textContent === expectedSuit;
      const minimumWidth = card.dataset.cardSize === 'hero' ? 52 : 42;
      const passed = textMatches
        && rankFits
        && suitFits
        && ratio >= 4.5
        && rankFontSize >= 16
        && suitFontSize >= 32
        && cardRectangle.width >= minimumWidth
        && cardRectangle.height >= 58;

      return {
        code: card.dataset.cardCode,
        size: card.dataset.cardSize,
        rank: rankElement.textContent,
        suit: suitElement.textContent,
        rectangle: {
          width: rounded(cardRectangle.width),
          height: rounded(cardRectangle.height)
        },
        rankFontSize: rounded(rankFontSize),
        suitFontSize: rounded(suitFontSize),
        foreground: cardStyle.color,
        background: cardStyle.backgroundColor,
        contrastRatio: ratio,
        rankFits,
        suitFits,
        textMatches,
        passed
      };
    });
    const failures = cards.filter((card) => !card.passed);
    return {
      required: true,
      status: failures.length ? 'FAIL' : 'PASS',
      cards,
      failures
    };
  }, cardReadabilitySamples);
}

function intersectionArea(left, right) {
  const width = Math.min(left.right, right.right) - Math.max(left.left, right.left);
  const height = Math.min(left.bottom, right.bottom) - Math.max(left.top, right.top);
  return width > 0 && height > 0 ? Math.round(width * height * 100) / 100 : 0;
}

function collectBlockingOverlaps(rectangles) {
  const overlaps = [];
  for (const [leftName, rightName] of blockingOverlapPairs) {
    const leftRectangles = rectangles[leftName] ?? [];
    const rightRectangles = rectangles[rightName] ?? [];
    for (let leftIndex = 0; leftIndex < leftRectangles.length; leftIndex += 1) {
      for (let rightIndex = 0; rightIndex < rightRectangles.length; rightIndex += 1) {
        const area = intersectionArea(leftRectangles[leftIndex], rightRectangles[rightIndex]);
        if (area > 1) {
          overlaps.push({ left: leftName, leftIndex, right: rightName, rightIndex, area });
        }
      }
    }
  }

  const seatNodes = rectangles.seatNodes ?? [];
  for (let leftIndex = 0; leftIndex < seatNodes.length; leftIndex += 1) {
    for (let rightIndex = leftIndex + 1; rightIndex < seatNodes.length; rightIndex += 1) {
      const area = intersectionArea(seatNodes[leftIndex], seatNodes[rightIndex]);
      if (area > 1) {
        overlaps.push({ left: 'seatNodes', leftIndex, right: 'seatNodes', rightIndex, area });
      }
    }
  }
  return overlaps;
}

function collectViewportOverflows(rectangles, viewport) {
  const tolerance = 1;
  return [...viewportBoundTargets].flatMap((name) => (rectangles[name] ?? []).flatMap((rectangle, index) => {
    const edges = [];
    if (rectangle.left < -tolerance) edges.push('left');
    if (rectangle.top < -tolerance) edges.push('top');
    if (rectangle.right > viewport.width + tolerance) edges.push('right');
    if (rectangle.bottom > viewport.height + tolerance) edges.push('bottom');
    return edges.length ? [{ name, index, edges, rectangle }] : [];
  }));
}

async function runViewport(browser, options, viewport) {
  const consoleErrors = [];
  const pageErrors = [];
  const contextOptions = {
    viewport: { width: viewport.width, height: viewport.height },
    locale: 'zh-CN'
  };
  if (options.storageState) contextOptions.storageState = options.storageState;

  const context = await browser.newContext(contextOptions);
  const page = await context.newPage();
  if (options.token) {
    await page.addInitScript((token) => {
      window.sessionStorage.setItem('depu.auth.token', token);
    }, options.token);
  }
  page.on('console', (message) => {
    if (message.type() === 'error') {
      consoleErrors.push({ text: message.text(), location: message.location() });
    }
  });
  page.on('pageerror', (error) => {
    pageErrors.push({ name: error.name, message: error.message, stack: error.stack || '' });
  });

  let navigationError = '';
  try {
    await page.goto(options.url, { waitUntil: 'domcontentloaded', timeout: 30_000 });
    await page.waitForTimeout(options.waitMs);
  } catch (error) {
    navigationError = error instanceof Error ? error.message : String(error);
  }

  const rectangles = await collectRectangles(page);
  const clickHits = await collectClickHits(page);
  const blockingOverlaps = collectBlockingOverlaps(rectangles);
  const viewportOverflows = collectViewportOverflows(rectangles, viewport);
  const cardReadability = await collectCardReadability(page, viewport);
  const missingRequiredRectangles = rectangleTargets
    .filter((target) => target.required && !rectangles[target.name]?.length)
    .map((target) => target.name);
  const missedClicks = clickHits.filter((hit) => !hit.passed);
  const screenshot = resolve(options.outputDir, `${viewport.name}.png`);
  await page.screenshot({ path: screenshot, fullPage: true });

  const result = {
    name: viewport.name,
    key: viewport.key,
    viewport: { width: viewport.width, height: viewport.height },
    finalUrl: page.url(),
    title: await page.title(),
    navigationError,
    missingRequiredRectangles,
    rectangles,
    clickHits,
    blockingOverlaps,
    viewportOverflows,
    cardReadability,
    consoleErrors,
    pageErrors,
    screenshot,
    status: navigationError
      || missingRequiredRectangles.length
      || missedClicks.length
      || blockingOverlaps.length
      || viewportOverflows.length
      || cardReadability.status === 'FAIL'
      || consoleErrors.length
      || pageErrors.length
      ? 'FAIL'
      : 'PASS'
  };

  await context.close();
  return result;
}

async function loadPlaywright() {
  try {
    return await import('playwright');
  } catch (error) {
    throw new Error(
      `无法加载 Playwright。请先使用 Node 20 执行 npm install，并执行 npx playwright install chromium。原始错误：${error instanceof Error ? error.message : String(error)}`
    );
  }
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    process.stdout.write(`${usage()}\n`);
    return;
  }
  if (!options.url) throw new Error(`必须提供 --url。\n\n${usage()}`);
  if (options.storageState && !existsSync(options.storageState)) {
    throw new Error(`storageState 文件不存在：${options.storageState}`);
  }

  options.outputDir = resolve(options.outputDir);
  await mkdir(options.outputDir, { recursive: true });
  const { chromium } = await loadPlaywright();
  const browser = await chromium.launch({ headless: !options.headed });
  const viewports = [];

  try {
    for (const viewport of viewportMatrix) {
      process.stdout.write(`[table-hardening] ${viewport.key}\n`);
      viewports.push(await runViewport(browser, options, viewport));
    }
  } finally {
    await browser.close();
  }

  const failed = viewports.filter((viewport) => viewport.status === 'FAIL');
  const report = {
    generatedAt: new Date().toISOString(),
    url: options.url,
    outputDir: options.outputDir,
    viewports,
    summary: {
      total: viewports.length,
      passed: viewports.length - failed.length,
      failed: failed.length,
      status: failed.length ? 'FAIL' : 'PASS'
    }
  };
  const reportPath = resolve(options.outputDir, 'report.json');
  await writeFile(reportPath, `${JSON.stringify(report, null, 2)}\n`);
  process.stdout.write(`[table-hardening] report ${reportPath}\n`);
  if (failed.length) process.exitCode = 1;
}

main().catch((error) => {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
});
