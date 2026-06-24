import type { ActionLog, GameSnapshot, RuleSet } from '../types/game';

const jsonHeaders = { 'Content-Type': 'application/json' };

export async function fetchRuleSets(): Promise<RuleSet[]> {
  const res = await fetch('/api/rulesets');
  return readJSON(res);
}

export async function createGame(rulesetId: string): Promise<GameSnapshot> {
  const res = await fetch('/api/games', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({
      rulesetId,
      buttonSeat: 1,
      smallBlind: 50,
      bigBlind: 100,
      dealMode: 'random',
      seats: [
        { seatNo: 1, name: 'BTN', stack: 1000 },
        { seatNo: 2, name: 'SB', stack: 1000 },
        { seatNo: 3, name: 'BB', stack: 1000 },
        { seatNo: 4, name: 'UTG', stack: 1000 }
      ]
    })
  });
  return readJSON(res);
}

export async function submitAction(game: GameSnapshot, type: string): Promise<GameSnapshot> {
  const amount = type === 'raise' ? Math.max(...game.seats.map((seat) => seat.streetCommitted)) + 100 : 0;
  const res = await fetch(`/api/games/${game.id}/actions`, {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({
      seatNo: game.currentSeat,
      type,
      amount,
      version: game.version
    })
  });
  return readJSON(res);
}

export async function fetchHistory(gameId: string): Promise<ActionLog[]> {
  const res = await fetch(`/api/games/${gameId}/history`);
  return readJSON(res);
}

export async function setDebugCards(game: GameSnapshot, holeCards: Record<string, string[]>, board: string[]): Promise<GameSnapshot> {
  const res = await fetch(`/api/games/${game.id}/debug/cards`, {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({
      version: game.version,
      holeCards,
      board
    })
  });
  return readJSON(res);
}

export async function replayTo(gameId: string, toSeq: number): Promise<GameSnapshot> {
  const res = await fetch(`/api/games/${gameId}/replay`, {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ toSeq })
  });
  return readJSON(res);
}

async function readJSON<T>(res: Response): Promise<T> {
  const body = await res.json();
  if (!res.ok) {
    throw new Error(body.message || '请求失败');
  }
  return body as T;
}
