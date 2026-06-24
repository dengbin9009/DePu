export interface RuleSet {
  id: string;
  name: string;
  ranks: string[];
  smallBlind: number;
  bigBlind: number;
  description?: string;
}

export interface SeatState {
  seatNo: number;
  name: string;
  stack: number;
  holeCards: string[] | null;
  status: string;
  streetCommitted: number;
  handCommitted: number;
}

export interface PotState {
  id: string;
  amount: number;
  eligibleSeats: number[];
}

export interface GameSnapshot {
  id: string;
  rulesetId: string;
  stage: string;
  buttonSeat: number;
  currentSeat: number;
  board: string[] | null;
  seats: SeatState[];
  pots: PotState[] | null;
  showdown?: ShowdownResult[] | null;
  legalActions: string[];
  version: number;
}

export interface ShowdownResult {
  seatNo: number;
  bestCards: string[];
  handClass: string;
  rankVector: number[];
  awards: Record<string, number> | null;
}

export interface ActionLog {
  seq: number;
  stage: string;
  seatNo: number;
  type: string;
  amount: number;
  summary: string;
  createdAt: string;
}
