import { describe, expect, it } from 'vitest';
import { calculateBetAmountBounds, clampBetAmount, presetBetAmount } from './bettingControls';
import type { GameSnapshot } from './types/game';

function snapshot(overrides: Partial<GameSnapshot> = {}): GameSnapshot {
  return {
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
    pots: [{ id: 'pot-1', amount: 150, eligibleSeats: [1, 2, 3, 4] }],
    legalActions: ['fold', 'call', 'raise'],
    isReplay: false,
    debugLocked: false,
    version: 3,
    ...overrides
  };
}

describe('bettingControls', () => {
  it('calculates raise slider bounds from the backend betting state', () => {
    expect(calculateBetAmountBounds(snapshot())).toEqual({
      action: 'raise',
      min: 200,
      max: 1000,
      step: 10,
      defaultAmount: 200,
      fullMin: 200,
      isShortAllIn: false
    });
  });

  it('allows a short all-in raise when remaining chips cannot reach a full min raise', () => {
    const game = snapshot({
      currentBet: 300,
      minRaise: 200,
      seats: [
        { seatNo: 4, name: 'UTG', stack: 100, holeCards: [], status: 'active', streetCommitted: 250, handCommitted: 250 }
      ]
    });

    expect(calculateBetAmountBounds(game)).toMatchObject({
      action: 'raise',
      min: 350,
      max: 350,
      fullMin: 500,
      isShortAllIn: true
    });
  });

  it('clamps manual input and quick presets inside the legal slider range', () => {
    const game = snapshot();

    expect(clampBetAmount(game, 9999)).toBe(1000);
    expect(clampBetAmount(game, 50)).toBe(200);
    expect(presetBetAmount(game, 'min')).toBe(200);
    expect(presetBetAmount(game, 'all_in')).toBe(1000);
  });
});
