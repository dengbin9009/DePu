import { describe, expect, it, vi } from 'vitest';
import { createRoomSocketClient } from './socketClient';

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static OPEN = 1;

  url: string;
  readyState = FakeWebSocket.OPEN;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = 3;
    this.onclose?.();
  }

  open() {
    this.onopen?.();
  }

  receive(message: unknown) {
    this.onmessage?.({ data: JSON.stringify(message) });
  }
}

describe('socketClient', () => {
  it('connects with token and sends commands with request ids', async () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    FakeWebSocket.instances = [];
    const client = createRoomSocketClient('tok_abc');

    const connected = client.connect();
    const socket = FakeWebSocket.instances[0];
    expect(socket.url).toContain('/api/socket?token=tok_abc');
    socket.open();
    await connected;

    const ack = client.send('room.subscribe', 'room_1', {});
    const sent = JSON.parse(socket.sent[0]);
    expect(sent).toMatchObject({ type: 'room.subscribe', roomId: 'room_1', payload: {} });
    expect(sent.requestId).toMatch(/^req_/);

    socket.receive({ type: 'ack', requestId: sent.requestId, roomId: 'room_1', payload: { command: 'room.subscribe' } });
    await expect(ack).resolves.toEqual({ command: 'room.subscribe' });
  });

  it('rejects a pending command when the server returns error', async () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    FakeWebSocket.instances = [];
    const client = createRoomSocketClient('tok_abc');
    const connected = client.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    await connected;

    const pending = client.send('room.action', 'room_1', { action: 'call', amount: 0 });
    const sent = JSON.parse(socket.sent[0]);
    socket.receive({ type: 'error', requestId: sent.requestId, roomId: 'room_1', payload: { code: 'not_your_turn', message: 'not your turn' } });

    await expect(pending).rejects.toThrow('not your turn');
  });

  it('dispatches room and hand events to subscribers', async () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    FakeWebSocket.instances = [];
    const client = createRoomSocketClient('tok_abc');
    const events: string[] = [];
    client.on('room.snapshot', () => events.push('snapshot'));
    client.on('hand.updated', () => events.push('updated'));
    const connected = client.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    await connected;

    socket.receive({ type: 'room.snapshot', roomId: 'room_1', payload: { room: { id: 'room_1' }, hand: null } });
    socket.receive({ type: 'hand.updated', roomId: 'room_1', payload: { hand: { handId: 'hand_1' } } });

    expect(events).toEqual(['snapshot', 'updated']);
  });
});

