export interface SocketEnvelope<T = unknown> {
  type: string;
  requestId?: string;
  roomId?: string;
  payload?: T;
  sentAt?: string;
  roomVersion?: number;
  handId?: string;
  handVersion?: number;
}

type SocketHandler = (message: SocketEnvelope) => void;

interface PendingCommand {
  resolve: (payload: unknown) => void;
  reject: (error: Error) => void;
}

const apiBase = import.meta.env.VITE_DEPU_API_BASE || '';

function socketUrl(token: string): string {
  const base = apiBase || window.location.origin;
  const url = new URL('/api/socket', base);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.searchParams.set('token', token);
  return url.toString();
}

export function createRoomSocketClient(token: string) {
  let socket: WebSocket | null = null;
  let requestSeq = 0;
  const pending = new Map<string, PendingCommand>();
  const handlers = new Map<string, Set<SocketHandler>>();

  function rejectPending(error: Error) {
    for (const command of pending.values()) {
      command.reject(error);
    }
    pending.clear();
  }

  function emit(message: SocketEnvelope) {
    const group = handlers.get(message.type);
    if (!group) return;
    for (const handler of group) {
      handler(message);
    }
  }

  function handleMessage(event: MessageEvent<string>) {
    const message = JSON.parse(event.data) as SocketEnvelope;
    if (message.type === 'ack' && message.requestId) {
      const command = pending.get(message.requestId);
      if (command) {
        pending.delete(message.requestId);
        command.resolve(message.payload);
      }
      return;
    }
    if (message.type === 'error' && message.requestId) {
      const command = pending.get(message.requestId);
      if (command) {
        pending.delete(message.requestId);
        const payload = message.payload as { message?: string; code?: string } | undefined;
        command.reject(new Error(payload?.message || payload?.code || 'socket error'));
      }
      emit(message);
      return;
    }
    emit(message);
  }

  function connect(): Promise<void> {
    if (socket && socket.readyState === WebSocket.OPEN) {
      return Promise.resolve();
    }
    socket = new WebSocket(socketUrl(token));
    socket.onmessage = handleMessage;
    socket.onclose = () => {
      socket = null;
      rejectPending(new Error('socket disconnected before acknowledgement'));
    };
    return new Promise((resolve, reject) => {
      if (!socket) return reject(new Error('socket unavailable'));
      socket.onopen = () => resolve();
      socket.onerror = () => reject(new Error('socket connection failed'));
    });
  }

  function send(type: string, roomId: string, payload: unknown = {}): Promise<unknown> {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return Promise.reject(new Error('socket is not connected'));
    }
    const requestId = `req_${Date.now()}_${++requestSeq}`;
    const message: SocketEnvelope = { type, requestId, roomId, payload };
    const result = new Promise<unknown>((resolve, reject) => {
      pending.set(requestId, { resolve, reject });
    });
    socket.send(JSON.stringify(message));
    return result;
  }

  function isConnected() {
    return socket?.readyState === WebSocket.OPEN;
  }

  function on(type: string, handler: SocketHandler): () => void {
    let group = handlers.get(type);
    if (!group) {
      group = new Set();
      handlers.set(type, group);
    }
    group.add(handler);
    return () => group?.delete(handler);
  }

  function close() {
    socket?.close();
    socket = null;
    rejectPending(new Error('socket closed before acknowledgement'));
  }

  return { connect, isConnected, send, on, close };
}
