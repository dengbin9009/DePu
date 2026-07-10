import type {
  ActionLog,
  CreateRoomPayload,
  CreateGameRequest,
  GameSnapshot,
  HandReplayResponse,
  ProfileResponse,
  RechargeOption,
  RechargeResponse,
  RoomHandHistoryRecord,
  RoomLeaderboardItem,
  RoomHandState,
  RoomResponse,
  RuleSet,
  UserHandRecord,
  WalletResponse
} from '../types/game';

const jsonHeaders = { 'Content-Type': 'application/json' };
const apiBase = import.meta.env.VITE_DEPU_API_BASE || '';

function apiUrl(path: string): string {
  return `${apiBase}${path}`;
}

function authHeaders(token?: string): HeadersInit {
  return token ? { ...jsonHeaders, Authorization: `Bearer ${token}` } : jsonHeaders;
}

export async function fetchRuleSets(): Promise<RuleSet[]> {
  const res = await fetch(apiUrl('/api/rulesets'));
  return readJSON(res);
}

export async function register(username: string, password: string, nickname: string): Promise<{ user: { id: string; username: string; nickname: string }; token: string }> {
  const res = await fetch(apiUrl('/api/auth/register'), { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ username, password, nickname }) });
  return readJSON(res);
}

export async function login(username: string, password: string): Promise<{ user: { id: string; username: string; nickname: string }; token: string }> {
  const res = await fetch(apiUrl('/api/auth/login'), { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ username, password }) });
  return readJSON(res);
}

export async function fetchMe(token: string): Promise<ProfileResponse> {
  const res = await fetch(apiUrl('/api/me'), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function updateProfile(token: string, nickname: string): Promise<ProfileResponse> {
  const res = await fetch(apiUrl('/api/me/profile'), { method: 'PATCH', headers: authHeaders(token), body: JSON.stringify({ nickname }) });
  return readJSON(res);
}

export async function fetchWallet(token: string): Promise<WalletResponse> {
  const res = await fetch(apiUrl('/api/me/wallet'), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function fetchRechargeOptions(): Promise<{ options: RechargeOption[] }> {
  const res = await fetch(apiUrl('/api/recharge/options'));
  return readJSON(res);
}

export async function recharge(token: string, optionCode: string): Promise<RechargeResponse> {
  const res = await fetch(apiUrl('/api/recharge'), { method: 'POST', headers: authHeaders(token), body: JSON.stringify({ optionCode, confirm: true }) });
  return readJSON(res);
}

export async function createRoom(token: string, payload: CreateRoomPayload): Promise<RoomResponse> {
  const res = await fetch(apiUrl('/api/rooms'), { method: 'POST', headers: authHeaders(token), body: JSON.stringify(payload) });
  return readJSON(res);
}

export async function joinRoom(token: string, inviteCode: string): Promise<RoomResponse> {
  const res = await fetch(apiUrl('/api/rooms/join'), { method: 'POST', headers: authHeaders(token), body: JSON.stringify({ inviteCode }) });
  return readJSON(res);
}

export async function fetchRoom(token: string, roomId: string): Promise<RoomResponse> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}`), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function takeSeat(token: string, roomId: string, seatNo: number, buyInChips: number): Promise<RoomResponse> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/seats/${seatNo}`), { method: 'POST', headers: authHeaders(token), body: JSON.stringify({ buyInChips }) });
  return readJSON(res);
}

export async function leaveSeat(token: string, roomId: string, seatNo: number): Promise<RoomResponse> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/seats/${seatNo}`), { method: 'DELETE', headers: authHeaders(token) });
  return readJSON(res);
}

export async function leaveRoom(token: string, roomId: string): Promise<RoomResponse> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/members/me`), { method: 'DELETE', headers: authHeaders(token) });
  return readJSON(res);
}



export async function startRoomHand(token: string, roomId: string): Promise<RoomHandState> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/start`), { method: 'POST', headers: authHeaders(token) });
  return readJSON(res);
}

export async function fetchCurrentRoomHand(token: string, roomId: string): Promise<RoomHandState> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/current-hand`), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function submitRoomAction(token: string, roomId: string, action: string, amount = 0): Promise<RoomHandState> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/actions`), {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ action, amount })
  });
  return readJSON(res);
}

export async function fetchUserHands(token: string): Promise<{ items: UserHandRecord[] }> {
  const res = await fetch(apiUrl('/api/me/hands'), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function fetchRoomHands(token: string, roomId: string): Promise<{ items: RoomHandHistoryRecord[] }> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/hands/recent`), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function fetchRoomLeaderboard(token: string, roomId: string): Promise<{ items: RoomLeaderboardItem[] }> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/leaderboard`), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function fetchRoomHandReplay(token: string, roomId: string, handId: string): Promise<HandReplayResponse> {
  const res = await fetch(apiUrl(`/api/rooms/${roomId}/hands/${handId}/replay`), { headers: authHeaders(token) });
  return readJSON(res);
}

export async function createGame(request: CreateGameRequest): Promise<GameSnapshot> {
  const res = await fetch(apiUrl('/api/games'), { method: 'POST', headers: jsonHeaders, body: JSON.stringify(request) });
  return readJSON(res);
}

export async function submitAction(game: GameSnapshot, type: string, amount = 0): Promise<GameSnapshot> {
  const res = await fetch(apiUrl(`/api/games/${game.id}/actions`), { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ seatNo: game.currentSeat, type, amount, version: game.version }) });
  return readJSON(res);
}

export async function fetchHistory(gameId: string): Promise<ActionLog[]> {
  const res = await fetch(apiUrl(`/api/games/${gameId}/history`));
  return readJSON(res);
}

export async function setDebugCards(game: GameSnapshot, holeCards: Record<string, string[]>, board: string[]): Promise<GameSnapshot> {
  const res = await fetch(apiUrl(`/api/games/${game.id}/debug/cards`), { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ version: game.version, holeCards, board }) });
  return readJSON(res);
}

export async function replayTo(gameId: string, toSeq: number): Promise<GameSnapshot> {
  const res = await fetch(apiUrl(`/api/games/${gameId}/replay`), { method: 'POST', headers: jsonHeaders, body: JSON.stringify({ toSeq }) });
  return readJSON(res);
}

async function readJSON<T>(res: Response): Promise<T> {
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body?.error?.message || body.message || '请求失败');
  }
  return body as T;
}
