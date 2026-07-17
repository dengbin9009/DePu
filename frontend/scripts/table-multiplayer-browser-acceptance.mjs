import { spawn } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = resolve(frontendRoot, '..');
const backendRoot = resolve(repoRoot, 'backend');
const outputDir = resolve(
  process.env.DEPU_MULTIPLAYER_ACCEPTANCE_OUTPUT || resolve(frontendRoot, '.artifacts/table-multiplayer-browser'),
  new Date().toISOString().replaceAll(':', '-'),
);
const backendPort = Number(process.env.DEPU_MULTIPLAYER_BACKEND_PORT || 15176);
const frontendPort = Number(process.env.DEPU_MULTIPLAYER_FRONTEND_PORT || 15177);
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
    error.code = payload.code || '';
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

function parseSocketPayload(payload) {
  if (typeof payload !== 'string') return null;
  try {
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function attachBrowserAudit(page, label) {
  const audit = {
    label,
    requests: [],
    socketFramesSent: [],
    socketFramesReceived: [],
    errors: [],
  };
  page.on('request', (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith('/api/')) {
      audit.requests.push({ method: request.method(), path: url.pathname, at: Date.now() });
    }
  });
  page.on('websocket', (socket) => {
    socket.on('framesent', (event) => {
      const message = parseSocketPayload(event.payload);
      if (message?.type) audit.socketFramesSent.push({
        type: message.type,
        roomId: message.roomId || '',
        handId: message.handId || message.payload?.hand?.handId || '',
        requestId: message.requestId || '',
        at: Date.now(),
      });
    });
    socket.on('framereceived', (event) => {
      const message = parseSocketPayload(event.payload);
      if (message?.type) audit.socketFramesReceived.push({
        type: message.type,
        roomId: message.roomId || '',
        handId: message.handId || message.payload?.hand?.handId || '',
        requestId: message.requestId || '',
        at: Date.now(),
      });
    });
  });
  page.on('console', (message) => {
    if (message.type() === 'error') audit.errors.push(`console: ${message.text()}`);
  });
  page.on('pageerror', (error) => audit.errors.push(`pageerror: ${error.message}`));
  return audit;
}

async function createAuthenticatedPage(browser, token, path, label) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  await context.addInitScript((authToken) => {
    window.sessionStorage.setItem('depu.auth.token', authToken);
  }, token);
  const page = await context.newPage();
  const audit = attachBrowserAudit(page, label);
  await page.goto(`${frontendURL}${path}`);
  return { context, page, audit };
}

function currentHandRequestCount(audit, roomId) {
  return audit.requests.filter((request) => request.path === `/api/rooms/${roomId}/current-hand`).length;
}

async function waitForCurrentHandRequestCountStable(audit, roomId, stableMs = 500, timeoutMs = 5_000) {
  const deadline = Date.now() + timeoutMs;
  let previousCount = currentHandRequestCount(audit, roomId);
  let stableSince = Date.now();
  while (Date.now() < deadline) {
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 100));
    const nextCount = currentHandRequestCount(audit, roomId);
    if (nextCount !== previousCount) {
      previousCount = nextCount;
      stableSince = Date.now();
      continue;
    }
    if (Date.now() - stableSince >= stableMs) return nextCount;
  }
  throw new Error(`${audit.label} current-hand request count did not stabilize`);
}

async function waitForAudit(audit, group, type, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (audit[group].some((entry) => entry.type === type)) return;
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 50));
  }
  throw new Error(`${audit.label} did not observe ${type} in ${group}`);
}

function auditEntries(audit, group, type) {
  return audit[group].filter((entry) => entry.type === type);
}

async function waitForAuditCount(audit, group, type, count, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const entries = auditEntries(audit, group, type);
    if (entries.length >= count) return entries;
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 50));
  }
  throw new Error(`${audit.label} observed ${auditEntries(audit, group, type).length}/${count} ${type} events in ${group}`);
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

async function waitForRoomHands(token, roomId, count, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const response = await api(`/api/rooms/${roomId}/hands/recent`, { token });
    if (response.items?.length >= count) return response.items;
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 100));
  }
  throw new Error(`room history did not reach ${count} hands`);
}

async function countdownValue(page) {
  const text = await page.getByText(/行动倒计时 \d+s/).innerText();
  const match = /行动倒计时 (\d+)s/.exec(text);
  assert(match, `invalid countdown text: ${text}`);
  return Number(match[1]);
}

async function openBuyInDialog(page, seatNo, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  const dialog = page.getByRole('dialog', { name: '补充记分牌' });
  while (Date.now() < deadline) {
    if (await dialog.isVisible().catch(() => false)) return dialog;
    const seatButton = page.getByRole('button', { name: `坐下 座位 ${seatNo}` });
    if (await seatButton.isVisible().catch(() => false)) {
      await seatButton.click();
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 100));
  }
  throw new Error(`buy-in dialog did not open for seat ${seatNo}`);
}

async function verifySettledHandDisplay(page, hand) {
  await page.getByRole('button', { name: '打开牌谱' }).click();
  const replayPanel = page.locator('.table-replay-panel');
  await replayPanel.getByRole('button', { name: `第${hand.handNo}手 · ${hand.winnerSummary}`, exact: true }).waitFor();
  await page.getByRole('button', { name: '关闭抽屉' }).click();
}

async function waitForCountdownDecrease(page, initialValue, timeoutMs = 4_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const value = await countdownValue(page);
    if (value < initialValue) return value;
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 250));
  }
  throw new Error(`行动倒计时 did not decrease from ${initialValue}`);
}

async function sendRawSocketAction(page, token, roomId, action) {
  return page.evaluate(async ({ socketURL, authToken, targetRoomId, actionType }) => {
    return new Promise((resolvePromise, rejectPromise) => {
      const socket = new WebSocket(`${socketURL}/api/socket?token=${encodeURIComponent(authToken)}`);
      const subscribeRequestId = `raw_subscribe_${Date.now()}`;
      const actionRequestId = `raw_action_${Date.now()}`;
      const timeout = window.setTimeout(() => {
        socket.close();
        rejectPromise(new Error('raw socket action timed out'));
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
          resolvePromise({ type: message.type, code: message.payload?.code || '', message: message.payload?.message || '' });
        }
      });
      socket.addEventListener('error', () => {
        window.clearTimeout(timeout);
        rejectPromise(new Error('raw socket connection failed'));
      });
    });
  }, {
    socketURL: backendURL.replace('http://', 'ws://'),
    authToken: token,
    targetRoomId: roomId,
    actionType: action,
  });
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
  const owner = await register(`multi_owner_${runId}`, `多账号房主${runId.slice(-5)}`);
  const player = await register(`multi_player_${runId}`, `多账号玩家${runId.slice(-5)}`);
  const room = await api('/api/rooms', {
    token: owner.token,
    method: 'POST',
    body: {
      ruleSetId: 'long-holdem',
      name: `多账号验收${runId.slice(-5)}`,
      seatCount: 2,
      minPlayersToStart: 2,
      minBuyIn: 1000,
      maxBuyIn: 1000,
    },
  });

  const browser = await chromium.launch({ headless: process.env.DEPU_MULTIPLAYER_HEADED !== '1' });
  let ownerBrowser;
  let playerBrowser;
  try {
    ownerBrowser = await createAuthenticatedPage(browser, owner.token, `/room/${room.id}`, 'owner');
    await ownerBrowser.page.getByRole('button', { name: '邀请好友' }).waitFor();
    const ownerBuyIn = await openBuyInDialog(ownerBrowser.page, 1);
    await ownerBuyIn.getByRole('button', { name: '确定' }).click();
    await ownerBrowser.page.getByText(`${owner.user.nickname} · 座位 #1`).waitFor();

    playerBrowser = await createAuthenticatedPage(browser, player.token, '/lobby', 'player');
    await playerBrowser.page.getByLabel('邀请码').fill(room.inviteCode);
    await playerBrowser.page.getByRole('button', { name: '加入房间' }).click();
    await playerBrowser.page.waitForURL(`${frontendURL}/room/${room.id}`);
    let playerBuyIn;
    try {
      playerBuyIn = await openBuyInDialog(playerBrowser.page, 2);
    } catch (error) {
      await playerBrowser.page.screenshot({ path: resolve(outputDir, 'player-buy-in-failure.png'), fullPage: true });
      const pageText = (await playerBrowser.page.locator('body').innerText()).slice(0, 2_000);
      throw new Error(`player buy-in modal did not open after clicking seat 2; page text: ${pageText}`, { cause: error });
    }
    await playerBuyIn.getByRole('button', { name: '确定' }).click();
    await playerBrowser.page.getByText(`${player.user.nickname} · 座位 #2`).waitFor();
    await ownerBrowser.page.getByRole('button', { name: `座位 2 ${player.user.nickname}` }).waitFor();

    await waitForAudit(ownerBrowser.audit, 'socketFramesSent', 'room.subscribe');
    await waitForAudit(playerBrowser.audit, 'socketFramesSent', 'room.subscribe');
    await Promise.all([
      waitForCurrentHandRequestCountStable(ownerBrowser.audit, room.id),
      waitForCurrentHandRequestCountStable(playerBrowser.audit, room.id),
    ]);
    const requestsBeforeStart = {
      owner: currentHandRequestCount(ownerBrowser.audit, room.id),
      player: currentHandRequestCount(playerBrowser.audit, room.id),
    };

    const startButton = ownerBrowser.page.getByRole('button', { name: '开 始' });
    await startButton.waitFor({ state: 'visible' });
    await startButton.click();
    await ownerBrowser.page.getByText('牌局已开始').waitFor();
    await waitForAudit(ownerBrowser.audit, 'socketFramesSent', 'room.start_hand');
    await waitForAudit(ownerBrowser.audit, 'socketFramesReceived', 'hand.started');
    await waitForAudit(playerBrowser.audit, 'socketFramesReceived', 'hand.started');
    await ownerBrowser.page.getByText(/行动倒计时 \d+s/).waitFor();
    await playerBrowser.page.getByText(/行动倒计时 \d+s/).waitFor();

    const requestsAfterStart = {
      owner: currentHandRequestCount(ownerBrowser.audit, room.id),
      player: currentHandRequestCount(playerBrowser.audit, room.id),
    };
    assert(requestsAfterStart.owner === requestsBeforeStart.owner + 1, `owner start current-hand requests: before=${requestsBeforeStart.owner}, after=${requestsAfterStart.owner}`);
    assert(requestsAfterStart.player === requestsBeforeStart.player, `player current-hand requests changed during socket start: before=${requestsBeforeStart.player}, after=${requestsAfterStart.player}`);

    const targetHandCount = 3;
    const pagesBySeat = new Map([
      [1, { ...ownerBrowser, token: owner.token }],
      [2, { ...playerBrowser, token: player.token }],
    ]);
    const requestAuditBaseline = {
      owner: currentHandRequestCount(ownerBrowser.audit, room.id),
      player: currentHandRequestCount(playerBrowser.audit, room.id),
    };
    let ownerCountdown = 0;
    let playerCountdown = 0;
    let decreasedCountdown = 0;
    let illegalAction = null;
    let legalActionCount = 0;
    const socketSynchronization = [];
    const completedHands = [];
    const automaticStarts = [];
    let hand = await currentHand(owner.token, room.id);
    assert(hand, 'hand was not created');
    for (let handIndex = 0; handIndex < targetHandCount; handIndex += 1) {
      const gameId = hand.handId;
      assert(gameId && !completedHands.some((item) => item.gameId === gameId), `duplicate current hand ${gameId}`);
      const actor = pagesBySeat.get(hand.currentSeat);
      const nonActor = [...pagesBySeat.entries()].find(([seatNo]) => seatNo !== hand.currentSeat)?.[1];
      assert(actor && nonActor, `invalid acting seat ${hand.currentSeat}`);
      await actor.page.getByText('轮到我').waitFor({ state: 'visible' });

      if (handIndex === 0) {
        ownerCountdown = await countdownValue(ownerBrowser.page);
        playerCountdown = await countdownValue(playerBrowser.page);
        assert(Math.abs(ownerCountdown - playerCountdown) <= 1, `countdown mismatch: owner=${ownerCountdown}, player=${playerCountdown}`);
        decreasedCountdown = await waitForCountdownDecrease(actor.page, await countdownValue(actor.page));
        illegalAction = await sendRawSocketAction(nonActor.page, nonActor.token, room.id, 'fold');
        assert(illegalAction.type === 'error' && illegalAction.code === 'not_your_turn', `illegal action result = ${JSON.stringify(illegalAction)}`);
      }

      let handActionCount = 0;
      while (hand) {
        const actingBrowser = pagesBySeat.get(hand.currentSeat);
        assert(actingBrowser, `no browser page for acting seat ${hand.currentSeat}`);
        await actingBrowser.page.getByText('轮到我').waitFor({ state: 'visible' });
        assert(hand.availableActions.includes('all_in'), `all_in unavailable for acting seat ${hand.currentSeat}: ${hand.availableActions.join(',')}`);
        const previousVersion = hand.version;
        await actingBrowser.page.getByRole('button', { name: '全下', exact: true }).click();
        legalActionCount += 1;
        handActionCount += 1;
        hand = await waitForHandChange(actingBrowser.token, room.id, previousVersion);
        if (hand) {
          await Promise.all([
            ownerBrowser.page.locator(`[data-testid="table-seat-${hand.currentSeat}"]`).evaluate((element) => element.classList.contains('acting')),
            playerBrowser.page.locator(`[data-testid="table-seat-${hand.currentSeat}"]`).evaluate((element) => element.classList.contains('acting')),
          ]).then((states) => {
            assert(states.every(Boolean), `acting seat ${hand.currentSeat} did not synchronize to both browsers`);
          });
          const [ownerPot, playerPot] = await Promise.all([
            ownerBrowser.page.locator('.table-pot-chip').innerText(),
            playerBrowser.page.locator('.table-pot-chip').innerText(),
          ]);
          assert(ownerPot === playerPot, `pot mismatch after action: owner=${ownerPot}, player=${playerPot}`);
          socketSynchronization.push({ handNo: handIndex + 1, currentSeat: hand.currentSeat, version: hand.version, pot: ownerPot });
        }
        assert(handActionCount <= 4, `hand ${handIndex + 1} did not settle after expected all-in actions`);
      }
      assert(handActionCount >= 2, `hand ${handIndex + 1} expected at least two legal actions, got ${handActionCount}`);

      await Promise.all([
        waitForAuditCount(ownerBrowser.audit, 'socketFramesReceived', 'hand.settled', handIndex + 1),
        waitForAuditCount(playerBrowser.audit, 'socketFramesReceived', 'hand.settled', handIndex + 1),
      ]);
      const historyHands = await waitForRoomHands(owner.token, room.id, handIndex + 1);
      const settledHand = historyHands.find((item) => item.gameId === gameId);
      assert(settledHand, `history missing settled game ${gameId}`);
      await Promise.all([
        verifySettledHandDisplay(ownerBrowser.page, settledHand),
        verifySettledHandDisplay(playerBrowser.page, settledHand),
      ]);
      completedHands.push({
        handNo: settledHand.handNo,
        handId: settledHand.handId,
        gameId,
        winnerSummary: settledHand.winnerSummary,
        totalPot: settledHand.totalPot,
        actions: handActionCount,
      });
      await ownerBrowser.page.screenshot({ path: resolve(outputDir, `owner-settled-${handIndex + 1}.png`), fullPage: true });
      await playerBrowser.page.screenshot({ path: resolve(outputDir, `player-settled-${handIndex + 1}.png`), fullPage: true });

      if (handIndex < targetHandCount - 1) {
        const expectedStartedCount = handIndex + 2;
        const [ownerStarts, playerStarts] = await Promise.all([
          waitForAuditCount(ownerBrowser.audit, 'socketFramesReceived', 'hand.started', expectedStartedCount, 12_000),
          waitForAuditCount(playerBrowser.audit, 'socketFramesReceived', 'hand.started', expectedStartedCount, 12_000),
        ]);
        const ownerStart = ownerStarts[expectedStartedCount - 1];
        const playerStart = playerStarts[expectedStartedCount - 1];
        assert(ownerStart.requestId === '' && playerStart.requestId === '', `hand ${expectedStartedCount} was not automatically started`);
        hand = await waitForCurrentHand(owner.token, room.id, (candidate) => candidate?.handId && candidate.handId !== gameId, 12_000);
        assert(ownerStart.handId === hand.handId && playerStart.handId === hand.handId, `automatic hand ${expectedStartedCount} socket ids did not match authority`);
        automaticStarts.push({ handNo: expectedStartedCount, handId: hand.handId, ownerAt: ownerStart.at, playerAt: playerStart.at });
      }
    }

    assert(completedHands.length === targetHandCount, `completed hands = ${completedHands.length}`);
    assert(automaticStarts.length === targetHandCount - 1, `automatic starts = ${automaticStarts.length}`);

    const insufficientPlayersStartedCount = auditEntries(ownerBrowser.audit, 'socketFramesReceived', 'hand.started').length;
    await api(`/api/rooms/${room.id}/seats/2`, { token: player.token, method: 'DELETE' });
    await ownerBrowser.page.getByText('玩家 1/2').waitFor({ timeout: 5_000 });
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 3_000));
    const insufficientPlayersRoom = await api(`/api/rooms/${room.id}`, { token: owner.token });
    const insufficientPlayersHand = await currentHand(owner.token, room.id);
    const insufficientPlayers = {
      roomStatus: insufficientPlayersRoom.status,
      occupiedSeats: insufficientPlayersRoom.seats.filter((seat) => seat.userId).length,
      currentHand: insufficientPlayersHand,
      startedEventsBefore: insufficientPlayersStartedCount,
      startedEventsAfter: auditEntries(ownerBrowser.audit, 'socketFramesReceived', 'hand.started').length,
    };
    assert(insufficientPlayers.roomStatus === 'waiting', `insufficient players room status = ${insufficientPlayers.roomStatus}`);
    assert(insufficientPlayers.occupiedSeats === 1, `insufficient players occupied seats = ${insufficientPlayers.occupiedSeats}`);
    assert(insufficientPlayers.currentHand === null, 'insufficient players created a current hand');
    assert(insufficientPlayers.startedEventsAfter === insufficientPlayers.startedEventsBefore, 'insufficient players broadcast a duplicate hand.started');

    const requestAudit = {
      owner: ownerBrowser.audit.requests,
      player: playerBrowser.audit.requests,
      currentHandBaseline: requestAuditBaseline,
      currentHandFinal: {
        owner: currentHandRequestCount(ownerBrowser.audit, room.id),
        player: currentHandRequestCount(playerBrowser.audit, room.id),
      },
    };
    assert(requestAudit.currentHandFinal.owner === requestAuditBaseline.owner, 'owner performed current-hand polling during socket play');
    assert(requestAudit.currentHandFinal.player === requestAuditBaseline.player, 'player performed current-hand polling during socket play');

    for (const browserAudit of [ownerBrowser.audit, playerBrowser.audit]) {
      assert(auditEntries(browserAudit, 'socketFramesReceived', 'hand.started').length === targetHandCount, `${browserAudit.label} hand.started count mismatch`);
      assert(browserAudit.socketFramesReceived.some((entry) => entry.type === 'hand.updated'), `${browserAudit.label} missed hand.updated`);
      assert(auditEntries(browserAudit, 'socketFramesReceived', 'hand.settled').length === targetHandCount, `${browserAudit.label} hand.settled count mismatch`);
      const unexpectedErrors = browserAudit.errors.filter((message) => !message.includes('status of 404'));
      assert(unexpectedErrors.length === 0, `${browserAudit.label} browser errors: ${unexpectedErrors.join(' | ')}`);
    }
    assert(auditEntries(ownerBrowser.audit, 'socketFramesSent', 'room.start_hand').length === 1, 'owner did not send exactly one room.start_hand');
    assert(ownerBrowser.audit.socketFramesSent.some((entry) => entry.type === 'room.action') || playerBrowser.audit.socketFramesSent.some((entry) => entry.type === 'room.action'), 'legal actions did not use room.action');

    const [
      [userCount],
      [memberCount],
      [settledHandCount],
      [distinctGameCount],
      [participantCount],
      [actionCount],
      databaseWalletRows,
      databaseProfitRows,
    ] = await Promise.all([
      queryRows(`select count(*) from users where username in ('multi_owner_${runId}', 'multi_player_${runId}')`),
      queryRows(`select count(*) from room_members where room_id = '${room.id}'`),
      queryRows(`select count(*) from hand_results where room_id = '${room.id}'`),
      queryRows(`select count(distinct game_id) from hand_results where room_id = '${room.id}'`),
      queryRows(`select count(*) from hand_participants where room_id = '${room.id}'`),
      queryRows(`select count(*) from actions where game_id in (select game_id from hand_results where room_id = '${room.id}')`),
      queryRows(`select user_id, balance from wallets where user_id in ('${owner.user.id}', '${player.user.id}') order by user_id`),
      queryRows(`select user_id, sum(profit), count(*) from hand_participants where room_id = '${room.id}' group by user_id order by user_id`),
    ]);
    const settledHands = Number(settledHandCount[0]);
    const duplicateHandIds = completedHands.filter((item, index) => completedHands.findIndex((candidate) => candidate.gameId === item.gameId) !== index);
    const databaseEvidence = {
      users: Number(userCount[0]),
      members: Number(memberCount[0]),
      settledHands,
      distinctGames: Number(distinctGameCount[0]),
      participants: Number(participantCount[0]),
      actions: Number(actionCount[0]),
    };
    assert(databaseEvidence.users === 2, `database users = ${databaseEvidence.users}`);
    assert(databaseEvidence.members === 2, `database members = ${databaseEvidence.members}`);
    assert(settledHands === targetHandCount, `database settled hands = ${databaseEvidence.settledHands}`);
    assert(databaseEvidence.distinctGames === targetHandCount, `database distinct games = ${databaseEvidence.distinctGames}`);
    assert(databaseEvidence.participants === targetHandCount * 2, `database participants = ${databaseEvidence.participants}`);
    assert(duplicateHandIds.length === 0, `duplicate hand ids = ${JSON.stringify(duplicateHandIds)}`);
    assert(databaseEvidence.actions >= legalActionCount, `database actions = ${databaseEvidence.actions}, browser legal actions = ${legalActionCount}`);

    const [ownerWallet, playerWallet, ownerHistory, playerHistory, roomHistory, leaderboard] = await Promise.all([
      api('/api/me/wallet', { token: owner.token }),
      api('/api/me/wallet', { token: player.token }),
      api('/api/me/hands', { token: owner.token }),
      api('/api/me/hands', { token: player.token }),
      api(`/api/rooms/${room.id}/hands/recent`, { token: owner.token }),
      api(`/api/rooms/${room.id}/leaderboard`, { token: owner.token }),
    ]);
    const databaseWallets = Object.fromEntries(databaseWalletRows.map(([userId, balance]) => [userId, Number(balance)]));
    const databaseProfits = Object.fromEntries(databaseProfitRows.map(([userId, profit, handsPlayed]) => [userId, { profit: Number(profit), handsPlayed: Number(handsPlayed) }]));
    const walletConsistency = {
      owner: { api: ownerWallet.balance, database: databaseWallets[owner.user.id] },
      player: { api: playerWallet.balance, database: databaseWallets[player.user.id] },
      totalProfit: (databaseProfits[owner.user.id]?.profit || 0) + (databaseProfits[player.user.id]?.profit || 0),
    };
    assert(walletConsistency.owner.api === walletConsistency.owner.database, `owner wallet mismatch ${JSON.stringify(walletConsistency.owner)}`);
    assert(walletConsistency.player.api === walletConsistency.player.database, `player wallet mismatch ${JSON.stringify(walletConsistency.player)}`);
    assert(walletConsistency.totalProfit === 0, `wallet hand profits do not balance: ${walletConsistency.totalProfit}`);

    const historyConsistency = {
      roomHands: roomHistory.items.length,
      ownerHands: ownerHistory.items.filter((item) => item.roomId === room.id).length,
      playerHands: playerHistory.items.filter((item) => item.roomId === room.id).length,
      handNumbers: roomHistory.items.map((item) => item.handNo).sort((left, right) => left - right),
    };
    assert(historyConsistency.roomHands === targetHandCount, `room history hands = ${historyConsistency.roomHands}`);
    assert(historyConsistency.ownerHands === targetHandCount, `owner history hands = ${historyConsistency.ownerHands}`);
    assert(historyConsistency.playerHands === targetHandCount, `player history hands = ${historyConsistency.playerHands}`);
    assert(historyConsistency.handNumbers.join(',') === '1,2,3', `room history hand numbers = ${historyConsistency.handNumbers.join(',')}`);

    const leaderboardConsistency = leaderboard.items.map((item) => ({
      userId: item.userId,
      apiHandsPlayed: item.handsPlayed,
      databaseHandsPlayed: databaseProfits[item.userId]?.handsPlayed,
      apiNetProfit: item.netProfit,
      databaseNetProfit: databaseProfits[item.userId]?.profit,
    }));
    assert(leaderboardConsistency.length === 2, `leaderboard items = ${leaderboardConsistency.length}`);
    assert(leaderboardConsistency.every((item) => item.apiHandsPlayed === targetHandCount && item.apiHandsPlayed === item.databaseHandsPlayed), `leaderboard hands mismatch ${JSON.stringify(leaderboardConsistency)}`);
    assert(leaderboardConsistency.every((item) => item.apiNetProfit === item.databaseNetProfit), `leaderboard profit mismatch ${JSON.stringify(leaderboardConsistency)}`);

    const report = {
      status: 'passed',
      database: databaseName,
      roomId: room.id,
      inviteCode: room.inviteCode,
      accounts: [owner.user.nickname, player.user.nickname],
      countdown: { owner: ownerCountdown, player: playerCountdown, decreasedTo: decreasedCountdown },
      illegalAction,
      legalActionCount,
      completedHands,
      automaticStarts,
      insufficientPlayers,
      duplicateHandIds,
      socketSynchronization,
      requestAudit,
      socketAudit: {
        owner: { sent: ownerBrowser.audit.socketFramesSent, received: ownerBrowser.audit.socketFramesReceived },
        player: { sent: playerBrowser.audit.socketFramesSent, received: playerBrowser.audit.socketFramesReceived },
      },
      databaseEvidence,
      walletConsistency,
      leaderboardConsistency,
      historyConsistency,
      artifacts: completedHands.flatMap((item) => [`owner-settled-${item.handNo}.png`, `player-settled-${item.handNo}.png`]),
    };
    await writeFile(resolve(outputDir, 'multiplayer-acceptance-report.json'), `${JSON.stringify(report, null, 2)}\n`);
    process.stdout.write(`table multiplayer browser acceptance passed: ${resolve(outputDir, 'multiplayer-acceptance-report.json')}\n`);
  } finally {
    await Promise.all([
      ownerBrowser?.context.close(),
      playerBrowser?.context.close(),
    ].filter(Boolean));
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
  await writeFile(resolve(outputDir, 'multiplayer-acceptance-report.json'), `${JSON.stringify({
    status: 'failed',
    database: databaseName,
    error: error instanceof Error ? error.stack || error.message : String(error),
  }, null, 2)}\n`);
  console.error(error);
  process.exitCode = 1;
} finally {
  stopProcesses();
}
