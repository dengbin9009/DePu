import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SocketEnvelope } from '../api/socketClient';
import type { RoomHandState, RoomResponse } from '../types/game';

const mocks = vi.hoisted(() => {
  const handlers = new Map<string, (message: SocketEnvelope) => void>();
  const connection = { connected: false };
  return {
    connection,
    handlers,
    createRoom: vi.fn(),
    fetchCurrentRoomHand: vi.fn(),
    fetchMe: vi.fn(),
    fetchRoom: vi.fn(),
    fetchRoomHands: vi.fn(),
    fetchRoomLeaderboard: vi.fn(),
    fetchUserHands: vi.fn(),
    fetchWallet: vi.fn(),
    joinRoom: vi.fn(),
    leaveRoom: vi.fn(),
    leaveSeat: vi.fn(),
    login: vi.fn(),
    recharge: vi.fn(),
    register: vi.fn(),
    takeSeat: vi.fn(),
    updateProfile: vi.fn(),
    socketClient: {
      connect: vi.fn(async () => {
        connection.connected = true;
      }),
      isConnected: vi.fn(() => connection.connected),
      send: vi.fn(async (_type: string, _roomId: string, _payload: unknown = {}) => ({})),
      on: vi.fn((type: string, handler: (message: SocketEnvelope) => void) => {
        handlers.set(type, handler);
        return () => handlers.delete(type);
      }),
      close: vi.fn(() => {
        connection.connected = false;
      }),
    },
  };
});

vi.mock('../api/client', () => ({
  createRoom: mocks.createRoom,
  fetchCurrentRoomHand: mocks.fetchCurrentRoomHand,
  fetchMe: mocks.fetchMe,
  fetchRoom: mocks.fetchRoom,
  fetchRoomHands: mocks.fetchRoomHands,
  fetchRoomLeaderboard: mocks.fetchRoomLeaderboard,
  fetchUserHands: mocks.fetchUserHands,
  fetchWallet: mocks.fetchWallet,
  joinRoom: mocks.joinRoom,
  leaveRoom: mocks.leaveRoom,
  leaveSeat: mocks.leaveSeat,
  login: mocks.login,
  recharge: mocks.recharge,
  register: mocks.register,
  takeSeat: mocks.takeSeat,
  updateProfile: mocks.updateProfile,
}));

vi.mock('../api/socketClient', () => ({
  createRoomSocketClient: vi.fn(() => mocks.socketClient),
}));

function roomFixture(version: number, ownerUserId: string): RoomResponse {
  return {
    id: 'room_1',
    version,
    inviteCode: 'ABC123',
    ownerUserId,
    status: 'playing',
    members: [],
    seats: [],
    seatCount: 6,
  };
}

function handFixture(handId: string, version: number, currentSeat: number): RoomHandState {
  return {
    roomId: 'room_1',
    handId,
    version,
    status: 'preflop',
    currentSeat,
    pot: 30,
    boardCards: [],
    players: [],
    availableActions: ['fold', 'call'],
  };
}

describe('useAppState socket event ordering', () => {
  beforeEach(() => {
    vi.resetModules();
    window.sessionStorage.clear();
    mocks.connection.connected = false;
    mocks.handlers.clear();
    mocks.socketClient.connect.mockReset().mockImplementation(async () => {
      mocks.connection.connected = true;
    });
    mocks.socketClient.isConnected.mockReset().mockImplementation(() => mocks.connection.connected);
    mocks.socketClient.send.mockReset().mockResolvedValue({});
    mocks.socketClient.on.mockReset().mockImplementation((type: string, handler: (message: SocketEnvelope) => void) => {
      mocks.handlers.set(type, handler);
      return () => mocks.handlers.delete(type);
    });
    mocks.socketClient.close.mockReset().mockImplementation(() => {
      mocks.connection.connected = false;
    });
    mocks.login.mockReset().mockResolvedValue({ token: 'token_1' });
    mocks.fetchMe.mockReset().mockResolvedValue({ id: 'user_1', username: 'owner', nickname: 'Owner' });
    mocks.fetchWallet.mockReset().mockResolvedValue({ balance: 1000 });
    mocks.fetchUserHands.mockReset().mockResolvedValue({ items: [] });
    mocks.createRoom.mockReset().mockResolvedValue(roomFixture(10, 'owner_initial'));
  });

  it('deduplicates concurrent start commands and converges after acknowledgement', async () => {
    let acknowledgeStart: (() => void) | undefined;
    const startAcknowledged = new Promise<void>((resolve) => {
      acknowledgeStart = resolve;
    });
    const startedHand = handFixture('hand_started', 1, 2);
    mocks.socketClient.send.mockImplementation(async (type: string) => {
      if (type === 'room.start_hand') await startAcknowledged;
      return {};
    });
    mocks.fetchRoom.mockResolvedValue({ ...roomFixture(11, 'owner_initial'), status: 'playing' });
    mocks.fetchCurrentRoomHand.mockResolvedValue(startedHand);
    mocks.fetchRoomHands.mockResolvedValue({ items: [] });

    const { useAppState } = await import('./useAppState');
    const state = useAppState();
    await state.doLogin('owner', 'password');
    await state.doCreateRoom({
      ruleSetId: 'texas-holdem',
      seatCount: 6,
      minPlayersToStart: 2,
    });

    const firstStart = state.doStartRoomHand();
    const duplicateStart = state.doStartRoomHand();
    await vi.waitFor(() => {
      expect(mocks.socketClient.send.mock.calls.filter(([type]) => type === 'room.start_hand')).toHaveLength(1);
    });

    acknowledgeStart?.();
    await Promise.all([firstStart, duplicateStart]);

    expect(mocks.fetchRoom).toHaveBeenCalledTimes(1);
    expect(mocks.fetchCurrentRoomHand).toHaveBeenCalledTimes(1);
    expect(state.room.value?.status).toBe('playing');
    expect(state.currentRoomHand.value).toEqual(startedHand);
  });

  it('ignores stale room versions, previous-hand events, and older versions of the current hand', async () => {
    const { useAppState } = await import('./useAppState');
    const state = useAppState();
    await state.doLogin('owner', 'password');
    await state.doCreateRoom({
      ruleSetId: 'texas-holdem',
      seatCount: 6,
      minPlayersToStart: 2,
    });

    const emit = (message: SocketEnvelope) => {
      const handler = mocks.handlers.get(message.type);
      expect(handler, `missing handler for ${message.type}`).toBeTypeOf('function');
      handler?.(message);
    };

    emit({
      type: 'room.updated',
      roomId: 'room_1',
      roomVersion: 9,
      payload: { room: roomFixture(9, 'owner_older_than_http') },
    });
    expect(state.room.value?.ownerUserId).toBe('owner_initial');

    emit({
      type: 'room.updated',
      roomId: 'room_1',
      roomVersion: 12,
      payload: { room: roomFixture(12, 'owner_current') },
    });
    emit({
      type: 'room.updated',
      roomId: 'room_1',
      roomVersion: 11,
      payload: { room: roomFixture(11, 'owner_stale') },
    });
    expect(state.room.value?.ownerUserId).toBe('owner_current');

    emit({
      type: 'hand.started',
      roomId: 'room_1',
      roomVersion: 13,
      handId: 'hand_new',
      handVersion: 2,
      payload: { hand: handFixture('hand_new', 2, 2) },
    });
    emit({
      type: 'hand.updated',
      roomId: 'room_1',
      roomVersion: 13,
      handId: 'hand_new',
      handVersion: 4,
      payload: { hand: handFixture('hand_new', 4, 4) },
    });
    emit({
      type: 'hand.updated',
      roomId: 'room_1',
      roomVersion: 13,
      handId: 'hand_new',
      handVersion: 3,
      payload: { hand: handFixture('hand_new', 3, 3) },
    });
    expect(state.currentRoomHand.value).toMatchObject({ handId: 'hand_new', version: 4, currentSeat: 4 });

    emit({
      type: 'hand.updated',
      roomId: 'room_1',
      roomVersion: 13,
      handId: 'hand_previous',
      handVersion: 99,
      payload: { hand: handFixture('hand_previous', 99, 1) },
    });
    emit({
      type: 'hand.settled',
      roomId: 'room_1',
      roomVersion: 13,
      handId: 'hand_previous',
      handVersion: 100,
      payload: { hand: { handId: 'hand_previous' } },
    });
    expect(state.currentRoomHand.value).toMatchObject({ handId: 'hand_new', version: 4, currentSeat: 4 });
  });

  it('handles duplicate events and settlement interleaved with a new hand exactly once', async () => {
    mocks.fetchRoomHands.mockResolvedValue({ items: [] });
    const { useAppState } = await import('./useAppState');
    const state = useAppState();
    await state.doLogin('owner', 'password');
    await state.doCreateRoom({
      ruleSetId: 'texas-holdem',
      seatCount: 6,
      minPlayersToStart: 2,
    });

    const emit = (message: SocketEnvelope) => {
      const handler = mocks.handlers.get(message.type);
      expect(handler, `missing handler for ${message.type}`).toBeTypeOf('function');
      handler?.(message);
    };

    const settled = {
      type: 'hand.settled',
      roomId: 'room_1',
      roomVersion: 11,
      handId: 'hand_old',
      handVersion: 5,
      payload: { hand: { handId: 'hand_old' } },
    } satisfies SocketEnvelope;
    emit({
      type: 'room.snapshot',
      roomId: 'room_1',
      roomVersion: 10,
      handId: 'hand_old',
      handVersion: 4,
      payload: {
        room: roomFixture(10, 'owner_initial'),
        hand: handFixture('hand_old', 4, 2),
      },
    });
    emit(settled);
    emit(settled);
    await vi.waitFor(() => expect(mocks.fetchRoomHands).toHaveBeenCalledTimes(1));

    const started = {
      type: 'hand.started',
      roomId: 'room_1',
      roomVersion: 12,
      handId: 'hand_new',
      handVersion: 1,
      payload: { hand: handFixture('hand_new', 1, 3) },
    } satisfies SocketEnvelope;
    emit(started);
    emit(started);
    emit(settled);

    expect(state.currentRoomHand.value).toMatchObject({ handId: 'hand_new', version: 1, currentSeat: 3 });
    expect(mocks.fetchRoomHands).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['waiting', null],
    ['playing', handFixture('hand_reconnected', 7, 4)],
    ['settlement window', null],
  ])('restores the authoritative %s lifecycle snapshot after disconnect', async (status, hand) => {
    const { useAppState } = await import('./useAppState');
    const state = useAppState();
    await state.doLogin('owner', 'password');
    await state.doCreateRoom({
      ruleSetId: 'texas-holdem',
      seatCount: 6,
      minPlayersToStart: 2,
    });

    mocks.connection.connected = false;
    await state.connectRoomSocket('room_1');
    const snapshotRoom = {
      ...roomFixture(15, 'owner_after_reconnect'),
      status: status === 'playing' ? 'playing' : 'waiting',
    } as RoomResponse;
    mocks.handlers.get('room.snapshot')?.({
      type: 'room.snapshot',
      roomId: 'room_1',
      roomVersion: 15,
      handId: hand?.handId,
      handVersion: hand?.version,
      payload: {
        room: snapshotRoom,
        hand,
        presence: [],
        recentActionLog: [],
        recentChatMessages: [],
        leaderboard: [],
      },
    });

    expect(state.room.value).toMatchObject({ version: 15, status: snapshotRoom.status });
    expect(state.currentRoomHand.value).toEqual(hand);
  });

  it('replaces every room-scoped state from an authoritative room snapshot', async () => {
    const { useAppState } = await import('./useAppState');
    const state = useAppState();
    await state.doLogin('owner', 'password');
    await state.doCreateRoom({
      ruleSetId: 'texas-holdem',
      seatCount: 6,
      minPlayersToStart: 2,
    });

    const emit = (message: SocketEnvelope) => {
      const handler = mocks.handlers.get(message.type);
      expect(handler, `missing handler for ${message.type}`).toBeTypeOf('function');
      handler?.(message);
    };

    emit({
      type: 'room.snapshot',
      roomId: 'room_1',
      roomVersion: 11,
      handId: 'hand_old',
      handVersion: 3,
      payload: {
        room: roomFixture(11, 'owner_before_reconnect'),
        hand: handFixture('hand_old', 3, 2),
        presence: [{ userId: 'user_2', seatNo: 2, status: 'online' }],
        recentActionLog: [{ seq: 1, kind: 'action', source: 'player', createdAt: '2026-07-11T00:00:00Z' }],
        recentChatMessages: [{ id: 'chat_1', kind: 'text', text: 'old', userId: 'user_2', nickname: 'Old', createdAt: '2026-07-11T00:00:00Z' }],
        leaderboard: [{ userId: 'user_2', nickname: 'Old', handsPlayed: 1, handsWon: 1, netProfit: 10, biggestPotWon: 30, lastSettledAt: '2026-07-11T00:00:00Z' }],
      },
    });

    emit({
      type: 'room.snapshot',
      roomId: 'room_1',
      roomVersion: 12,
      payload: {
        room: { ...roomFixture(12, 'owner_after_reconnect'), status: 'waiting' },
      },
    });

    expect(state.room.value).toMatchObject({ version: 12, ownerUserId: 'owner_after_reconnect', status: 'waiting' });
    expect(state.currentRoomHand.value).toBeNull();
    expect(state.roomPresence.value).toEqual([]);
    expect(state.actionLog.value).toEqual([]);
    expect(state.chatMessages.value).toEqual([]);
    expect(state.roomLeaderboard.value).toEqual([]);
  });

  it('resubscribes for a fresh snapshot without replaying an unacknowledged action', async () => {
    let actionAttempts = 0;
    mocks.socketClient.send.mockImplementation(async (type: string) => {
      if (type === 'room.action' && actionAttempts++ === 0) {
        mocks.connection.connected = false;
        throw new Error('socket disconnected before acknowledgement');
      }
      return {};
    });
    const { useAppState } = await import('./useAppState');
    const state = useAppState();
    await state.doLogin('owner', 'password');
    await state.doCreateRoom({
      ruleSetId: 'texas-holdem',
      seatCount: 6,
      minPlayersToStart: 2,
    });

    await expect(state.doRoomAction('call')).rejects.toThrow('socket disconnected before acknowledgement');
    expect(mocks.socketClient.send.mock.calls.map(([type]) => type)).toEqual(['room.subscribe', 'room.action']);

    await expect(state.doRoomAction('call')).rejects.toThrow('socket reconnected; retry after the latest room snapshot');
    expect(mocks.socketClient.connect).toHaveBeenCalledTimes(2);
    expect(mocks.socketClient.send.mock.calls.map(([type]) => type)).toEqual([
      'room.subscribe',
      'room.action',
      'room.subscribe',
    ]);

    mocks.handlers.get('room.snapshot')?.({
      type: 'room.snapshot',
      roomId: 'room_1',
      roomVersion: 11,
      handId: 'hand_latest',
      handVersion: 2,
      payload: {
        room: roomFixture(11, 'owner_initial'),
        hand: handFixture('hand_latest', 2, 2),
      },
    });
    expect(state.currentRoomHand.value?.handId).toBe('hand_latest');

    await state.doRoomAction('call');
    expect(mocks.socketClient.send.mock.calls.map(([type]) => type)).toEqual([
      'room.subscribe',
      'room.action',
      'room.subscribe',
      'room.action',
    ]);
  });
});
