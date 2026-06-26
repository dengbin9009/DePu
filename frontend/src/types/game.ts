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


export interface ProfileResponse {
  id: string;
  username: string;
  nickname: string;
  walletBalance: number;
  handsPlayed: number;
  totalProfit: number;
  lastPlayedAt: string | null;
}

export interface WalletTransaction {
  id: string;
  type: string;
  amount: number;
  balanceAfter: number;
  referenceType?: string | null;
  referenceId?: string | null;
  note?: string | null;
  createdAt: string;
}

export interface WalletResponse {
  balance: number;
  transactions: WalletTransaction[];
}

export interface RechargeOption {
  code: string;
  label: string;
  amount: number;
}

export interface RechargeResponse {
  status: 'simulated_success';
  walletBalance: number;
  transaction: WalletTransaction;
}

export interface RoomMember {
  userId: string;
  nickname: string;
  role: 'owner' | 'player';
  joinedAt: string;
}

export interface RoomSeat {
  seatNo: number;
  seatStatus: 'empty' | 'occupied' | 'sitting_out';
  userId?: string | null;
  nickname?: string | null;
  buyInChips?: number | null;
}

export interface RoomResponse {
  id: string;
  inviteCode: string;
  ownerUserId: string;
  status: 'waiting' | 'playing' | 'closed';
  ruleSetId?: string;
  seatCount?: number;
  minPlayersToStart?: number;
  members: RoomMember[];
  seats: RoomSeat[];
}

export interface UserHandRecord {
  handId: string;
  roomId: string;
  handNo: number;
  completedAt: string;
  nickname: string;
  profit: number;
  winnerSummary: string;
}

export interface RoomHandHistoryParticipant {
  userId: string;
  nickname: string;
  seatNo: number;
  profit: number;
  awardAmount: number;
  handCommitted: number;
  resultType: string;
  holeCards?: string[] | null;
  bestCards?: string[] | null;
  handClass?: string;
}

export interface RoomHandHistoryRecord {
  handId: string;
  roomId: string;
  handNo: number;
  ruleSetId: string;
  completedAt: string;
  winnerSummary: string;
  potSummary: string;
  totalPot: number;
  boardCards: string[];
  participants: RoomHandHistoryParticipant[];
}


export interface RoomHandPlayer {
  seatNo: number;
  name: string;
  stack: number;
  holeCards?: string[] | null;
  status: string;
  streetCommitted: number;
  handCommitted: number;
  hasActed?: boolean;
}

export interface RoomHandState {
  roomId: string;
  handId: string;
  status: string;
  currentSeat: number | null;
  pot: number;
  boardCards: string[] | null;
  players: RoomHandPlayer[];
  availableActions: string[];
}
