import { describe, expect, it } from 'vitest';
import roomSource from './pages/RoomPage.vue?raw';
import roomInfoSource from './pages/RoomInfoPage.vue?raw';
import roomPlayersSource from './pages/RoomPlayersPage.vue?raw';
import appStateSource from './composables/useAppState.ts?raw';

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
});
