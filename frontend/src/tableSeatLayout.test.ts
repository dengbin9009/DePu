import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import roomSource from './pages/RoomPage.vue?raw';
import { tableSeatPosition } from './tableSeatLayout';

const styleSource = readFileSync(resolve(process.cwd(), 'src/style.css'), 'utf8');

describe('table seat ring layout', () => {
  it('uses stable unique coordinates for two through nine seats', () => {
    const ring = { left: 2, top: 100, width: 356, height: 244 };
    const seat = { width: 64, height: 56 };

    for (let seatCount = 2; seatCount <= 9; seatCount += 1) {
      const positions = Array.from({ length: seatCount }, (_, index) => tableSeatPosition(index + 1, seatCount));
      expect(new Set(positions.map((position) => `${position['--seat-x']}:${position['--seat-y']}`)).size).toBe(seatCount);

      const rectangles = positions.map((position) => {
        expect(Number.parseFloat(position['--seat-x'])).toBeGreaterThanOrEqual(8);
        expect(Number.parseFloat(position['--seat-x'])).toBeLessThanOrEqual(92);
        expect(Number.parseFloat(position['--seat-y'])).toBeGreaterThanOrEqual(8);
        expect(Number.parseFloat(position['--seat-y'])).toBeLessThanOrEqual(92);

        const centerX = ring.left + ring.width * Number.parseFloat(position['--seat-x']) / 100;
        const centerY = ring.top + ring.height * Number.parseFloat(position['--seat-y']) / 100;
        return {
          left: centerX - seat.width / 2,
          right: centerX + seat.width / 2,
          top: centerY - seat.height / 2,
          bottom: centerY + seat.height / 2
        };
      });

      rectangles.forEach((rectangle) => {
        expect(rectangle.left).toBeGreaterThanOrEqual(0);
        expect(rectangle.right).toBeLessThanOrEqual(360);
        expect(rectangle.top).toBeGreaterThanOrEqual(90);
        expect(rectangle.bottom).toBeLessThanOrEqual(360);
      });

      for (let leftIndex = 0; leftIndex < rectangles.length; leftIndex += 1) {
        for (let rightIndex = leftIndex + 1; rightIndex < rectangles.length; rightIndex += 1) {
          const left = rectangles[leftIndex];
          const right = rectangles[rightIndex];
          const overlaps = left.left < right.right
            && left.right > right.left
            && left.top < right.bottom
            && left.bottom > right.top;
          expect(overlaps).toBe(false);
        }
      }
    }
  });

  it('keeps nine mobile seats outside the center card zone', () => {
    const ring = { left: 2, top: 100, width: 356, height: 244 };
    const seat = { width: 64, height: 56 };
    const center = { left: 65, right: 295, top: 174, bottom: 274 };
    const rectangles = Array.from({ length: 9 }, (_, index) => {
      const position = tableSeatPosition(index + 1, 9);
      const centerX = ring.left + ring.width * Number.parseFloat(position['--seat-x']) / 100;
      const centerY = ring.top + ring.height * Number.parseFloat(position['--seat-y']) / 100;
      return {
        left: centerX - seat.width / 2,
        right: centerX + seat.width / 2,
        top: centerY - seat.height / 2,
        bottom: centerY + seat.height / 2
      };
    });

    rectangles.forEach((rectangle) => {
      const overlapsCenter = rectangle.left < center.right
        && rectangle.right > center.left
        && rectangle.top < center.bottom
        && rectangle.bottom > center.top;
      expect(overlapsCenter).toBe(false);
    });
  });

  it('binds the coordinate helper and constrains seat content', () => {
    expect(roomSource).toContain("import { tableSeatPosition } from '../tableSeatLayout';");
    expect(roomSource).toContain(':style="tableSeatPosition(seatNo, room.seatCount)"');
    expect(styleSource).toMatch(/\.seat-ring-casino \.casino-seat-node\s*\{[^}]*left:\s*var\(--seat-x\);[^}]*top:\s*var\(--seat-y\);[^}]*transform:\s*translate\(-50%,\s*-50%\);/s);
    expect(styleSource).toMatch(/\.casino-seat-node\s*\{[^}]*box-sizing:\s*border-box;[^}]*width:\s*var\(--table-seat-width\);[^}]*height:\s*var\(--table-seat-height\);[^}]*overflow:\s*hidden;/s);
    expect(styleSource).toMatch(/\.casino-seat-node \.seat-name[^}]*\{[^}]*overflow:\s*hidden;[^}]*text-overflow:\s*ellipsis;[^}]*white-space:\s*nowrap;/s);
  });
});
