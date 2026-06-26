import { describe, expect, it } from 'vitest';
import {
  actionLabel,
  bettingStructureLabel,
  bettingTypeLabel,
  handClassLabel,
  potLabel,
  stageLabel,
  statusLabel
} from './displayLabels';

describe('displayLabels', () => {
  it('translates player-facing poker actions to Chinese', () => {
    expect(actionLabel('fold')).toBe('弃牌');
    expect(actionLabel('check')).toBe('过牌');
    expect(actionLabel('call')).toBe('跟注');
    expect(actionLabel('raise')).toBe('加注');
    expect(actionLabel('all_in')).toBe('全下');
    expect(actionLabel('forced_bet')).toBe('强制投入');
  });

  it('translates stages and seat statuses to Chinese', () => {
    expect(stageLabel('preflop')).toBe('翻牌前');
    expect(stageLabel('flop')).toBe('翻牌圈');
    expect(stageLabel('showdown')).toBe('摊牌');
    expect(statusLabel('active')).toBe('牌局中');
    expect(statusLabel('folded')).toBe('已弃牌');
    expect(statusLabel('all_in')).toBe('已全下');
  });

  it('formats betting labels in Chinese while preserving numeric values', () => {
    expect(bettingTypeLabel('blinds')).toBe('小盲/大盲');
    expect(bettingTypeLabel('ante')).toBe('前注 + 按钮盲注');
    expect(bettingStructureLabel({ type: 'blinds', smallBlind: 50, bigBlind: 100 })).toBe('小盲 50 · 大盲 100');
    expect(bettingStructureLabel({ type: 'ante', ante: 10, buttonBlind: 50 })).toBe('前注 10 · 按钮盲注 50');
  });

  it('translates showdown hand classes and pot ids to Chinese', () => {
    expect(handClassLabel('straight_flush')).toBe('同花顺');
    expect(handClassLabel('full_house')).toBe('葫芦');
    expect(handClassLabel('high_card')).toBe('高牌');
    expect(potLabel('pot-1')).toBe('底池 1');
    expect(potLabel('main')).toBe('主池');
  });
});
