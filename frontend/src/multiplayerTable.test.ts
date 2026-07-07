import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import roomSource from './pages/RoomPage.vue?raw';
import roomInfoSource from './pages/RoomInfoPage.vue?raw';
import roomPlayersSource from './pages/RoomPlayersPage.vue?raw';
import lobbySource from './pages/LobbyPage.vue?raw';
import appStateSource from './composables/useAppState.ts?raw';

const styleSource = readFileSync(resolve(process.cwd(), 'src/style.css'), 'utf-8');

describe('multiplayer table contract', () => {
  it('keeps room subpages reloadable by restoring token and room context from route', () => {
    [
      "sessionStorage.getItem",
      "sessionStorage.setItem",
      "sessionStorage.removeItem",
      "setToken(payload.token)",
    ].forEach((token) => expect(appStateSource).toContain(token));

    [
      'ensureRouteRoom',
      "room.value = emptyRoom(route.params.roomId)",
      'await refreshRoom()',
    ].forEach((token) => expect(roomInfoSource).toContain(token));

    [
      'ensureRouteRoom',
      "room.value = emptyRoom(route.params.roomId)",
      'await refreshRoom()',
    ].forEach((token) => expect(roomPlayersSource).toContain(token));

    expect(roomInfoSource).not.toContain("router.push('/lobby')");
    expect(roomPlayersSource).not.toContain("router.push('/lobby')");
  });

  it('does not surface an expected missing current hand as a lobby error', () => {
    [
      'safeRefreshCurrentRoomHand',
      'current hand not found',
      'currentRoomHand.value = null',
      'refreshCurrentRoomHand: safeRefreshCurrentRoomHand'
    ].forEach((token) => expect(appStateSource).toContain(token));
  });

  it('clears a stale persisted token when profile refresh is unauthorized', () => {
    [
      'authentication required',
      'doLogout()',
      'throw err'
    ].forEach((token) => expect(appStateSource).toContain(token));
  });

  it('clears stale room errors when returning to the lobby', () => {
    expect(appStateSource).toContain('function clearError()');
    expect(lobbySource).toContain('clearError');
    expect(lobbySource).toContain('clearError();');
  });

  it('keeps the main room page focused on table actions and table state', () => {
    [
      'room-shell-fullscreen',
      '返回',
      'router.back()',
      'refreshCurrentRoomHand',
      'doRoomAction',
      'currentRoomHand',
      'myRoomSeat',
		'myRoomHandPlayer',
		'doTakeSeat',
		'sitAtFirstOpenSeat',
		'roomSeat(seatNo)?.userId',
		'v-if="myRoomSeat"',
		'router.push(`/room/${room.id}/players`)',
      'router.push(`/room/${room.id}/info`)',
      'board-zone',
      'seat-ring',
      'casino-table',
      'seat-ring-casino',
      '坐下 #',
      '选座位',
      'hero-actions-dock'
    ].forEach((token) => expect(roomSource).toContain(token));

    [
      '当前战绩',
      '观众（',
      'player-board-table',
      '座位操作'
    ].forEach((token) => expect(roomSource).not.toContain(token));
  });

  it('moves room metadata and owner controls into room info page', () => {
    [
      '房间信息',
      '邀请码',
      '状态',
      '房主',
      '房主开局'
    ].forEach((token) => expect(roomInfoSource).toContain(token));
  });

  it('moves player list and spectator info into room players page', () => {
    [
      '当前战绩',
      '人数',
      '观众（',
      'player-board-table',
      'live-score-table',
      'audience-grid',
      '房主坐下',
      '选择一个空位即可上桌',
      '坐下 #'
    ].forEach((token) => expect(roomPlayersSource).toContain(token));
  });

  it('uses socket commands for formal room start and player actions', () => {
    [
      'createRoomSocketClient',
      "room.start_hand",
      "room.action",
      "room.snapshot",
      "hand.started",
      "hand.updated",
      "hand.settled",
      "wallet.updated",
      'connectRoomSocket'
    ].forEach((token) => expect(appStateSource).toContain(token));

    expect(appStateSource).not.toContain('startRoomHand,');
    expect(appStateSource).not.toContain('submitRoomAction,');
    expect(appStateSource).not.toContain('currentRoomHand.value = await startRoomHand');
    expect(appStateSource).not.toContain('currentRoomHand.value = await submitRoomAction');
  });

  it('surfaces V1.1 realtime table experience state on the room page', () => {
    [
      'roomPresence',
      'actionLog',
      'chatMessages',
      'roomLeaderboard',
      'sendRoomChat',
      'hand.log.appended',
      'player.presence.updated',
      'chat.message',
      'room.leaderboard.updated',
      "chat.send"
    ].forEach((token) => expect(appStateSource).toContain(token));

    [
      'remainingActionSeconds',
      '行动倒计时',
      'nowMs',
      'window.setInterval',
      'window.clearInterval',
      '在线',
      '离线',
      '动作日志',
      '房间战绩榜',
      '聊天表情',
      'chatInput',
      'sendChat',
      'sendEmoji'
    ].forEach((token) => expect(roomSource).toContain(token));

    expect(roomSource).not.toContain('startRoomPolling');
    expect(roomSource).not.toContain('stopRoomPolling');
  });

  it('lets players choose an action amount and sends it with socket actions', () => {
    [
      'actionAmount',
      'minActionAmount',
      'maxActionAmount',
      'canChooseActionAmount',
      'type="range"',
      'v-model.number="actionAmount"',
      '执行 {{ actionLabel(action) }}',
      'submitRoomAction(action)'
    ].forEach((token) => expect(roomSource).toContain(token));

    expect(appStateSource).toContain('async function doRoomAction(action: string, amount = 0)');
    expect(appStateSource).toContain("roomSocket?.send('room.action', room.value!.id, { action, amount })");
    expect(appStateSource).not.toContain("roomSocket?.send('room.action', room.value!.id, { action, amount: 0 })");
  });

  it('provides table exits and room-specific history from the table surface', () => {
    [
      "router.push('/lobby')",
      'finally',
      'data-testid="table-leave-button"',
      'data-testid="table-lobby-button"',
      '离开牌桌',
      '返回大厅',
      '牌桌历史',
      'room-history-preview',
      'recentRoomHands.slice(0, 3)',
      '`/room/${room.id}/hands/${hand.handId}/replay`'
    ].forEach((token) => expect(roomSource).toContain(token));
  });

  it('prevents already seated users from taking another seat from the players page', () => {
    [
      'disabled: true',
      '你已在其他座位',
      'seat.userId !== myRoomSeat.value?.userId'
    ].forEach((token) => expect(roomPlayersSource).toContain(token));
  });

  it('keeps V1.1 log leaderboard and chat tools outside the table surface', () => {
    const tableCloseThenTools = '</div>\n\n      <aside class="table-side-panel v11-table-tools" aria-label="牌桌工具">';

    expect(roomSource).toContain(tableCloseThenTools);
    expect(styleSource).toContain('overflow-y: auto;');
    expect(styleSource).toContain('grid-template-columns: repeat(3, minmax(0, 1fr));');
    expect(styleSource).toContain('grid-template-columns: 1fr;');
    expect(styleSource).not.toContain('.room-shell-fullscreen {\n  min-height: 100dvh;\n  padding: 0;\n  overflow: hidden;');
  });
});
