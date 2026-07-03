import { describe, expect, it } from 'vitest';
import appStateSource from './composables/useAppState.ts?raw';
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

  it('renders archived cards as poker card visuals instead of raw card codes', () => {
    for (const token of [
      'cardImagePath',
      'cardAltText',
      'history-card-row',
      'history-playing-card',
      '公共牌',
      '手牌',
      '最佳牌'
    ]) {
      expect(historySource).toContain(token);
    }

    expect(historySource).not.toContain("hand.boardCards.join(' ')");
    expect(historySource).not.toContain('{{ card }}');
  });

  it('auto-loads room archived hands from personal history when no room context exists', () => {
    for (const token of ['refreshHistoryDetails', 'await refreshHistoryDetails()']) {
      expect(historySource).toContain(token);
    }

    for (const token of ['function refreshHistoryDetails', 'roomHistory.value[0]?.roomId', 'fetchRoomHands(token.value, historyRoomId)']) {
      expect(appStateSource).toContain(token);
    }
  });
});
