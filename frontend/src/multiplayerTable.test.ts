import { describe, expect, it } from 'vitest';
import roomSource from './pages/RoomPage.vue?raw';

describe('multiplayer table contract', () => {
  it('keeps multiplayer room controls and player perspective hints in the room page', () => {
    for (const token of [
      'doTakeSeat',
      'doLeaveSeat',
      'doStartRoomHand',
      'refreshCurrentRoomHand',
      'doRoomAction',
      'currentRoomHand',
      'myRoomSeat',
      'myRoomHandPlayer',
      'isMyTurn',
      '现在轮到我操作',
      '我当前还未入座',
      '当前不是我的回合',
      '房主开局',
      '刷新当前手牌',
      '当前战绩',
      '观众（',
      '旁观 / 坐下 / 已入座'
    ]) {
      expect(roomSource).toContain(token);
    }
  });
});
