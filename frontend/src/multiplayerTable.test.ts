import { describe, expect, it } from 'vitest';
import appSource from './App.vue?raw';

describe('multiplayer table contract', () => {
  it('keeps multiplayer room controls and player perspective hints in the main table page', () => {
    for (const token of [
      'doCreateRoom',
      'doJoinRoom',
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
      '刷新当前手牌'
    ]) {
      expect(appSource).toContain(token);
    }
  });
});
