import { describe, expect, it } from 'vitest';
import viteConfigSource from '../vite.config?raw';

describe('vite dev proxy', () => {
  it('proxies websocket upgrades for the socket endpoint', () => {
    const apiProxyBlock = viteConfigSource.slice(
      viteConfigSource.indexOf("'/api'"),
      viteConfigSource.indexOf("'/health'")
    );

    expect(apiProxyBlock).toContain('ws: true');
  });
});
