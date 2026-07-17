import { describe, expect, it } from 'vitest';
import { cardFaceVisual, cardRank, holeCardVisuals, isRedCard, seatPositions, tableVisualState } from './pokerVisuals';
import type { GameSnapshot } from './types/game';

function snapshotWithSeats(count: number): GameSnapshot {
  return {
    id: 'game-1',
    rulesetId: 'long-holdem',
    bettingStructure: { type: 'blinds', smallBlind: 50, bigBlind: 100 },
    stage: 'preflop',
    buttonSeat: 1,
    currentSeat: 2,
    currentBet: 100,
    minRaise: 100,
    board: [],
    seats: Array.from({ length: count }, (_, i) => ({
      seatNo: i + 1,
      name: `P${i + 1}`,
      stack: 1000,
      holeCards: ['As', 'Kh'],
      status: 'active',
      streetCommitted: 0,
      handCommitted: 0
    })),
    pots: [{ id: 'pot-1', amount: 150, eligibleSeats: [1, 2] }],
    legalActions: [],
    isReplay: false,
    debugLocked: false,
    version: 1
  };
}

describe('pokerVisuals', () => {
  it('uses two non-leaking code-native backs for unavailable or invalid private cards', () => {
    expect(holeCardVisuals(null)).toEqual([
      { kind: 'back', ariaLabel: '牌背' },
      { kind: 'back', ariaLabel: '牌背' }
    ]);
    expect(holeCardVisuals(['As'])).toEqual([
      expect.objectContaining({ kind: 'face', card: 'As', ariaLabel: '黑桃 A' }),
      { kind: 'back', ariaLabel: '牌背' }
    ]);

    const hiddenCards = holeCardVisuals(['hidden-card-text', 'Kh']);
    expect(hiddenCards).toEqual([
      { kind: 'back', ariaLabel: '牌背' },
      expect.objectContaining({ kind: 'face', card: 'Kh', ariaLabel: '红桃 K' })
    ]);
    expect(JSON.stringify(hiddenCards)).not.toContain('hidden-card-text');
    expect(JSON.stringify(hiddenCards)).not.toContain('image');
  });

  it('maps formal table card codes to readable face visuals', () => {
    expect(cardFaceVisual(' As ')).toEqual({
      valid: true,
      rankLabel: 'A',
      suit: 'spades',
      suitSymbol: '♠',
      color: 'black',
      colorClass: 'black',
      ariaLabel: '黑桃 A'
    });
    expect(cardFaceVisual('10H')).toMatchObject({ rankLabel: '10', suit: 'hearts', suitSymbol: '♥', color: 'red', colorClass: 'red' });
    expect(cardFaceVisual('Td')).toMatchObject({ rankLabel: '10', suit: 'diamonds', suitSymbol: '♦', color: 'red', colorClass: 'red' });
    expect(cardFaceVisual('qc')).toMatchObject({ rankLabel: 'Q', suit: 'clubs', suitSymbol: '♣', color: 'black', colorClass: 'black' });
  });

  it('degrades invalid card codes without exposing misleading rank or suit data', () => {
    expect(cardFaceVisual('joker')).toEqual({
      valid: false,
      rankLabel: '?',
      suit: 'unknown',
      suitSymbol: '•',
      color: 'neutral',
      colorClass: 'invalid',
      ariaLabel: '无效牌'
    });
    expect(cardFaceVisual('')).toEqual(cardFaceVisual('1x'));
  });

  it('parses card rank and color helpers', () => {
    expect(cardRank('Ah')).toBe('A');
    expect(isRedCard('Ah')).toBe(true);
    expect(isRedCard('Ks')).toBe(false);
  });

  it('calculates stable positions for 2 to 10 players', () => {
    for (let count = 2; count <= 10; count += 1) {
      const positions = seatPositions(snapshotWithSeats(count).seats, 2, 1);
      expect(positions).toHaveLength(count);
      expect(positions.every((seat) => seat.x >= 12 && seat.x <= 88 && seat.y >= 12 && seat.y <= 88)).toBe(true);
      expect(positions.some((seat) => seat.active)).toBe(true);
    }
  });

  it('marks compact table state and replay transition', () => {
    const visual = tableVisualState(snapshotWithSeats(9), { replayTransition: true });
    expect(visual.potTotal).toBe(150);
    expect(visual.replayTransition).toBe(true);
    expect(visual.seatPositions.every((seat) => seat.compact)).toBe(true);
  });
});
