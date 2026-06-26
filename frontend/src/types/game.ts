export interface RuleSet {
  id: string;
  name: string;
  ranks: string[];
  deckSize: number;
  bettingStructures: BettingStructureType[];
  defaultBettingStructure: BettingStructureType;
  description?: string;
}

export type BettingStructureType = 'blinds' | 'ante';

export type BettingStructure =
  | { type: 'blinds'; smallBlind: number; bigBlind: number }
  | { type: 'ante'; ante: number; buttonBlind: number };

export interface CreateSeat {
  seatNo: number;
  name: string;
  stack: number;
}

export interface CreateGameRequest {
  rulesetId: string;
  seats: CreateSeat[];
  buttonSeat: number;
  bettingStructure: BettingStructure;
  dealMode: 'random' | 'debug';
}

export interface SeatState {
  seatNo: number;
  name: string;
  stack: number;
  holeCards: string[] | null;
  status: string;
  streetCommitted: number;
  handCommitted: number;
  currentHand?: CurrentHand | null;
}

export interface CurrentHand {
  handClass: string;
  bestCards: string[];
  rankVector: number[];
}

export interface PotState {
  id: string;
  amount: number;
  eligibleSeats: number[];
}

export interface GameSnapshot {
  id: string;
  rulesetId: string;
  bettingStructure: BettingStructure;
  dealMode?: string;
  stage: string;
  buttonSeat: number;
  currentSeat: number;
  currentBet: number;
  minRaise: number;
  board: string[] | null;
  seats: SeatState[];
  pots: PotState[] | null;
  showdown?: ShowdownResult[] | null;
  legalActions: string[];
  isReplay: boolean;
  debugLocked: boolean;
  version: number;
}

export interface ShowdownResult {
  seatNo: number;
  bestCards: string[];
  handClass: string;
  rankVector: number[];
  potAwards: Record<string, number> | null;
}

export interface StateSummary {
  stage: string;
  currentSeat?: number;
  currentBet: number;
  potTotal: number;
  board: string[];
  activeSeats: number[];
  allInSeats: number[];
  foldedSeats: number[];
  isReplay: boolean;
}

export interface ActionLog {
  seq: number;
  stage: string;
  seatNo: number;
  type: string;
  amount: number;
  payload?: Record<string, unknown>;
  stateSummary: StateSummary;
  createdAt: string;
}
