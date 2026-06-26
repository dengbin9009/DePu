import { describe, expect, it, vi } from 'vitest';
import { createGame, submitAction } from './client';

describe('api client', () => {
  it('sends the selected betting structure when creating a game', async () => {
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
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
