import { spawn } from 'node:child_process';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = resolve(frontendRoot, '..');
const backendRoot = resolve(repoRoot, 'backend');
const outputDir = resolve(
  process.env.DEPU_RECONNECT_ACCEPTANCE_OUTPUT || resolve(frontendRoot, '.artifacts/table-reconnect-browser'),
  new Date().toISOString().replaceAll(':', '-'),
);
const backendPort = Number(process.env.DEPU_RECONNECT_BACKEND_PORT || 15178);
const frontendPort = Number(process.env.DEPU_RECONNECT_FRONTEND_PORT || 15179);
const backendURL = `http://127.0.0.1:${backendPort}`;
const frontendURL = `http://127.0.0.1:${frontendPort}`;
const databaseDSN = process.env.DEPU_DSN || '';
const databaseName = process.env.DEPU_TEST_DATABASE || '';
const children = [];

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function sleep(milliseconds) {
  return new Promise((resolvePromise) => setTimeout(resolvePromise, milliseconds));
}

function parseMessage(message) {
  const text = Buffer.isBuffer(message) ? message.toString('utf8') : String(message);
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
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
    await sleep(200);
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

async function waitForCondition(predicate, message, timeoutMs = 10_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const result = await predicate();
    if (result) return result;
    await sleep(50);
  }
  throw new Error(message);
}

function attachPageAudit(page, label) {
  const audit = { label, requests: [], consoleErrors: [], pageErrors: [] };
  page.on('request', (request) => {
    const url = new URL(request.url());
    if (url.pathname.startsWith('/api/')) {
      audit.requests.push({ method: request.method(), path: url.pathname, at: Date.now() });
    }
  });
  page.on('console', (message) => {
    if (message.type() === 'error') audit.consoleErrors.push(message.text());
  });
  page.on('pageerror', (error) => audit.pageErrors.push(error.message));
  return audit;
}

async function attachSocketController(context, label) {
  const state = {
    label,
    active: null,
    commands: [],
    serverMessages: [],
    snapshots: [],
    blockedRequests: [],
    droppedAcknowledgements: [],
    armedType: '',
    disconnects: 0,
  };

  await context.routeWebSocket(/\/api\/socket(?:\?|$)/, (socket) => {
    const server = socket.connectToServer();
    const connection = { socket, server };
    state.active = connection;
    socket.onMessage((message) => {
      const parsed = parseMessage(message);
      if (parsed?.type) {
        state.commands.push({
          type: parsed.type,
          requestId: parsed.requestId || '',
          roomId: parsed.roomId || '',
          payload: parsed.payload || {},
          at: Date.now(),
        });
        if (state.armedType === parsed.type) {
          state.armedType = '';
          state.blockedRequests.push({ type: parsed.type, requestId: parsed.requestId, at: Date.now() });
        }
      }
      server.send(message);
    });
    server.onMessage((message) => {
      const parsed = parseMessage(message);
      if (parsed?.type) {
        state.serverMessages.push({ ...parsed, at: Date.now() });
        if (parsed.type === 'room.snapshot') state.snapshots.push(parsed);
        if (
          (parsed.type === 'ack' || parsed.type === 'error')
          && state.blockedRequests.some((entry) => entry.requestId === parsed.requestId)
        ) {
          state.droppedAcknowledgements.push({ type: parsed.type, requestId: parsed.requestId, at: Date.now() });
          return;
        }
      }
      socket.send(message);
    });
  });

  return {
    state,
    arm(type) {
      state.armedType = type;
    },
    async disconnect() {
      const connection = state.active;
      assert(connection, `${label} has no active socket to disconnect`);
      state.disconnects += 1;
      await connection.socket.close({ code: 1001, reason: 'acceptance reconnect' });
      if (state.active === connection) state.active = null;
    },
    commandCount(type) {
      return state.commands.filter((entry) => entry.type === type).length;
    },
    messageCount(type) {
      return state.serverMessages.filter((entry) => entry.type === type).length;
    },
  };
}

async function createAuthenticatedPage(browser, token, roomId, label) {
  const context = await browser.newContext({ viewport: { width: 1280, height: 720 } });
  await context.addInitScript((authToken) => {
    window.sessionStorage.setItem('depu.auth.token', authToken);
  }, token);
  const controller = await attachSocketController(context, label);
  const page = await context.newPage();
  const audit = attachPageAudit(page, label);
  await page.goto(`${frontendURL}/room/${roomId}`);
  await waitForCondition(() => controller.state.snapshots.length > 0, `${label} did not receive initial room.snapshot`);
  return { context, page, audit, controller, token };
}

async function reconnectPage(browserState, roomId, artifactName) {
  const previousSnapshots = browserState.controller.state.snapshots.length;
  await browserState.controller.disconnect();
  await browserState.page.reload();
  const snapshot = await waitForCondition(
    () => browserState.controller.state.snapshots.length > previousSnapshots
      ? browserState.controller.state.snapshots.at(-1)
      : null,
    `${browserState.audit.label} did not receive room.snapshot after reconnect`,
  );
  await browserState.page.screenshot({ path: resolve(outputDir, artifactName), fullPage: true });
  return snapshot;
}

function assertSnapshotShape(snapshot, expectedRoomId) {
  assert(snapshot.type === 'room.snapshot', `expected room.snapshot, received ${snapshot.type}`);
  assert(snapshot.roomId === expectedRoomId, `snapshot roomId = ${snapshot.roomId}`);
  assert(snapshot.payload?.room?.id === expectedRoomId, 'snapshot room payload missing');
  for (const key of ['presence', 'recentActionLog', 'recentChatMessages', 'leaderboard']) {
    assert(Array.isArray(snapshot.payload?.[key]), `snapshot ${key} is not an array`);
  }
}

async function waitForTableHand(page, handId) {
  await page.locator('.subtle-strip').waitFor({ state: 'visible' });
  await waitForCondition(async () => {
    const actingSeat = await page.locator('.casino-seat-node.acting').getAttribute('aria-label').catch(() => null);
    return actingSeat ? { actingSeat } : null;
  }, `table did not render acting seat for ${handId}`);
}

async function waitForWaitingTable(page) {
  await waitForCondition(async () => !(await page.locator('.subtle-strip').isVisible().catch(() => false)), 'table still displays a current hand');
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
  const owner = await register(`reconnect_owner_${runId}`, `重连房主${runId.slice(-5)}`);
  const player = await register(`reconnect_player_${runId}`, `重连玩家${runId.slice(-5)}`);
  const room = await api('/api/rooms', {
    token: owner.token,
    method: 'POST',
    body: {
      ruleSetId: 'long-holdem',
      name: `重连验收${runId.slice(-5)}`,
      seatCount: 2,
      minPlayersToStart: 2,
      minBuyIn: 1000,
      maxBuyIn: 1000,
    },
  });
  await api('/api/rooms/join', { token: player.token, method: 'POST', body: { inviteCode: room.inviteCode } });
  await api(`/api/rooms/${room.id}/seats/1`, { token: owner.token, method: 'POST', body: { buyInChips: 1000 } });
  await api(`/api/rooms/${room.id}/seats/2`, { token: player.token, method: 'POST', body: { buyInChips: 1000 } });

  const browser = await chromium.launch({ headless: process.env.DEPU_RECONNECT_HEADED !== '1' });
  let ownerBrowser;
  let playerBrowser;
  try {
    ownerBrowser = await createAuthenticatedPage(browser, owner.token, room.id, 'owner');
    playerBrowser = await createAuthenticatedPage(browser, player.token, room.id, 'player');
    await Promise.all([
      ownerBrowser.page.getByText(`${owner.user.nickname} · 座位 #1`).waitFor(),
      playerBrowser.page.getByText(`${player.user.nickname} · 座位 #2`).waitFor(),
    ]);

    const waitingSnapshot = await reconnectPage(ownerBrowser, room.id, 'waiting-reconnect.png');
    assertSnapshotShape(waitingSnapshot, room.id);
    assert(waitingSnapshot.payload.room.status === 'waiting', `waiting snapshot room status = ${waitingSnapshot.payload.room.status}`);
    assert(waitingSnapshot.payload.hand === null, 'waiting snapshot unexpectedly contains a hand');
    await waitForWaitingTable(ownerBrowser.page);
    const waitingReconnect = {
      roomVersion: waitingSnapshot.roomVersion,
      roomStatus: waitingSnapshot.payload.room.status,
      hand: waitingSnapshot.payload.hand,
      snapshotCollections: {
        presence: waitingSnapshot.payload.presence.length,
        actionLog: waitingSnapshot.payload.recentActionLog.length,
        chatMessages: waitingSnapshot.payload.recentChatMessages.length,
        leaderboard: waitingSnapshot.payload.leaderboard.length,
      },
    };

    ownerBrowser.controller.arm('room.start_hand');
    await ownerBrowser.page.getByRole('button', { name: '开 始' }).click();
    await waitForCondition(
      () => ownerBrowser.controller.state.droppedAcknowledgements.some((entry) => ownerBrowser.controller.state.blockedRequests.some((blocked) => blocked.type === 'room.start_hand' && blocked.requestId === entry.requestId)),
      'room.start_hand acknowledgement was not intercepted',
    );
    const startedMessage = await waitForCondition(
      () => ownerBrowser.controller.state.serverMessages.find((entry) => entry.type === 'hand.started'),
      'owner did not receive hand.started after the unacknowledged start command',
    );
    const startedHandId = startedMessage.handId || startedMessage.payload?.hand?.handId;
    assert(startedHandId, 'hand.started did not include handId');
    await waitForTableHand(ownerBrowser.page, startedHandId);
    const startCommandCountBeforeReconnect = ownerBrowser.controller.commandCount('room.start_hand');
    const playingSnapshot = await reconnectPage(ownerBrowser, room.id, 'playing-reconnect.png');
    assertSnapshotShape(playingSnapshot, room.id);
    assert(playingSnapshot.payload.room.status === 'playing', `playing snapshot room status = ${playingSnapshot.payload.room.status}`);
    assert(playingSnapshot.payload.hand?.handId === startedHandId, `playing snapshot handId = ${playingSnapshot.payload.hand?.handId}`);
    await waitForTableHand(ownerBrowser.page, startedHandId);
    await sleep(500);
    assert(ownerBrowser.controller.commandCount('room.start_hand') === startCommandCountBeforeReconnect, 'room.start_hand was replayed after reconnect');
    const authoritativePlayingHand = await currentHand(owner.token, room.id);
    assert(authoritativePlayingHand?.handId === playingSnapshot.payload.hand.handId, 'playing snapshot does not match authoritative current hand');
    assert(authoritativePlayingHand?.version === playingSnapshot.payload.hand.version, 'playing snapshot hand version mismatch');
    const playingReconnect = {
      roomVersion: playingSnapshot.roomVersion,
      handId: playingSnapshot.payload.hand.handId,
      handVersion: playingSnapshot.handVersion,
      currentSeat: playingSnapshot.payload.hand.currentSeat,
      availableActions: playingSnapshot.payload.hand.availableActions,
    };

    const actorSeat = authoritativePlayingHand.currentSeat;
    const actorBrowser = actorSeat === 1 ? ownerBrowser : playerBrowser;
    const otherAccount = actorSeat === 1 ? player : owner;
    const otherSeat = actorSeat === 1 ? 2 : 1;
    actorBrowser.controller.arm('room.action');
    await actorBrowser.page.getByRole('button', { name: '弃牌', exact: true }).click();
    await waitForCondition(
      () => actorBrowser.controller.state.droppedAcknowledgements.some((entry) => actorBrowser.controller.state.blockedRequests.some((blocked) => blocked.type === 'room.action' && blocked.requestId === entry.requestId)),
      'room.action acknowledgement was not intercepted',
    );
    const settledMessage = await waitForCondition(
      () => actorBrowser.controller.state.serverMessages.find((entry) => entry.type === 'hand.settled' && entry.handId === startedHandId),
      'actor did not receive hand.settled after the unacknowledged action command',
    );
    await api(`/api/rooms/${room.id}/seats/${otherSeat}`, { token: otherAccount.token, method: 'DELETE' });
    const actionCommandCountBeforeReconnect = actorBrowser.controller.commandCount('room.action');
    const settlementSnapshot = await reconnectPage(actorBrowser, room.id, 'settlement-reconnect.png');
    assertSnapshotShape(settlementSnapshot, room.id);
    assert(settlementSnapshot.payload.room.status === 'waiting', `settlement snapshot room status = ${settlementSnapshot.payload.room.status}`);
    assert(settlementSnapshot.payload.hand === null, 'settlement snapshot unexpectedly contains a current hand');
    assert(settlementSnapshot.roomVersion >= settledMessage.roomVersion, 'settlement snapshot room version regressed');
    await waitForWaitingTable(actorBrowser.page);
    await sleep(2_500);
    assert(actorBrowser.controller.commandCount('room.action') === actionCommandCountBeforeReconnect, 'room.action was replayed after reconnect');
    assert(await currentHand(actorBrowser.token, room.id) === null, 'insufficient-player room unexpectedly auto-started after settlement reconnect');
    const recentHands = await api(`/api/rooms/${room.id}/hands/recent`, { token: actorBrowser.token });
    assert(recentHands.items?.some((hand) => hand.gameId === startedHandId), 'settled hand missing from authoritative history');
    const settlementReconnect = {
      roomVersion: settlementSnapshot.roomVersion,
      roomStatus: settlementSnapshot.payload.room.status,
      settledHandId: startedHandId,
      historyHandIds: recentHands.items.map((hand) => hand.gameId),
      seatedPlayers: settlementSnapshot.payload.room.seats.filter((seat) => seat.userId).length,
    };

    const unacknowledgedCommands = [
      ...ownerBrowser.controller.state.blockedRequests,
      ...playerBrowser.controller.state.blockedRequests,
    ];
    const replayedCommands = [
      ...(ownerBrowser.controller.commandCount('room.start_hand') > 1 ? ['room.start_hand'] : []),
      ...(ownerBrowser.controller.commandCount('room.action') + playerBrowser.controller.commandCount('room.action') > 1 ? ['room.action'] : []),
    ];
    assert(unacknowledgedCommands.some((entry) => entry.type === 'room.start_hand'), 'missing unacknowledged room.start_hand evidence');
    assert(unacknowledgedCommands.some((entry) => entry.type === 'room.action'), 'missing unacknowledged room.action evidence');
    assert(replayedCommands.length === 0, `commands replayed after reconnect: ${replayedCommands.join(', ')}`);
    assert(ownerBrowser.audit.pageErrors.length === 0, `owner page errors: ${ownerBrowser.audit.pageErrors.join('; ')}`);
    assert(playerBrowser.audit.pageErrors.length === 0, `player page errors: ${playerBrowser.audit.pageErrors.join('; ')}`);

    const report = {
      status: 'passed',
      database: databaseName,
      roomId: room.id,
      accounts: [owner.user.nickname, player.user.nickname],
      waitingReconnect,
      playingReconnect,
      settlementReconnect,
      unacknowledgedCommands,
      replayedCommands,
      socketEvidence: {
        owner: {
          disconnects: ownerBrowser.controller.state.disconnects,
          snapshots: ownerBrowser.controller.state.snapshots.length,
          startCommands: ownerBrowser.controller.commandCount('room.start_hand'),
          actionCommands: ownerBrowser.controller.commandCount('room.action'),
        },
        player: {
          disconnects: playerBrowser.controller.state.disconnects,
          snapshots: playerBrowser.controller.state.snapshots.length,
          startCommands: playerBrowser.controller.commandCount('room.start_hand'),
          actionCommands: playerBrowser.controller.commandCount('room.action'),
        },
      },
      browserAudit: {
        owner: ownerBrowser.audit,
        player: playerBrowser.audit,
      },
      artifacts: ['waiting-reconnect.png', 'playing-reconnect.png', 'settlement-reconnect.png'],
    };
    await writeFile(resolve(outputDir, 'reconnect-acceptance-report.json'), `${JSON.stringify(report, null, 2)}\n`);
    process.stdout.write(`table reconnect browser acceptance passed: ${resolve(outputDir, 'reconnect-acceptance-report.json')}\n`);
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
  await writeFile(resolve(outputDir, 'reconnect-acceptance-report.json'), `${JSON.stringify({
    status: 'failed',
    database: databaseName,
    error: error instanceof Error ? error.stack || error.message : String(error),
  }, null, 2)}\n`);
  console.error(error);
  process.exitCode = 1;
} finally {
  stopProcesses();
}
