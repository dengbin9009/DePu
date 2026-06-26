import type { GameSnapshot, SeatState } from './types/game';

export type AnimationPhase = 'idle' | 'deal' | 'reveal_board' | 'action_shift' | 'pot_update';

export interface SeatVisual {
  seat: SeatState;
  angle: number;
  x: number;
  y: number;
  compact: boolean;
  active: boolean;
  dealer: boolean;
}

export interface TableVisualState {
  seatPositions: SeatVisual[];
  activeSeat?: number;
  dealerButtonSeat: number;
  potTotal: number;
  animationPhase: AnimationPhase;
  replayTransition: boolean;
  reducedMotion: boolean;
}

export function cardSuit(card: string): string {
  return card.slice(-1).toLowerCase();
}

export function cardRank(card: string): string {
  return card.slice(0, -1).toUpperCase();
}

export function isRedCard(card: string): boolean {
  return ['h', 'd'].includes(cardSuit(card));
}

export function seatPositions(seats: SeatState[], activeSeat?: number, dealerButtonSeat = 1): SeatVisual[] {
  const total = Math.max(seats.length, 1);
  return seats.map((seat, index) => {
    const angle = -90 + (360 / total) * index;
    const radians = (angle * Math.PI) / 180;
    return {
      seat,
      angle,
      x: 50 + Math.cos(radians) * 34,
      y: 50 + Math.sin(radians) * 32,
      compact: total >= 6,
      active: seat.seatNo === activeSeat,
      dealer: seat.seatNo === dealerButtonSeat
    };
  });
}

export function visibleOpponentSeats(seats: SeatVisual[], heroSeatNo?: number): SeatVisual[] {
  const opponents = seats.filter((seat) => seat.seat.seatNo !== heroSeatNo);
  const layout = opponentSeatLayout(opponents.length);
  return opponents.map((seat, index) => ({
    ...seat,
    x: layout[index]?.x ?? seat.x,
    y: layout[index]?.y ?? seat.y
  }));
}

function opponentSeatLayout(count: number): Array<{ x: number; y: number }> {
  const layouts: Record<number, Array<{ x: number; y: number }>> = {
    0: [],
    1: [{ x: 50, y: 16 }],
    2: [
      { x: 16, y: 34 },
      { x: 84, y: 34 }
    ],
    3: [
      { x: 15, y: 34 },
      { x: 50, y: 16 },
      { x: 85, y: 34 }
    ],
    4: [
      { x: 14, y: 38 },
      { x: 34, y: 20 },
      { x: 66, y: 20 },
      { x: 86, y: 38 }
    ],
    5: [
      { x: 13, y: 40 },
      { x: 29, y: 25 },
      { x: 50, y: 16 },
      { x: 71, y: 25 },
      { x: 87, y: 40 }
    ],
    6: [
      { x: 12, y: 42 },
      { x: 26, y: 29 },
      { x: 42, y: 18 },
      { x: 58, y: 18 },
      { x: 74, y: 29 },
      { x: 88, y: 42 }
    ],
    7: [
      { x: 12, y: 43 },
      { x: 23, y: 34 },
      { x: 35, y: 22 },
      { x: 50, y: 16 },
      { x: 65, y: 22 },
      { x: 77, y: 34 },
      { x: 88, y: 43 }
    ],
    8: [
      { x: 12, y: 43 },
      { x: 22, y: 36 },
      { x: 31, y: 25 },
      { x: 44, y: 17 },
      { x: 56, y: 17 },
      { x: 69, y: 25 },
      { x: 78, y: 36 },
      { x: 88, y: 43 }
    ],
    9: [
      { x: 12, y: 43 },
      { x: 20, y: 38 },
      { x: 29, y: 28 },
      { x: 40, y: 20 },
      { x: 50, y: 16 },
      { x: 60, y: 20 },
      { x: 71, y: 28 },
      { x: 80, y: 38 },
      { x: 88, y: 43 }
    ]
  };
  return layouts[count] ?? layouts[9];
}

export function tableVisualState(game: GameSnapshot | null, options: { replayTransition?: boolean; reducedMotion?: boolean } = {}): TableVisualState {
  const seats = game?.seats ?? [];
  const potTotal = (game?.pots ?? []).reduce((sum, pot) => sum + pot.amount, 0);
  return {
    seatPositions: seatPositions(seats, game?.currentSeat, game?.buttonSeat ?? 1),
    activeSeat: game?.currentSeat || undefined,
    dealerButtonSeat: game?.buttonSeat ?? 1,
    potTotal,
    animationPhase: game?.isReplay ? 'action_shift' : 'idle',
    replayTransition: Boolean(options.replayTransition),
    reducedMotion: Boolean(options.reducedMotion)
  };
}
