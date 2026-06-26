import { describe, expect, it } from 'vitest';
import appSource from './App.vue?raw';

describe('history page contract', () => {
  it('keeps room history and personal record sections for multiplayer history browsing', () => {
    for (const token of [
      '房间最近牌局',
      'recentRoomHands',
      'winnerSummary',
      'potSummary',
      'participant.handCommitted',
      'participant.awardAmount',
      'participant.profit',
      '个人战绩',
      'roomHistory',
      'item.winnerSummary',
      'formatDateTime(hand.completedAt)'
    ]) {
      expect(appSource).toContain(token);
    }
  });
});
