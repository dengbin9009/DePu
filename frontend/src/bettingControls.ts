import type { GameSnapshot, PotState, SeatState } from './types/game';

export type AmountAction = 'bet' | 'raise';
export type BetPreset = 'min' | 'half_pot' | 'pot' | 'all_in';

export interface BetAmountBounds {
  action: AmountAction;
  min: number;
  max: number;
  step: number;
  defaultAmount: number;
  fullMin: number;
  isShortAllIn: boolean;
}

export function currentActor(game: GameSnapshot | null): SeatState | null {
  if (!game) return null;
  return game.seats.find((seat) => seat.seatNo === game.currentSeat) ?? null;
}

export function amountAction(game: GameSnapshot | null): AmountAction | null {
  const actions = game?.legalActions ?? [];
  if (actions.includes('raise')) return 'raise';
  if (actions.includes('bet')) return 'bet';
  return null;
}

export function calculateBetAmountBounds(game: GameSnapshot | null): BetAmountBounds | null {
  const actor = currentActor(game);
  const action = amountAction(game);
  if (!game || !actor || !action) return null;

  const maxAmount = actor.streetCommitted + actor.stack;
  if (maxAmount <= 0) return null;

  const fullMin = action === 'raise' ? game.currentBet + Math.max(1, game.minRaise) : Math.max(1, game.minRaise);
  const minAmount = Math.min(fullMin, maxAmount);
  const step = sliderStep(game);

  return {
    action,
    min: minAmount,
    max: maxAmount,
    step,
    defaultAmount: minAmount,
    fullMin,
    isShortAllIn: maxAmount < fullMin
  };
}

export function clampBetAmount(game: GameSnapshot | null, amount: number): number {
  const bounds = calculateBetAmountBounds(game);
  if (!bounds) return 0;
  if (!Number.isFinite(amount)) return bounds.defaultAmount;
  return Math.min(bounds.max, Math.max(bounds.min, Math.round(amount)));
}

export function presetBetAmount(game: GameSnapshot | null, preset: BetPreset): number {
  const bounds = calculateBetAmountBounds(game);
  if (!bounds) return 0;

  const potAmount = potTotal(game?.pots ?? []);
  const presetAmount = (() => {
    switch (preset) {
      case 'min':
        return bounds.min;
      case 'half_pot':
        return bounds.min + Math.floor(potAmount / 2);
      case 'pot':
        return bounds.min + potAmount;
      case 'all_in':
        return bounds.max;
    }
  })();

  return clampBetAmount(game, presetAmount);
}

function potTotal(pots: PotState[]): number {
  return pots.reduce((sum, pot) => sum + pot.amount, 0);
}

function sliderStep(game: GameSnapshot): number {
  const blindUnit = game.bettingStructure.type === 'ante' ? game.bettingStructure.ante : game.bettingStructure.smallBlind;
  return Math.max(1, Math.floor(blindUnit / 5));
}
