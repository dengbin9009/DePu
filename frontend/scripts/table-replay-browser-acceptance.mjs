import { spawn } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = resolve(frontendRoot, '..');
const backendRoot = resolve(repoRoot, 'backend');
const outputDir = resolve(
  process.env.DEPU_REPLAY_ACCEPTANCE_OUTPUT || resolve(frontendRoot, '.artifacts/table-replay-browser'),
  new Date().toISOString().replaceAll(':', '-'),
);
const backendPort = Number(process.env.DEPU_REPLAY_BACKEND_PORT || 15174);
const frontendPort = Number(process.env.DEPU_REPLAY_FRONTEND_PORT || 15175);
const backendURL = `http://127.0.0.1:${backendPort}`;
const frontendURL = `http://127.0.0.1:${frontendPort}`;
const databaseDSN = process.env.DEPU_DSN || '';
const databaseName = process.env.DEPU_TEST_DATABASE || '';
const children = [];

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

async function currentHand(token, roomId) {
  try {
    return await api(`/api/rooms/${roomId}/current-hand`, { token });
  } catch (error) {
    if (error.status === 404) return null;
    throw error;
  }
}

async function waitForHandChange(token, roomId, previousVersion, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const hand = await currentHand(token, roomId);
    if (!hand || hand.version !== previousVersion) return hand;
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 100));
  }
  throw new Error(`hand version did not advance from ${previousVersion}`);
}

async function createAuthenticatedPage(browser, token, roomId) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  await context.addInitScript((authToken) => {
    window.sessionStorage.setItem('depu.auth.token', authToken);
  }, token);
  const page = await context.newPage();
  await page.goto(`${frontendURL}/room/${roomId}`);
  await page.getByRole('button', { name: '打开牌谱' }).waitFor();
  return { context, page };
}

function mysqlOptions() {
  const match = /^(?:([^:@/]+)(?::([^@]*))?@)?tcp\(([^:)]+)(?::(\d+))?\)\//.exec(databaseDSN);
  assert(match, 'DEPU_DSN must use the Go MySQL tcp(...) format');
  const [, user = 'root', password = '', host, port = '3306'] = match;
  return { user, password, host, port };
}

async function queryRows(sql) {
  const { user, password, host, port } = mysqlOptions();
  return new Promise((resolvePromise, rejectPromise) => {
    const child = spawn('mysql', [
      '--batch',
      '--raw',
      '--skip-column-names',
      '--host', host,
      '--port', port,
      '--user', user,
      '--database', databaseName,
      '--execute', sql,
    ], {
      env: { ...process.env, ...(password ? { MYSQL_PWD: password } : {}) },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (chunk) => { stdout += chunk; });
    child.stderr.on('data', (chunk) => { stderr += chunk; });
    child.on('error', rejectPromise);
    child.on('close', (code) => {
      if (code !== 0) {
        rejectPromise(new Error(`mysql query failed: ${stderr.trim()}`));
        return;
      }
      resolvePromise(stdout.trim() ? stdout.trim().split('\n').map((line) => line.split('\t')) : []);
    });
  });
}

function canonicalJSON(value) {
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(',')}]`;
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonicalJSON(value[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

async function inspectRenderedCards(locator) {
  return locator.evaluateAll((nodes) => nodes.map((node) => {
    const bounds = node.getBoundingClientRect();
    const rank = node.querySelector('.card-rank')?.textContent?.trim() || '';
    const suit = node.querySelector('.card-suit')?.textContent?.trim() || '';
    return {
      card: node.getAttribute('data-card') || '',
      ariaLabel: node.getAttribute('aria-label') || '',
      rank,
      suit,
      width: Math.round(bounds.width),
      height: Math.round(bounds.height),
    };
  }));
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
  await Promise.all([
    waitForURL(`${backendURL}/health`),
    waitForURL(`${frontendURL}/`),
  ]);

  const runId = `${Date.now()}_${Math.floor(Math.random() * 100_000)}`;
  const owner = await register(`replay_owner_${runId}`, `牌谱房主${runId.slice(-5)}`);
  const player = await register(`replay_player_${runId}`, `牌谱玩家${runId.slice(-5)}`);
  const room = await api('/api/rooms', {
    token: owner.token,
    method: 'POST',
    body: { ruleSetId: 'long-holdem', seatCount: 2, minPlayersToStart: 2 },
  });
  await api('/api/rooms/join', {
    token: player.token,
    method: 'POST',
    body: { inviteCode: room.inviteCode },
  });
  await api(`/api/rooms/${room.id}/seats/1`, { token: owner.token, method: 'POST', body: { buyInChips: 1000 } });
  await api(`/api/rooms/${room.id}/seats/2`, { token: player.token, method: 'POST', body: { buyInChips: 1000 } });

  const browser = await chromium.launch({ headless: process.env.DEPU_REPLAY_HEADED !== '1' });
  const browserErrors = [];
  try {
    const ownerBrowser = await createAuthenticatedPage(browser, owner.token, room.id);
    const playerBrowser = await createAuthenticatedPage(browser, player.token, room.id);
    for (const [label, page] of [['owner', ownerBrowser.page], ['player', playerBrowser.page]]) {
      page.on('console', (message) => {
        if (message.type() === 'error') browserErrors.push(`${label} console: ${message.text()}`);
      });
      page.on('pageerror', (error) => browserErrors.push(`${label} pageerror: ${error.message}`));
    }

    const startButton = ownerBrowser.page.getByRole('button', { name: '开 始' });
    await startButton.waitFor({ state: 'visible' });
    await startButton.click();
    await ownerBrowser.page.getByText('牌局已开始').waitFor();

    const pagesBySeat = new Map([
      [1, { page: ownerBrowser.page, token: owner.token }],
      [2, { page: playerBrowser.page, token: player.token }],
    ]);
    let browserActionCount = 0;
    while (true) {
      const hand = await currentHand(owner.token, room.id);
      if (!hand) break;
      const actor = pagesBySeat.get(hand.currentSeat);
      assert(actor, `no browser page for acting seat ${hand.currentSeat}`);
      await actor.page.getByText('轮到我').waitFor({ state: 'visible' });
      const allInButton = actor.page.getByRole('button', { name: '全下', exact: true });
      await allInButton.waitFor({ state: 'visible' });
      await allInButton.click();
      browserActionCount += 1;
      await waitForHandChange(actor.token, room.id, hand.version);
      if (browserActionCount > 4) throw new Error('hand did not settle after expected all-in actions');
    }
    assert(browserActionCount >= 2, `expected at least two browser actions, got ${browserActionCount}`);

    await ownerBrowser.page.screenshot({ path: resolve(outputDir, 'settled-table-history.png'), fullPage: true });

    await ownerBrowser.page.getByRole('button', { name: '打开牌谱' }).click();
    const replayPanel = ownerBrowser.page.locator('.table-replay-panel');
    await replayPanel.getByText('牌谱回顾 - 1').waitFor();
    await replayPanel.getByText(/第1手 ·/).waitFor();
    await replayPanel.locator('.replay-step-detail').waitFor();
    const historyCards = await inspectRenderedCards(
      replayPanel.locator('.replay-hand-list li:first-child .replay-hand-board .replay-card-face'),
    );
    const replayStepTexts = [];
    while (true) {
      const stepText = (await replayPanel.locator('.replay-step-detail').innerText()).trim();
      if (replayStepTexts.at(-1) !== stepText) replayStepTexts.push(stepText);
      const nextActionButton = replayPanel.getByRole('button', { name: '下一动作' });
      if (await nextActionButton.isDisabled()) break;
      await nextActionButton.click();
      await ownerBrowser.page.waitForFunction(
        ({ selector, previousText }) => document.querySelector(selector)?.textContent?.trim() !== previousText,
        { selector: '.replay-step-detail', previousText: stepText },
      );
    }
    assert(replayStepTexts.length >= 3, `expected replay navigation across concrete steps, got ${replayStepTexts.length}`);
    const finalBoardCards = await inspectRenderedCards(replayPanel.locator('.replay-step-board .replay-card-face'));
    const finalHoleCards = await inspectRenderedCards(replayPanel.locator('.replay-player-hole-cards .replay-card-face'));
    const finalStepText = replayStepTexts.at(-1);
    await ownerBrowser.page.screenshot({ path: resolve(outputDir, 'replay-final-step.png'), fullPage: true });

    const history = await api(`/api/rooms/${room.id}/hands/recent`, { token: owner.token });
    assert(history.items.length === 1, `expected one archived hand, got ${history.items.length}`);
    const archivedHand = history.items[0];
    const replay = await api(`/api/rooms/${room.id}/hands/${archivedHand.handId}/replay`, { token: owner.token });
    const resultRows = await queryRows(`select id, game_id, hand_no, winner_summary, pot_summary, board_cards_json, total_pot from hand_results where room_id = '${room.id}' order by hand_no desc limit 1`);
    const participantRows = await queryRows(`select seat_no, nickname_snapshot, profit, result_type, hole_cards_json, best_cards_json, hand_class, hand_committed, award_amount from hand_participants where hand_id = '${archivedHand.handId}' order by seat_no`);
    const actionRows = await queryRows(`select count(*), coalesce(max(seq), 0) from actions where game_id = '${replay.gameId}'`);
    assert(resultRows.length === 1, 'hand_results archive row is missing');
    assert(participantRows.length === 2, `expected two hand_participants rows, got ${participantRows.length}`);
    assert(actionRows.length === 1, 'actions archive summary is missing');

    const [handId, gameId, handNo, winnerSummary, potSummary, boardCardsJSON, totalPot] = resultRows[0];
    const databaseParticipants = participantRows.map((row) => ({
      seatNo: Number(row[0]),
      nickname: row[1],
      profit: Number(row[2]),
      resultType: row[3],
      holeCards: JSON.parse(row[4]),
      bestCards: JSON.parse(row[5]),
      handClass: row[6],
      handCommitted: Number(row[7]),
      awardAmount: Number(row[8]),
    }));
    const databaseBoardCards = JSON.parse(boardCardsJSON);
    const finalReplayStep = replay.steps.at(-1);
    const firstPlayerActionStep = replay.steps.find((step) => step.action?.seatNo > 0 && ['all_in', 'fold', 'check', 'call', 'bet', 'raise'].includes(step.action.type));
    const actionCount = Number(actionRows[0][0]);
    const maxActionSeq = Number(actionRows[0][1]);
    const archiveConsistency = {
      handId: handId === archivedHand.handId && handId === replay.handId,
      gameId: gameId === replay.gameId,
      handNo: Number(handNo) === archivedHand.handNo,
      winnerSummary: winnerSummary === archivedHand.winnerSummary,
      potSummary: potSummary === archivedHand.potSummary,
      totalPot: Number(totalPot) === archivedHand.totalPot && Number(totalPot) === finalReplayStep.pot,
      boardCards: canonicalJSON(databaseBoardCards) === canonicalJSON(archivedHand.boardCards)
        && canonicalJSON(databaseBoardCards) === canonicalJSON(finalReplayStep.boardCards),
      participants: canonicalJSON(databaseParticipants.map(({ seatNo, holeCards }) => ({ seatNo, holeCards })))
        === canonicalJSON(finalReplayStep.players.map(({ seatNo, holeCards }) => ({ seatNo, holeCards: holeCards || [] }))),
      replaySteps: replay.steps.length === actionCount + 1 && finalReplayStep.seq === maxActionSeq,
      historicalSteps: firstPlayerActionStep?.stage === 'preflop'
        && firstPlayerActionStep.boardCards.length === 0
        && firstPlayerActionStep.players.every((replayPlayer) => !replayPlayer.holeCards?.length)
        && finalReplayStep.stage === 'showdown',
      browserHistoryCards: canonicalJSON(historyCards.map(({ card }) => card)) === canonicalJSON(databaseBoardCards),
      browserFinalStep: canonicalJSON(finalBoardCards.map(({ card }) => card)) === canonicalJSON(databaseBoardCards)
        && canonicalJSON(finalHoleCards.map(({ card }) => card))
          === canonicalJSON(databaseParticipants.flatMap((participant) => participant.holeCards)),
      browserCardVisuals: [...historyCards, ...finalBoardCards, ...finalHoleCards].every((card) => (
        card.card && card.ariaLabel && card.rank && ['♠', '♥', '♦', '♣'].includes(card.suit)
          && card.width >= 40 && card.height >= 56
      )),
      rawCardCodesAbsent: [...databaseBoardCards, ...databaseParticipants.flatMap((participant) => participant.holeCards)]
        .every((card) => !finalStepText.includes(card)),
    };
    const failedConsistencyChecks = Object.entries(archiveConsistency).filter(([, passed]) => !passed).map(([name]) => name);
    assert(failedConsistencyChecks.length === 0, `archive consistency failed: ${failedConsistencyChecks.join(', ')}`);
    const unexpectedBrowserErrors = browserErrors.filter((message) => (
      !message.includes('Failed to load resource: the server responded with a status of 404')
    ));
    assert(unexpectedBrowserErrors.length === 0, `browser errors detected: ${unexpectedBrowserErrors.join(' | ')}`);

    const report = {
      status: 'passed',
      database: databaseName,
      roomId: room.id,
      handId: archivedHand.handId,
      browserActionCount,
      replayStepCount: replay.steps.length,
      replayStepTexts,
      renderedCards: {
        history: historyCards,
        finalBoard: finalBoardCards,
        finalHoleCards,
      },
      archiveConsistency,
      browserErrors,
      artifacts: ['settled-table-history.png', 'replay-final-step.png'],
    };
    await writeFile(resolve(outputDir, 'replay-acceptance-report.json'), `${JSON.stringify(report, null, 2)}\n`);
    process.stdout.write(`table replay browser acceptance passed: ${resolve(outputDir, 'replay-acceptance-report.json')}\n`);
    await Promise.all([ownerBrowser.context.close(), playerBrowser.context.close()]);
  } finally {
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
  await writeFile(resolve(outputDir, 'replay-acceptance-report.json'), `${JSON.stringify({
    status: 'failed',
    database: databaseName,
    error: error instanceof Error ? error.stack || error.message : String(error),
  }, null, 2)}\n`);
  console.error(error);
  process.exitCode = 1;
} finally {
  stopProcesses();
}
