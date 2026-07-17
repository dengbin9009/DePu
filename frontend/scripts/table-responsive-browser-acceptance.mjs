import { spawn } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = resolve(frontendRoot, '..');
const backendRoot = resolve(repoRoot, 'backend');
const outputDir = resolve(
  process.env.DEPU_RESPONSIVE_ACCEPTANCE_OUTPUT || resolve(frontendRoot, '.artifacts/table-responsive-browser'),
  new Date().toISOString().replaceAll(':', '-'),
);
const backendPort = Number(process.env.DEPU_RESPONSIVE_BACKEND_PORT || 15180);
const frontendPort = Number(process.env.DEPU_RESPONSIVE_FRONTEND_PORT || 15181);
const backendURL = `http://127.0.0.1:${backendPort}`;
const frontendURL = `http://127.0.0.1:${frontendPort}`;
const databaseDSN = process.env.DEPU_DSN || '';
const databaseName = process.env.DEPU_TEST_DATABASE || '';
const children = [];
const viewportMatrix = [
  { key: '360x640', width: 360, height: 640 },
  { key: '390x844', width: 390, height: 844 },
  { key: '430x932', width: 430, height: 932 },
  { key: '768x1024', width: 768, height: 1024 },
  { key: '1280x720', width: 1280, height: 720 },
];
const scenarioNames = [
  'waiting-nine-seats',
  'buy-in-modal',
  'playing-amount-controls',
  'playing-five-board-cards',
  'drawer-settings',
  'drawer-chat',
  'drawer-score',
  'drawer-replay',
];
const rectangleTargets = [
  { name: 'screen', selector: '.room-mobile-screen' },
  { name: 'stage', selector: '.room-stage' },
  { name: 'topbar', selector: '.table-topbar-compact' },
  { name: 'center', selector: '.table-center-stack-compact' },
  { name: 'seatNodes', selector: '.seat-ring-casino .casino-seat-node' },
  { name: 'hero', selector: '.hero-panel' },
  { name: 'actions', selector: '.hero-actions-safe, .owner-action-row-dock' },
  { name: 'amountControl', selector: '.action-amount-control' },
  { name: 'boardCards', selector: '.table-center-stack-compact .board-card' },
  { name: 'toolbar', selector: '.mock-bottom-toolbar-safe' },
  { name: 'drawer', selector: '.table-drawer' },
  { name: 'modal', selector: '.buy-in-modal' },
];
const viewportBoundTargets = new Set([
  'screen', 'stage', 'topbar', 'center', 'seatNodes', 'hero', 'actions', 'amountControl', 'boardCards', 'toolbar', 'drawer', 'modal',
]);
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
  ['actions', 'toolbar'],
];
const clickableSelector = [
  'button:not([disabled])',
  'a[href]',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[role="button"]:not([aria-disabled="true"])',
].join(', ');

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function startProcess(command, args, options) {
  const child = spawn(command, args, {
    ...options,
    detached: process.platform !== 'win32',
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  children.push(child);
  child.stdout.on('data', (chunk) => process.stdout.write(`[${options.label}] ${chunk}`));
  child.stderr.on('data', (chunk) => process.stderr.write(`[${options.label}] ${chunk}`));
  return child;
}

function stopProcesses() {
  for (const child of children.reverse()) {
    if (!child.pid || child.exitCode !== null) continue;
    try {
      if (process.platform === 'win32') child.kill('SIGTERM');
      else process.kill(-child.pid, 'SIGTERM');
    } catch {
      child.kill('SIGTERM');
    }
  }
}

async function waitForURL(url, timeoutMs = 30_000) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
      lastError = new Error(`${url} returned ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 200));
  }
  throw new Error(`service did not become ready: ${lastError instanceof Error ? lastError.message : url}`);
}

async function api(path, { token = '', method = 'GET', body } = {}) {
  const response = await fetch(`${backendURL}${path}`, {
    method,
    headers: {
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    const error = new Error(payload.message || payload.code || `${method} ${path} returned ${response.status}`);
    error.status = response.status;
    throw error;
  }
  return payload;
}

async function register(username, nickname) {
  return api('/api/auth/register', {
    method: 'POST',
    body: { username, password: 'password1', nickname },
  });
}

async function createRoom(token, runId, seatCount, name) {
  return api('/api/rooms', {
    token,
    method: 'POST',
    body: {
      ruleSetId: 'long-holdem',
      name: `${name}${runId.slice(-5)}`,
      seatCount,
      minPlayersToStart: 2,
      minBuyIn: 1000,
      maxBuyIn: 3000,
    },
  });
}

async function createAuthenticatedPage(browser, token, path, errors) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  await context.addInitScript((authToken) => {
    window.sessionStorage.setItem('depu.auth.token', authToken);
  }, token);
  const page = await context.newPage();
  page.on('console', (message) => {
    if (message.type() === 'error') errors.push(`console: ${message.text()}`);
  });
  page.on('pageerror', (error) => errors.push(`pageerror: ${error.message}`));
  await page.goto(`${frontendURL}${path}`);
  await page.getByRole('button', { name: '邀请好友' }).waitFor();
  return { context, page, token };
}

async function currentHand(token, roomId) {
  try {
    return await api(`/api/rooms/${roomId}/current-hand`, { token });
  } catch (error) {
    if (error.status === 404) return null;
    throw error;
  }
}

async function waitForCurrentHand(token, roomId, predicate, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const hand = await currentHand(token, roomId);
    if (predicate(hand)) return hand;
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 100));
  }
  throw new Error('current hand did not reach the expected state');
}

async function sendSocketAction(page, token, roomId, action) {
  return page.evaluate(async ({ socketURL, authToken, targetRoomId, actionType }) => {
    return new Promise((resolvePromise, rejectPromise) => {
      const socket = new WebSocket(`${socketURL}/api/socket?token=${encodeURIComponent(authToken)}`);
      const subscribeRequestId = `responsive_subscribe_${Date.now()}`;
      const actionRequestId = `responsive_action_${Date.now()}`;
      const timeout = window.setTimeout(() => {
        socket.close();
        rejectPromise(new Error('responsive socket action timed out'));
      }, 5_000);
      socket.addEventListener('message', (event) => {
        const message = JSON.parse(String(event.data));
        if (message.type === 'connection.ready') {
          socket.send(JSON.stringify({ type: 'room.subscribe', requestId: subscribeRequestId, roomId: targetRoomId, payload: {} }));
          return;
        }
        if (message.type === 'ack' && message.requestId === subscribeRequestId) {
          socket.send(JSON.stringify({ type: 'room.action', requestId: actionRequestId, roomId: targetRoomId, payload: { action: actionType, amount: 0 } }));
          return;
        }
        if ((message.type === 'ack' || message.type === 'error') && message.requestId === actionRequestId) {
          window.clearTimeout(timeout);
          socket.close();
          if (message.type === 'error') {
            rejectPromise(new Error(message.payload?.message || message.payload?.code || 'room.action failed'));
            return;
          }
          resolvePromise();
        }
      });
      socket.addEventListener('error', () => {
        window.clearTimeout(timeout);
        rejectPromise(new Error('responsive socket connection failed'));
      });
    });
  }, {
    socketURL: backendURL.replace('http://', 'ws://'),
    authToken: token,
    targetRoomId: roomId,
    actionType: action,
  });
}

async function collectRectangles(page) {
  return page.evaluate((targets) => {
    const round = (value) => Math.round(value * 100) / 100;
    return Object.fromEntries(targets.map(({ name, selector }) => [name, [...document.querySelectorAll(selector)].flatMap((element) => {
      const style = window.getComputedStyle(element);
      const rectangle = element.getBoundingClientRect();
      if (style.display === 'none' || style.visibility === 'hidden' || rectangle.width <= 0 || rectangle.height <= 0) return [];
      return [{
        x: round(rectangle.x),
        y: round(rectangle.y),
        width: round(rectangle.width),
        height: round(rectangle.height),
        top: round(rectangle.top),
        right: round(rectangle.right),
        bottom: round(rectangle.bottom),
        left: round(rectangle.left),
      }];
    })]));
  }, rectangleTargets);
}

async function collectClickHits(page) {
  return page.evaluate((selector) => {
    const activeRoot = document.querySelector('.modal-backdrop .buy-in-modal, .table-drawer-backdrop .table-drawer') || document.querySelector('.room-mobile-screen');
    if (!activeRoot) return [];
    return [...activeRoot.querySelectorAll(selector)].flatMap((candidate, index) => {
      const style = window.getComputedStyle(candidate);
      const rectangle = candidate.getBoundingClientRect();
      const left = Math.max(0, rectangle.left);
      const right = Math.min(window.innerWidth, rectangle.right);
      const top = Math.max(0, rectangle.top);
      const bottom = Math.min(window.innerHeight, rectangle.bottom);
      if (style.display === 'none' || style.visibility === 'hidden' || style.pointerEvents === 'none' || right <= left || bottom <= top) return [];
      const x = Math.round((left + right) / 2);
      const y = Math.round((top + bottom) / 2);
      const hit = document.elementFromPoint(x, y);
      const text = (candidate.getAttribute('aria-label') || candidate.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80);
      return [{
        index,
        tag: candidate.tagName.toLowerCase(),
        text,
        point: { x, y },
        hitText: (hit?.getAttribute('aria-label') || hit?.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80),
        passed: Boolean(hit && (hit === candidate || candidate.contains(hit))),
      }];
    });
  }, clickableSelector);
}

function intersectionArea(left, right) {
  return Math.max(0, Math.min(left.right, right.right) - Math.max(left.left, right.left))
    * Math.max(0, Math.min(left.bottom, right.bottom) - Math.max(left.top, right.top));
}

function collectBlockingOverlaps(rectangles, overlayOpen) {
  if (overlayOpen) return [];
  const overlaps = [];
  for (const [leftName, rightName] of blockingOverlapPairs) {
    for (const [leftIndex, left] of (rectangles[leftName] || []).entries()) {
      for (const [rightIndex, right] of (rectangles[rightName] || []).entries()) {
        const area = intersectionArea(left, right);
        if (area > 1) overlaps.push({ left: leftName, leftIndex, right: rightName, rightIndex, area });
      }
    }
  }
  const seatNodes = rectangles.seatNodes || [];
  for (let leftIndex = 0; leftIndex < seatNodes.length; leftIndex += 1) {
    for (let rightIndex = leftIndex + 1; rightIndex < seatNodes.length; rightIndex += 1) {
      const area = intersectionArea(seatNodes[leftIndex], seatNodes[rightIndex]);
      if (area > 1) overlaps.push({ left: 'seatNodes', leftIndex, right: 'seatNodes', rightIndex, area });
    }
  }
  return overlaps;
}

function collectViewportOverflows(rectangles, viewport) {
  const tolerance = 1;
  return [...viewportBoundTargets].flatMap((name) => (rectangles[name] || []).flatMap((rectangle, index) => {
    const edges = [];
    if (rectangle.left < -tolerance) edges.push('left');
    if (rectangle.top < -tolerance) edges.push('top');
    if (rectangle.right > viewport.width + tolerance) edges.push('right');
    if (rectangle.bottom > viewport.height + tolerance) edges.push('bottom');
    return edges.length ? [{ name, index, edges, rectangle }] : [];
  }));
}

function validateScenarioGeometry(scenario, rectangles) {
  const failures = [];
  if (scenario === 'waiting-nine-seats' && rectangles.seatNodes.length !== 9) failures.push(`seatNodes=${rectangles.seatNodes.length}`);
  if (scenario === 'buy-in-modal' && rectangles.modal.length !== 1) failures.push(`modal=${rectangles.modal.length}`);
  if (scenario === 'playing-amount-controls' && rectangles.amountControl.length !== 1) failures.push(`amountControl=${rectangles.amountControl.length}`);
  if (scenario === 'playing-five-board-cards' && rectangles.boardCards.length !== 5) failures.push(`boardCards=${rectangles.boardCards.length}`);
  if (scenario.startsWith('drawer-') && rectangles.drawer.length !== 1) failures.push(`drawer=${rectangles.drawer.length}`);
  return failures;
}

async function captureScenario(page, scenario, viewport) {
  await page.setViewportSize({ width: viewport.width, height: viewport.height });
  await page.waitForTimeout(150);
  const rectangles = await collectRectangles(page);
  const clickHits = await collectClickHits(page);
  const overlayOpen = scenario === 'buy-in-modal' || scenario.startsWith('drawer-');
  const blockingOverlaps = collectBlockingOverlaps(rectangles, overlayOpen);
  const viewportOverflows = collectViewportOverflows(rectangles, viewport);
  const geometryFailures = validateScenarioGeometry(scenario, rectangles);
  const missedClicks = clickHits.filter((hit) => !hit.passed);
  const screenshotName = `${viewport.key}-${scenario}.png`;
  await page.screenshot({ path: resolve(outputDir, screenshotName) });
  const status = blockingOverlaps.length || viewportOverflows.length || geometryFailures.length || missedClicks.length ? 'FAIL' : 'PASS';
  return {
    scenario,
    viewport,
    status,
    rectangles,
    clickHits,
    blockingOverlaps,
    viewportOverflows,
    geometryFailures,
    missedClicks,
    screenshot: screenshotName,
  };
}

async function captureMatrix(page, scenario) {
  const results = [];
  for (const viewport of viewportMatrix) {
    process.stdout.write(`[table-responsive] ${viewport.key} ${scenario}\n`);
    results.push(await captureScenario(page, scenario, viewport));
  }
  return results;
}

async function openDrawerMatrix(page, scenario, buttonName) {
  const results = [];
  for (const viewport of viewportMatrix) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.getByRole('button', { name: buttonName }).click();
    await page.locator('.table-drawer').waitFor();
    results.push(await captureScenario(page, scenario, viewport));
    await page.getByRole('button', { name: '关闭抽屉' }).click();
  }
  return results;
}

async function advanceToRiver(ownerBrowser, playerBrowser, roomId) {
  let hand = await currentHand(ownerBrowser.token, roomId);
  let actionCount = 0;
  while (hand && (hand.boardCards?.length || 0) < 5) {
    const actingBrowser = hand.currentSeat === 1 ? ownerBrowser : playerBrowser;
    const action = hand.availableActions.includes('check') ? 'check' : hand.availableActions.includes('call') ? 'call' : '';
    assert(action, `cannot advance board from ${hand.status}: ${hand.availableActions.join(',')}`);
    const previousVersion = hand.version;
    await sendSocketAction(actingBrowser.page, actingBrowser.token, roomId, action);
    hand = await waitForCurrentHand(ownerBrowser.token, roomId, (candidate) => Boolean(candidate && candidate.version !== previousVersion));
    actionCount += 1;
    assert(actionCount <= 10, `river advance exceeded expected actions: ${actionCount}`);
  }
  assert(hand?.boardCards?.length === 5, `board cards=${hand?.boardCards?.length || 0}`);
  return hand;
}

async function main() {
  assert(databaseDSN, 'DEPU_DSN is required; run through scripts/with-test-mysql.sh');
  assert(databaseName, 'DEPU_TEST_DATABASE is required; run through scripts/with-test-mysql.sh');
  await mkdir(outputDir, { recursive: true });

  startProcess('go', ['run', './cmd/depu-server'], {
    cwd: backendRoot,
    env: { ...process.env, DEPU_ADDR: `:${backendPort}` },
    label: 'backend',
  });
  startProcess('npm', ['run', 'dev', '--', '--host', '127.0.0.1', '--port', String(frontendPort), '--strictPort'], {
    cwd: frontendRoot,
    env: { ...process.env, DEPU_API_TARGET: backendURL },
    label: 'frontend',
  });
  await Promise.all([waitForURL(`${backendURL}/health`), waitForURL(`${frontendURL}/`)]);

  const runId = `${Date.now()}_${Math.floor(Math.random() * 100_000)}`;
  const waitingAccounts = [];
  for (let index = 0; index < 9; index += 1) {
    waitingAccounts.push(await register(`responsive_seat_${index}_${runId}`, `超长响应式玩家昵称${index + 1}_${runId.slice(-4)}`));
  }
  const waitingRoom = await createRoom(waitingAccounts[0].token, runId, 9, '九人响应式验收');
  for (let index = 0; index < waitingAccounts.length; index += 1) {
    const account = waitingAccounts[index];
    if (index > 0) await api('/api/rooms/join', { token: account.token, method: 'POST', body: { inviteCode: waitingRoom.inviteCode } });
    await api(`/api/rooms/${waitingRoom.id}/seats/${index + 1}`, { token: account.token, method: 'POST', body: { buyInChips: 1000 } });
  }

  const modalOwner = await register(`responsive_modal_${runId}`, `买入弹窗玩家_${runId.slice(-4)}`);
  const modalRoom = await createRoom(modalOwner.token, runId, 2, '买入弹窗验收');

  const playingOwner = await register(`responsive_owner_${runId}`, `行动房主_${runId.slice(-4)}`);
  const playingPlayer = await register(`responsive_player_${runId}`, `行动玩家_${runId.slice(-4)}`);
  const playingRoom = await createRoom(playingOwner.token, runId, 2, '行动响应式验收');
  await api('/api/rooms/join', { token: playingPlayer.token, method: 'POST', body: { inviteCode: playingRoom.inviteCode } });
  await api(`/api/rooms/${playingRoom.id}/seats/1`, { token: playingOwner.token, method: 'POST', body: { buyInChips: 3000 } });
  await api(`/api/rooms/${playingRoom.id}/seats/2`, { token: playingPlayer.token, method: 'POST', body: { buyInChips: 3000 } });

  const browserErrors = [];
  const browser = await chromium.launch({ headless: process.env.DEPU_RESPONSIVE_HEADED !== '1' });
  const contexts = [];
  try {
    const observerBrowser = await createAuthenticatedPage(browser, waitingAccounts[0].token, `/room/${waitingRoom.id}`, browserErrors);
    const modalBrowser = await createAuthenticatedPage(browser, modalOwner.token, `/room/${modalRoom.id}`, browserErrors);
    const ownerBrowser = await createAuthenticatedPage(browser, playingOwner.token, `/room/${playingRoom.id}`, browserErrors);
    const playerBrowser = await createAuthenticatedPage(browser, playingPlayer.token, `/room/${playingRoom.id}`, browserErrors);
    contexts.push(observerBrowser.context, modalBrowser.context, ownerBrowser.context, playerBrowser.context);

    await observerBrowser.page.getByText('玩家 9/9').waitFor();
    const results = await captureMatrix(observerBrowser.page, 'waiting-nine-seats');

    await modalBrowser.page.getByRole('button', { name: '坐下 座位 1' }).click();
    await modalBrowser.page.getByRole('dialog', { name: '补充记分牌' }).waitFor();
    results.push(...await captureMatrix(modalBrowser.page, 'buy-in-modal'));
    await modalBrowser.page.getByRole('button', { name: '取消' }).click();

    await ownerBrowser.page.getByRole('button', { name: '开 始' }).click();
    const startedHand = await waitForCurrentHand(playingOwner.token, playingRoom.id, (hand) => Boolean(hand?.handId));
    const amountBrowser = startedHand.currentSeat === 1 ? ownerBrowser : playerBrowser;
    await amountBrowser.page.locator('.action-amount-control').waitFor();
    results.push(...await captureMatrix(amountBrowser.page, 'playing-amount-controls'));

    const riverHand = await advanceToRiver(ownerBrowser, playerBrowser, playingRoom.id);
    const riverBrowser = riverHand.currentSeat === 1 ? ownerBrowser : playerBrowser;
    await riverBrowser.page.locator('.table-center-stack-compact .board-card').nth(4).waitFor();
    results.push(...await captureMatrix(riverBrowser.page, 'playing-five-board-cards'));

    results.push(...await openDrawerMatrix(observerBrowser.page, 'drawer-settings', '打开设置'));
    results.push(...await openDrawerMatrix(observerBrowser.page, 'drawer-chat', '打开聊天'));
    results.push(...await openDrawerMatrix(observerBrowser.page, 'drawer-score', '打开战绩'));
    results.push(...await openDrawerMatrix(observerBrowser.page, 'drawer-replay', '打开牌谱'));

    const unexpectedBrowserErrors = browserErrors.filter((message) => !message.includes('Failed to load resource: the server responded with a status of 404'));
    const failed = results.filter((result) => result.status === 'FAIL');
    const coveredScenarios = [...new Set(results.map((result) => result.scenario))];
    const report = {
      status: failed.length || unexpectedBrowserErrors.length ? 'failed' : 'passed',
      database: databaseName,
      rooms: { waiting: waitingRoom.id, modal: modalRoom.id, playing: playingRoom.id },
      viewports: viewportMatrix,
      scenarios: scenarioNames,
      summary: { total: results.length, passed: results.length - failed.length, failed: failed.length },
      browserErrors,
      results,
    };
    const reportPath = resolve(outputDir, 'responsive-acceptance-report.json');
    await writeFile(reportPath, `${JSON.stringify(report, null, 2)}\n`);
    assert(results.length === viewportMatrix.length * scenarioNames.length, `result count=${results.length}`);
    assert(scenarioNames.every((scenario) => coveredScenarios.includes(scenario)), `scenario coverage=${coveredScenarios.join(',')}`);
    assert(unexpectedBrowserErrors.length === 0, `browser errors: ${unexpectedBrowserErrors.join(' | ')}`);
    assert(failed.length === 0, `responsive failures: ${failed.map((result) => `${result.viewport.key}/${result.scenario}`).join(', ')}`);
    process.stdout.write(`table responsive browser acceptance passed: ${reportPath}\n`);
  } finally {
    await Promise.all(contexts.map((context) => context.close()));
    await browser.close();
  }
}

process.on('SIGINT', () => {
  stopProcesses();
  process.exit(130);
});
process.on('SIGTERM', () => {
  stopProcesses();
  process.exit(143);
});

try {
  await main();
} catch (error) {
  await mkdir(outputDir, { recursive: true });
  await writeFile(resolve(outputDir, 'responsive-acceptance-error.json'), `${JSON.stringify({
    status: 'failed',
    database: databaseName,
    error: error instanceof Error ? error.stack || error.message : String(error),
  }, null, 2)}\n`);
  console.error(error);
  process.exitCode = 1;
} finally {
  stopProcesses();
}
