import { describe, expect, it } from 'vitest';
import historySource from './pages/HistoryPage.vue?raw';

describe('history page contract', () => {
  it('keeps room history and personal record sections for multiplayer history browsing', () => {
    for (const token of [
      '房间最近牌局',
      'recentRoomHands',
      'winnerSummary',
      'potSummary',
      '个人战绩',
      'roomHistory',
      'item.winnerSummary',
      'formatDateTime(hand.completedAt)'
    ]) {
      expect(historySource).toContain(token);
    }
  });
});
