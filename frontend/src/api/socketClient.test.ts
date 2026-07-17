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

  disconnect() {
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

  it('defaults the socket URL to the current origin so Vite proxy can handle local development', async () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    FakeWebSocket.instances = [];
    const client = createRoomSocketClient('tok_proxy');

    const connected = client.connect();
    const socket = FakeWebSocket.instances[0];
    expect(socket.url).toBe('ws://localhost:3000/api/socket?token=tok_proxy');
    socket.open();
    await connected;
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

  it.each([
    ['room.start_hand', {}],
    ['room.action', { action: 'call', amount: 0 }]
  ])('does not replay an unacknowledged %s command after reconnect', async (type, payload) => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    FakeWebSocket.instances = [];
    const client = createRoomSocketClient('tok_reconnect');
    const snapshots: unknown[] = [];
    client.on('room.snapshot', (message) => snapshots.push(message.payload));
    const connected = client.connect();
    const firstSocket = FakeWebSocket.instances[0];
    firstSocket.open();
    await connected;

    const unacknowledged = client.send(type, 'room_1', payload);
    const firstCommand = JSON.parse(firstSocket.sent[0]);
    firstSocket.disconnect();

    await expect(unacknowledged).rejects.toThrow('socket disconnected before acknowledgement');

    const reconnected = client.connect();
    const secondSocket = FakeWebSocket.instances[1];
    secondSocket.open();
    await reconnected;
    expect(secondSocket.sent).toEqual([]);

    const subscribed = client.send('room.subscribe', 'room_1', {});
    const subscribeCommand = JSON.parse(secondSocket.sent[0]);
    secondSocket.receive({
      type: 'ack',
      requestId: subscribeCommand.requestId,
      roomId: 'room_1',
      payload: { command: 'room.subscribe' }
    });
    await expect(subscribed).resolves.toEqual({ command: 'room.subscribe' });
    secondSocket.receive({
      type: 'room.snapshot',
      roomId: 'room_1',
      payload: { room: { id: 'room_1', status: 'waiting' }, hand: null }
    });
    expect(snapshots).toEqual([{ room: { id: 'room_1', status: 'waiting' }, hand: null }]);

    const retried = client.send(type, 'room_1', payload);
    const retryCommand = JSON.parse(secondSocket.sent[1]);
    expect(retryCommand.requestId).not.toBe(firstCommand.requestId);
    secondSocket.receive({
      type: 'ack',
      requestId: retryCommand.requestId,
      roomId: 'room_1',
      payload: { command: type }
    });
    await expect(retried).resolves.toEqual({ command: type });
  });
});
