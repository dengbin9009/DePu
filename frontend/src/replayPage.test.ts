import { describe, expect, it } from 'vitest';
import clientSource from './api/client.ts?raw';
import routerSource from './router/index.ts?raw';
import replaySource from './pages/HandReplayPage.vue?raw';

describe('formal hand replay page contract', () => {
  it('uses the formal multiplayer replay endpoint and route', () => {
    expect(clientSource).toContain('fetchRoomHandReplay');
    expect(clientSource).toContain('/api/rooms/${roomId}/hands/${handId}/replay');
    expect(routerSource).toContain('/room/:roomId/hands/:handId/replay');
    expect(routerSource).toContain('HandReplayPage');
  });

  it('renders step controls and public replay state', () => {
    for (const token of ['replayStep', '下一步', '上一步', '播放', '暂停', '公共牌', '底池']) {
      expect(replaySource).toContain(token);
    }
  });
});
