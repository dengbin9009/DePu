import { describe, expect, it, vi } from 'vitest';
import { createGame, createRoom, leaveRoom, submitAction } from './client';

describe('api client', () => {
  it('sends mockup-driven room configuration when creating a room', async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
      expect(_url).toBe('/api/rooms');
      expect(init?.method).toBe('POST');
      expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok_room');
      const body = JSON.parse(String(init?.body));
      expect(body).toEqual({
        ruleSetId: 'short-deck',
        name: '周末短牌局',
        mode: 'training',
        variant: 'short_holdem',
        ante: 20,
        minBuyIn: 2000,
        maxBuyIn: 8000,
        buyInCap: 60000,
        durationMinutes: 120,
        seatCount: 9,
        minPlayersToStart: 2
      });
      return new Response(JSON.stringify({
        id: 'room_1',
        inviteCode: 'R123456',
        ownerUserId: 'user_1',
        status: 'waiting',
        ruleSetId: body.ruleSetId,
        name: body.name,
        mode: body.mode,
        variant: body.variant,
        ante: body.ante,
        minBuyIn: body.minBuyIn,
        maxBuyIn: body.maxBuyIn,
        buyInCap: body.buyInCap,
        durationMinutes: body.durationMinutes,
        level: 1,
        seatCount: body.seatCount,
        minPlayersToStart: body.minPlayersToStart,
        members: [],
        seats: []
      }), { status: 201, headers: { 'Content-Type': 'application/json' } });
    });
    vi.stubGlobal('fetch', fetchMock);

    const room = await createRoom('tok_room', {
      ruleSetId: 'short-deck',
      name: '周末短牌局',
      mode: 'training',
      variant: 'short_holdem',
      ante: 20,
      minBuyIn: 2000,
      maxBuyIn: 8000,
      buyInCap: 60000,
      durationMinutes: 120,
      seatCount: 9,
      minPlayersToStart: 2
    });

    expect(room.name).toBe('周末短牌局');
    expect(room.minBuyIn).toBe(2000);
  });

  it('sends the selected betting structure when creating a game', async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
      expect(_url).toBe('/api/games');
      const body = JSON.parse(String(init?.body));
      expect(body.bettingStructure).toEqual({ type: 'ante', ante: 10, buttonBlind: 50 });
      return new Response(
        JSON.stringify({
          id: 'game-1',
          rulesetId: 'short-deck',
          bettingStructure: body.bettingStructure,
          stage: 'preflop',
          buttonSeat: 1,
          currentSeat: 2,
          board: [],
          seats: [],
          pots: [],
          legalActions: [],
          isReplay: false,
          debugLocked: false,
          version: 1
        }),
        { status: 201, headers: { 'Content-Type': 'application/json' } }
      );
    });
    vi.stubGlobal('fetch', fetchMock);

    const snapshot = await createGame({
      rulesetId: 'short-deck',
      buttonSeat: 1,
      bettingStructure: { type: 'ante', ante: 10, buttonBlind: 50 },
      dealMode: 'random',
      seats: [
        { seatNo: 1, name: 'BTN', stack: 1000 },
        { seatNo: 2, name: 'A', stack: 1000 }
      ]
    });

    expect(snapshot.bettingStructure.type).toBe('ante');
    expect(snapshot.isReplay).toBe(false);
    expect(snapshot.debugLocked).toBe(false);
  });

  it('leaves the current room membership through the room leave endpoint', async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
      expect(_url).toBe('/api/rooms/room_1/members/me');
      expect(init?.method).toBe('DELETE');
      expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok_room');
      return new Response(JSON.stringify({
        id: 'room_1',
        inviteCode: 'R123456',
        ownerUserId: 'user_2',
        status: 'waiting',
        members: [{ userId: 'user_2', nickname: '下一位', role: 'owner', joinedAt: '2026-07-10T00:00:00Z' }],
        seats: []
      }), { status: 200, headers: { 'Content-Type': 'application/json' } });
    });
    vi.stubGlobal('fetch', fetchMock);

    const room = await leaveRoom('tok_room', 'room_1');

    expect(room.ownerUserId).toBe('user_2');
    expect(room.members[0].role).toBe('owner');
  });

  it('sends the selected bet or raise amount when submitting an action', async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
      const body = JSON.parse(String(init?.body));
      expect(body).toMatchObject({
        seatNo: 4,
        type: 'raise',
        amount: 350,
        version: 3
      });
      return new Response(
        JSON.stringify({
          id: 'game-1',
          rulesetId: 'long-holdem',
          bettingStructure: { type: 'blinds', smallBlind: 50, bigBlind: 100 },
          stage: 'preflop',
          buttonSeat: 1,
          currentSeat: 1,
          currentBet: 350,
          minRaise: 250,
          board: [],
          seats: [],
          pots: [],
          legalActions: [],
          isReplay: false,
          debugLocked: true,
          version: 4
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      );
    });
    vi.stubGlobal('fetch', fetchMock);

    await submitAction(
      {
        id: 'game-1',
        rulesetId: 'long-holdem',
        bettingStructure: { type: 'blinds', smallBlind: 50, bigBlind: 100 },
        stage: 'preflop',
        buttonSeat: 1,
        currentSeat: 4,
        currentBet: 100,
        minRaise: 100,
        board: [],
        seats: [
          { seatNo: 1, name: '按钮', stack: 950, holeCards: [], status: 'active', streetCommitted: 0, handCommitted: 0 },
          { seatNo: 2, name: '小盲', stack: 950, holeCards: [], status: 'active', streetCommitted: 50, handCommitted: 50 },
          { seatNo: 3, name: '大盲', stack: 900, holeCards: [], status: 'active', streetCommitted: 100, handCommitted: 100 },
          { seatNo: 4, name: 'UTG', stack: 1000, holeCards: [], status: 'active', streetCommitted: 0, handCommitted: 0 }
        ],
        pots: [],
        legalActions: ['fold', 'call', 'raise'],
        isReplay: false,
        debugLocked: false,
        version: 3
      },
      'raise',
      350
    );
  });
});
