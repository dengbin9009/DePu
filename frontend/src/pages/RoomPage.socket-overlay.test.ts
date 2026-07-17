import { createApp, nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SocketEnvelope } from '../api/socketClient';
import type { RoomHandState, RoomResponse } from '../types/game';

const mocks = vi.hoisted(() => {
  const handlers = new Map<string, (message: SocketEnvelope) => void>();
  const connection = { connected: false };
  return {
    connection,
    handlers,
    fetchCurrentRoomHand: vi.fn(),
    fetchMe: vi.fn(),
    fetchRoom: vi.fn(),
    fetchRoomHandReplay: vi.fn(),
    fetchRoomHands: vi.fn(),
    fetchRoomLeaderboard: vi.fn(),
    fetchUserHands: vi.fn(),
    fetchWallet: vi.fn(),
    socketClient: {
      connect: vi.fn(async () => {
        connection.connected = true;
      }),
      isConnected: vi.fn(() => connection.connected),
      send: vi.fn(async () => ({})),
      on: vi.fn((type: string, handler: (message: SocketEnvelope) => void) => {
        handlers.set(type, handler);
        return () => handlers.delete(type);
      }),
      close: vi.fn(() => {
        connection.connected = false;
      }),
    },
    router: {
      back: vi.fn(),
      currentRoute: { value: { fullPath: '/room/room_1' } },
      push: vi.fn(),
    },
  };
});

vi.mock('../api/client', () => ({
  createRoom: vi.fn(),
  fetchCurrentRoomHand: mocks.fetchCurrentRoomHand,
  fetchMe: mocks.fetchMe,
  fetchRoom: mocks.fetchRoom,
  fetchRoomHandReplay: mocks.fetchRoomHandReplay,
  fetchRoomHands: mocks.fetchRoomHands,
  fetchRoomLeaderboard: mocks.fetchRoomLeaderboard,
  fetchUserHands: mocks.fetchUserHands,
  fetchWallet: mocks.fetchWallet,
  joinRoom: vi.fn(),
  leaveRoom: vi.fn(),
  leaveSeat: vi.fn(),
  login: vi.fn(),
  recharge: vi.fn(),
  register: vi.fn(),
  takeSeat: vi.fn(),
  updateProfile: vi.fn(),
}));

vi.mock('../api/socketClient', () => ({
  createRoomSocketClient: vi.fn(() => mocks.socketClient),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { roomId: 'room_1' } }),
  useRouter: () => mocks.router,
}));

function roomFixture(version: number, name: string): RoomResponse {
  return {
    id: 'room_1',
    version,
    inviteCode: 'ABC123',
    ownerUserId: 'user_1',
    name,
    status: 'playing',
    minPlayersToStart: 2,
    members: [
      { userId: 'user_1', nickname: 'Owner', role: 'owner', joinedAt: '2026-07-11T00:00:00Z' },
      { userId: 'user_2', nickname: 'Player', role: 'player', joinedAt: '2026-07-11T00:00:00Z' },
    ],
    seats: [
      { seatNo: 1, seatStatus: 'occupied', userId: 'user_1', nickname: 'Owner', buyInChips: 1000 },
      { seatNo: 2, seatStatus: 'occupied', userId: 'user_2', nickname: 'Player', buyInChips: 1000 },
    ],
    seatCount: 2,
  };
}

function handFixture(version: number, pot: number, boardCards: string[]): RoomHandState {
  return {
    roomId: 'room_1',
    handId: 'hand_1',
    version,
    status: 'flop',
    currentSeat: 2,
    pot,
    boardCards,
    players: [
      { seatNo: 1, name: 'Owner', stack: 900, holeCards: ['As', 'Kh'], status: 'active', streetCommitted: 50, handCommitted: 100 },
      { seatNo: 2, name: 'Player', stack: 900, status: 'active', streetCommitted: 50, handCommitted: 100 },
    ],
    availableActions: [],
  };
}

describe('RoomPage socket updates while overlays are open', () => {
  beforeEach(() => {
    vi.resetModules();
    window.sessionStorage.clear();
    window.sessionStorage.setItem('depu.auth.token', 'token_1');
    document.body.innerHTML = '<div id="app"></div>';
    mocks.connection.connected = false;
    mocks.handlers.clear();
    mocks.socketClient.connect.mockClear();
    mocks.socketClient.isConnected.mockClear();
    mocks.socketClient.send.mockClear();
    mocks.socketClient.on.mockClear();
    mocks.socketClient.close.mockClear();
    mocks.fetchMe.mockReset().mockResolvedValue({
      id: 'user_1',
      username: 'owner',
      nickname: 'Owner',
      walletBalance: 1000,
      handsPlayed: 0,
      totalProfit: 0,
      lastPlayedAt: null,
    });
    mocks.fetchWallet.mockReset().mockResolvedValue({ balance: 1000, transactions: [] });
    mocks.fetchUserHands.mockReset().mockResolvedValue({ items: [] });
    mocks.fetchRoom.mockReset().mockResolvedValue(roomFixture(1, '初始牌桌'));
    mocks.fetchCurrentRoomHand.mockReset().mockResolvedValue(handFixture(1, 100, ['2s', '3h', '4d']));
    mocks.fetchRoomHands.mockReset().mockResolvedValue({ items: [] });
    mocks.fetchRoomLeaderboard.mockReset().mockResolvedValue({ items: [] });
    mocks.fetchRoomHandReplay.mockReset();
  });

  it('keeps applying room and hand events while a drawer is open', async () => {
    const { default: RoomPage } = await import('./RoomPage.vue');
    const app = createApp(RoomPage);
    app.mount('#app');

    await vi.waitFor(() => {
      expect(mocks.handlers.has('room.updated')).toBe(true);
      expect(document.body.textContent).toContain('底池 100');
    });

    const scoreButton = document.querySelector<HTMLButtonElement>('[aria-label="打开战绩"]');
    expect(scoreButton).not.toBeNull();
    scoreButton?.click();
    await nextTick();
    expect(document.querySelector('.table-drawer-backdrop')).not.toBeNull();

    document.querySelector('.table-drawer')?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await nextTick();
    expect(document.querySelector('.table-drawer-backdrop')).not.toBeNull();

    const updatedRoom = roomFixture(2, '实时更新牌桌');
    const updatedHand = handFixture(2, 260, ['2s', '3h', '4d', '5c']);
    mocks.handlers.get('room.updated')?.({
      type: 'room.updated',
      roomId: 'room_1',
      roomVersion: 2,
      handId: 'hand_1',
      handVersion: 2,
      payload: { room: updatedRoom, hand: updatedHand },
    });
    await nextTick();

    expect(document.querySelector('.table-drawer-backdrop')).not.toBeNull();
    expect(document.body.textContent).toContain('实时更新牌桌');
    expect(document.body.textContent).toContain('底池 260');
    expect(document.querySelector('[aria-label="梅花 5"]')).not.toBeNull();

    document.querySelector('.table-drawer-backdrop')?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await nextTick();

    expect(document.querySelector('.table-drawer-backdrop')).toBeNull();
    expect(document.body.textContent).toContain('实时更新牌桌');
    expect(document.body.textContent).toContain('底池 260');
    expect(document.querySelector('[aria-label="梅花 5"]')).not.toBeNull();

    app.unmount();
  });

  it('keeps applying room and hand events while the buy-in modal is open', async () => {
    const initialRoom = roomFixture(1, '初始牌桌');
    initialRoom.seats = [
      { seatNo: 1, seatStatus: 'empty' },
      { seatNo: 2, seatStatus: 'occupied', userId: 'user_2', nickname: 'Player', buyInChips: 1000 },
    ];
    mocks.fetchRoom.mockResolvedValue(initialRoom);
    mocks.fetchCurrentRoomHand.mockResolvedValue({
      ...handFixture(1, 100, ['2s', '3h', '4d']),
      players: [{ seatNo: 2, name: 'Player', stack: 900, status: 'active', streetCommitted: 50, handCommitted: 100 }],
    });

    const { default: RoomPage } = await import('./RoomPage.vue');
    const app = createApp(RoomPage);
    app.mount('#app');

    await vi.waitFor(() => {
      expect(mocks.handlers.has('room.updated')).toBe(true);
      expect(document.body.textContent).toContain('底池 100');
    });

    document.querySelector<HTMLButtonElement>('[data-testid="table-seat-1"]')?.click();
    await nextTick();
    expect(document.querySelector('.buy-in-modal')).not.toBeNull();

    document.querySelector('.buy-in-modal')?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await nextTick();
    expect(document.querySelector('.buy-in-modal')).not.toBeNull();

    const updatedRoom = { ...initialRoom, version: 2, name: '模态框实时更新牌桌' };
    const updatedHand = {
      ...handFixture(2, 340, ['2s', '3h', '4d', '5c']),
      players: [{ seatNo: 2, name: 'Player', stack: 760, status: 'active', streetCommitted: 120, handCommitted: 240 }],
    };
    mocks.handlers.get('room.updated')?.({
      type: 'room.updated',
      roomId: 'room_1',
      roomVersion: 2,
      handId: 'hand_1',
      handVersion: 2,
      payload: { room: updatedRoom, hand: updatedHand },
    });
    await nextTick();

    expect(document.querySelector('.buy-in-modal')).not.toBeNull();
    expect(document.body.textContent).toContain('模态框实时更新牌桌');
    expect(document.body.textContent).toContain('底池 340');
    expect(document.querySelector('[aria-label="梅花 5"]')).not.toBeNull();

    document.querySelector('.modal-backdrop')?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    await nextTick();

    expect(document.querySelector('.buy-in-modal')).toBeNull();
    expect(document.body.textContent).toContain('模态框实时更新牌桌');
    expect(document.body.textContent).toContain('底池 340');
    expect(document.querySelector('[aria-label="梅花 5"]')).not.toBeNull();

    app.unmount();
  });
});
