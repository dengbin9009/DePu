import { describe, expect, it } from 'vitest';
import { existsSync } from 'node:fs';
import { cardAltText, cardBackAltText, cardBackImagePath, cardImagePath } from './cardAssets';

describe('cardAssets', () => {
  it('maps backend card codes to static card image paths', () => {
    expect(cardImagePath('As')).toBe('/德扑完整牌组_已重命名_透明PNG/cards/AS.png');
    expect(cardImagePath('Kh')).toBe('/德扑完整牌组_已重命名_透明PNG/cards/KH.png');
    expect(cardImagePath('Qd')).toBe('/德扑完整牌组_已重命名_透明PNG/cards/QD.png');
    expect(cardImagePath('Tc')).toBe('/德扑完整牌组_已重命名_透明PNG/cards/10C.png');
  });

  it('normalizes whitespace and rank casing while rejecting unknown cards', () => {
    expect(cardImagePath('  ah ')).toBe('/德扑完整牌组_已重命名_透明PNG/cards/AH.png');
    expect(cardImagePath('10s')).toBe('/德扑完整牌组_已重命名_透明PNG/cards/10S.png');
    expect(cardImagePath('1x')).toBeNull();
    expect(cardImagePath('')).toBeNull();
  });

  it('builds readable Chinese alt text for card images', () => {
    expect(cardAltText('As')).toBe('黑桃 A');
    expect(cardAltText('Th')).toBe('红桃 10');
    expect(cardAltText('7d')).toBe('方块 7');
    expect(cardAltText('Qc')).toBe('梅花 Q');
  });

  it('exposes the card back asset used for face-down board cards', () => {
    expect(cardBackImagePath()).toBe('/德扑完整牌组_已重命名_透明PNG/backs/BACK_1.png');
    expect(cardBackAltText()).toBe('扑克牌背面');
    expect(existsSync('public/德扑完整牌组_已重命名_透明PNG/backs/BACK_1.png')).toBe(true);
  });
});
