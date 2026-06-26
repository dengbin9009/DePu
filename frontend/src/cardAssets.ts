const rankLabels: Record<string, string> = {
  '2': '2',
  '3': '3',
  '4': '4',
  '5': '5',
  '6': '6',
  '7': '7',
  '8': '8',
  '9': '9',
  T: '10',
  J: 'J',
  Q: 'Q',
  K: 'K',
  A: 'A'
};

const suitLabels: Record<string, string> = {
  s: '黑桃',
  h: '红桃',
  d: '方块',
  c: '梅花'
};

const cardAssetBasePath = '/德扑完整牌组_已重命名_透明PNG';

export function cardImagePath(card: string): string | null {
  const normalized = normalizeCardCode(card);
  if (!normalized) return null;
  const rank = normalized.slice(0, -1).toUpperCase();
  const suit = normalized.slice(-1).toUpperCase();
  const fileRank = rank === 'T' ? '10' : rank;
  return `${cardAssetBasePath}/cards/${fileRank}${suit}.png`;
}

export function cardBackImagePath(): string {
  return `${cardAssetBasePath}/backs/BACK_1.png`;
}

export function cardBackAltText(): string {
  return '扑克牌背面';
}

export function cardAltText(card: string): string {
  const normalized = normalizeCardCode(card);
  if (!normalized) return card;
  const rank = normalized.slice(0, -1).toUpperCase();
  const suit = normalized.slice(-1);
  return `${suitLabels[suit]} ${rankLabels[rank]}`;
}

function normalizeCardCode(card: string): string | null {
  const trimmed = card.trim();
  if (!trimmed) return null;

  const normalizedTen = trimmed.replace(/^10/i, 'T');
  const rank = normalizedTen.slice(0, -1).toUpperCase();
  const suit = normalizedTen.slice(-1).toLowerCase();
  if (!rankLabels[rank] || !suitLabels[suit]) return null;
  return `${rank.toLowerCase()}${suit}`;
}
