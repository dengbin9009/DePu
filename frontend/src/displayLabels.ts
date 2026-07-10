import type { BettingStructure, BettingStructureType } from './types/game';

const actionLabels: Record<string, string> = {
  fold: '弃牌',
  check: '过牌',
  call: '跟注',
  bet: '下注',
  raise: '加注',
  all_in: '全下',
  deal: '发牌',
  debug_set_cards: '指定调试牌',
  forced_bet: '强制投入',
  settle: '结算'
};

const stageLabels: Record<string, string> = {
  waiting: '等待开始',
  preflop: '翻牌前',
  flop: '翻牌圈',
  turn: '转牌圈',
  river: '河牌圈',
  showdown: '摊牌',
  finished: '已结束'
};

const statusLabels: Record<string, string> = {
  active: '牌局中',
  folded: '已弃牌',
  all_in: '已全下',
  out: '已出局'
};

const walletTransactionLabels: Record<string, string> = {
  buy_in: '买入冻结',
  leave_refund: '离座返还',
  hand_result: '牌局结算',
  recharge: '模拟充值'
};

const bettingTypeLabels: Record<BettingStructureType, string> = {
  blinds: '小盲/大盲',
  ante: '前注 + 按钮盲注'
};

const handClassLabels: Record<string, string> = {
  straight_flush: '同花顺',
  four_of_a_kind: '四条',
  full_house: '葫芦',
  flush: '同花',
  straight: '顺子',
  three_of_a_kind: '三条',
  two_pair: '两对',
  one_pair: '一对',
  high_card: '高牌'
};

export function actionLabel(action: string): string {
  return actionLabels[action] ?? action;
}

export function stageLabel(stage: string): string {
  return stageLabels[stage] ?? stage;
}

export function statusLabel(status: string): string {
  return statusLabels[status] ?? status;
}

export function bettingTypeLabel(type: BettingStructureType): string {
  return bettingTypeLabels[type] ?? type;
}

export function bettingStructureLabel(betting?: BettingStructure | null): string {
  if (!betting) return '未开始牌局';
  if (betting.type === 'ante') return `前注 ${betting.ante} · 按钮盲注 ${betting.buttonBlind}`;
  return `小盲 ${betting.smallBlind} · 大盲 ${betting.bigBlind}`;
}

export function handClassLabel(handClass: string): string {
  return handClassLabels[handClass] ?? handClass;
}

export function potLabel(potId: string): string {
  if (potId === 'main') return '主池';
  const match = /^pot-(\d+)$/.exec(potId);
  if (match) return `底池 ${match[1]}`;
  return potId;
}

export function walletTransactionLabel(type: string): string {
  return walletTransactionLabels[type] ?? type;
}
