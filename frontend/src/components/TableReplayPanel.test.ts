import { createApp, nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { HandReplayResponse, RoomHandHistoryRecord } from '../types/game';

const mocks = vi.hoisted(() => ({
  fetchRoomHands: vi.fn(),
  fetchRoomHandReplay: vi.fn(),
}));

vi.mock('../api/client', () => ({
  fetchRoomHands: mocks.fetchRoomHands,
  fetchRoomHandReplay: mocks.fetchRoomHandReplay,
}));

const latestHand: RoomHandHistoryRecord = {
  handId: 'hand_2',
  roomId: 'room_1',
  handNo: 2,
  ruleSetId: 'long-holdem',
  completedAt: '2026-07-11T02:00:00Z',
  winnerSummary: 'Hero 赢得 600',
  potSummary: '主池 600',
  boardCards: ['As', 'Kh', '10d', '2c', '7s'],
  totalPot: 600,
  participants: [],
};

const olderHand: RoomHandHistoryRecord = {
  ...latestHand,
  handId: 'hand_1',
  handNo: 1,
  completedAt: '2026-07-11T01:00:00Z',
  winnerSummary: 'Villain 赢得 300',
  potSummary: '主池 300',
  totalPot: 300,
};

const replay: HandReplayResponse = {
  handId: latestHand.handId,
  roomId: latestHand.roomId,
  gameId: 'game_2',
  steps: [{
    seq: 0,
    stage: 'preflop',
    currentSeat: 1,
    boardCards: [],
    pot: 0,
    players: [],
  }],
};

const detailedReplay: HandReplayResponse = {
  handId: latestHand.handId,
  roomId: latestHand.roomId,
  gameId: 'game_2',
  steps: [
    {
      seq: 0,
      stage: 'preflop',
      currentSeat: 1,
      boardCards: [],
      pot: 30,
      players: [
        {
          seatNo: 1,
          nickname: 'Hero',
          stack: 980,
          status: 'active',
          handCommitted: 20,
          holeCards: null,
        },
      ],
    },
    {
      seq: 7,
      stage: 'flop',
      currentSeat: 2,
      boardCards: ['As', 'Kh', '10d'],
      pot: 70,
      action: {
        type: 'call',
        seatNo: 2,
        amount: 20,
      },
      players: [
        {
          seatNo: 1,
          nickname: 'Hero',
          stack: 960,
          status: 'active',
          handCommitted: 40,
          holeCards: ['Ah', 'Ad'],
        },
      ],
    },
  ],
};

const olderReplay: HandReplayResponse = {
  handId: olderHand.handId,
  roomId: olderHand.roomId,
  gameId: 'game_1',
  steps: [{
    seq: 0,
    stage: 'finished',
    currentSeat: 0,
    boardCards: olderHand.boardCards,
    pot: olderHand.totalPot,
    players: [],
  }],
};

async function mountPanel() {
  const { default: TableReplayPanel } = await import('./TableReplayPanel.vue');
  const app = createApp(TableReplayPanel, {
    roomId: 'room_1',
    hands: [],
    token: 'token_1',
  });
  app.mount('#app');
  return app;
}

describe('TableReplayPanel', () => {
  beforeEach(() => {
    document.body.innerHTML = '<div id="app"></div>';
    mocks.fetchRoomHands.mockReset();
    mocks.fetchRoomHandReplay.mockReset();
  });

  it('refreshes room history on open and automatically loads the latest hand', async () => {
    mocks.fetchRoomHands.mockResolvedValue({ items: [latestHand, olderHand] });
    mocks.fetchRoomHandReplay.mockResolvedValue(replay);

    const app = await mountPanel();

    await vi.waitFor(() => {
      expect(mocks.fetchRoomHands).toHaveBeenCalledWith('token_1', 'room_1');
      expect(mocks.fetchRoomHandReplay).toHaveBeenCalledWith('token_1', 'room_1', 'hand_2');
    });

    expect(document.body.textContent).toContain('牌谱回顾 - 2');
    expect(document.body.textContent).toContain('Hero 赢得 600');
    expect(document.body.textContent).toContain('底池 600');
    const historyCards = Array.from(document.querySelectorAll<HTMLElement>('.replay-hand-list li:first-child .replay-hand-board .replay-card-face'));
    expect(historyCards).toHaveLength(5);
    expect(historyCards.map((card) => card.getAttribute('aria-label'))).toEqual([
      '黑桃 A',
      '红桃 K',
      '方块 10',
      '梅花 2',
      '黑桃 7',
    ]);
    expect(historyCards.map((card) => card.dataset.card)).toEqual(['As', 'Kh', '10d', '2c', '7s']);
    expect(document.body.textContent).not.toContain('As Kh 10d 2c 7s');

    app.unmount();
  });

  it('shows a clear empty state after history loads', async () => {
    mocks.fetchRoomHands.mockResolvedValue({ items: [] });

    const app = await mountPanel();

    await vi.waitFor(() => expect(mocks.fetchRoomHands).toHaveBeenCalledOnce());
    expect(document.body.textContent).toContain('暂无已结算牌谱');
    expect(mocks.fetchRoomHandReplay).not.toHaveBeenCalled();

    app.unmount();
  });

  it('shows a retry action when history loading fails', async () => {
    mocks.fetchRoomHands
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce({ items: [latestHand] });
    mocks.fetchRoomHandReplay.mockResolvedValue(replay);

    const app = await mountPanel();

    await vi.waitFor(() => expect(document.body.textContent).toContain('牌谱历史加载失败'));
    const retryButton = Array.from(document.querySelectorAll('button')).find((button) => button.textContent?.includes('重试'));
    expect(retryButton).toBeDefined();

    retryButton?.click();
    await nextTick();

    await vi.waitFor(() => {
      expect(mocks.fetchRoomHands).toHaveBeenCalledTimes(2);
      expect(mocks.fetchRoomHandReplay).toHaveBeenCalledWith('token_1', 'room_1', 'hand_2');
    });
    expect(document.body.textContent).toContain('Hero 赢得 600');

    app.unmount();
  });

  it('shows authoritative replay step details and resets navigation when switching hands', async () => {
    mocks.fetchRoomHands.mockResolvedValue({ items: [latestHand, olderHand] });
    mocks.fetchRoomHandReplay.mockImplementation(async (_token, _roomId, handId) => (
      handId === latestHand.handId ? detailedReplay : olderReplay
    ));

    const app = await mountPanel();

    await vi.waitFor(() => expect(document.body.textContent).toContain('步骤 #0 · 1/2'));
    expect(document.body.textContent).toContain('阶段 翻牌前');
    expect(document.body.textContent).toContain('上一动作：手牌开始');
    expect(document.body.textContent).toContain('底池 30');
    expect(document.body.textContent).toContain('公共牌：暂无');
    expect(document.body.textContent).toContain('#1 Hero · 牌局中 · 剩余 980 · 已投 20');
    expect(document.body.textContent).not.toContain('手牌 Ah Ad');
    expect(document.querySelectorAll('.replay-step-board .replay-card-face')).toHaveLength(0);
    expect(document.querySelectorAll('.replay-player-hole-cards .replay-card-face')).toHaveLength(0);

    const nextActionButton = Array.from(document.querySelectorAll('button')).find((button) => button.textContent === '下一动作');
    nextActionButton?.click();
    await nextTick();

    expect(document.body.textContent).toContain('步骤 #7 · 2/2');
    expect(document.body.textContent).toContain('阶段 翻牌圈');
    expect(document.body.textContent).toContain('上一动作：#2 跟注 20');
    expect(document.body.textContent).toContain('底池 70');
    const boardCards = Array.from(document.querySelectorAll<HTMLElement>('.replay-step-board .replay-card-face'));
    const holeCards = Array.from(document.querySelectorAll<HTMLElement>('.replay-player-hole-cards .replay-card-face'));
    expect(boardCards.map((card) => card.dataset.card)).toEqual(['As', 'Kh', '10d']);
    expect(holeCards.map((card) => card.dataset.card)).toEqual(['Ah', 'Ad']);
    expect(document.body.textContent).not.toContain('As Kh 10d');
    expect(document.body.textContent).not.toContain('Ah Ad');

    const olderHandButton = Array.from(document.querySelectorAll<HTMLButtonElement>('.replay-hand-list button')).find((button) => button.textContent?.includes('第1手'));
    olderHandButton?.click();

    await vi.waitFor(() => {
      expect(mocks.fetchRoomHandReplay).toHaveBeenCalledWith('token_1', 'room_1', 'hand_1');
      expect(document.body.textContent).toContain('牌谱回顾 - 1');
      expect(document.body.textContent).toContain('步骤 #0 · 1/1');
    });
    expect(document.body.textContent).not.toContain('上一动作：#2 跟注 20');

    app.unmount();
  });

  it('renders only hole cards returned by the current replay step', async () => {
    const archivedHand: RoomHandHistoryRecord = {
      ...latestHand,
      participants: [{
        userId: 'user_1',
        nickname: 'Hero',
        seatNo: 1,
        profit: 600,
        awardAmount: 600,
        handCommitted: 300,
        resultType: 'won',
        holeCards: ['Ah', 'Ad'],
        bestCards: ['Ah', 'Ad', 'As', 'Kh', '10d'],
        handClass: 'three_of_a_kind',
      }],
    };
    mocks.fetchRoomHands.mockResolvedValue({ items: [archivedHand] });
    mocks.fetchRoomHandReplay.mockResolvedValue({
      ...detailedReplay,
      steps: [detailedReplay.steps[0]],
    });

    const app = await mountPanel();

    await vi.waitFor(() => expect(document.body.textContent).toContain('步骤 #0 · 1/1'));
    expect(document.body.textContent).toContain('#1 Hero · 牌局中');
    expect(document.body.textContent).not.toContain('手牌 Ah Ad');

    app.unmount();
  });

  it('clears the previous hand steps while switching and keeps them cleared when the new replay fails', async () => {
    let rejectOlderReplay: ((reason?: unknown) => void) | undefined;
    const olderReplayRequest = new Promise<HandReplayResponse>((_resolve, reject) => {
      rejectOlderReplay = reject;
    });
    mocks.fetchRoomHands.mockResolvedValue({ items: [latestHand, olderHand] });
    mocks.fetchRoomHandReplay.mockImplementation((_token, _roomId, handId) => (
      handId === latestHand.handId ? Promise.resolve(detailedReplay) : olderReplayRequest
    ));

    const app = await mountPanel();

    await vi.waitFor(() => expect(document.body.textContent).toContain('步骤 #0 · 1/2'));
    expect(document.body.textContent).toContain('#1 Hero · 牌局中');

    const olderHandButton = Array.from(document.querySelectorAll<HTMLButtonElement>('.replay-hand-list button')).find((button) => button.textContent?.includes('第1手'));
    olderHandButton?.click();
    await nextTick();

    expect(document.body.textContent).toContain('牌谱回顾 - 1');
    expect(document.body.textContent).not.toContain('步骤 #0 · 1/2');
    expect(document.body.textContent).not.toContain('#1 Hero · 牌局中');

    rejectOlderReplay?.(new Error('replay unavailable'));

    await vi.waitFor(() => expect(document.body.textContent).toContain('牌谱回放加载失败'));
    expect(document.body.textContent).toContain('牌谱回顾 - 1');
    expect(document.body.textContent).not.toContain('步骤 #0 · 1/2');
    expect(document.body.textContent).not.toContain('#1 Hero · 牌局中');

    app.unmount();
  });
});
