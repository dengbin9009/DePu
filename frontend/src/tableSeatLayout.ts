export type TableSeatPosition = Record<`--${string}`, string> & {
  '--seat-x': string;
  '--seat-y': string;
};

const tableSeatPositions: Record<number, ReadonlyArray<readonly [number, number]>> = {
  2: [[50, 8], [50, 92]],
  3: [[50, 8], [78, 82], [22, 82]],
  4: [[50, 8], [91.5, 50], [50, 92], [8.5, 50]],
  5: [[50, 8], [88, 34], [76, 84], [24, 84], [12, 34]],
  6: [[50, 8], [86, 26], [86, 74], [50, 92], [14, 74], [14, 26]],
  7: [[50, 8], [82, 20], [91.5, 52], [72, 88], [28, 88], [8.5, 52], [18, 20]],
  8: [[50, 8], [75, 14], [91.5, 38], [85, 76], [50, 92], [15, 76], [8.5, 38], [25, 14]],
  9: [[50, 8], [70, 14], [91.5, 35], [91.5, 67], [70, 90], [30, 90], [8.5, 67], [8.5, 35], [30, 14]]
};

export function tableSeatPosition(seatNo: number, seatCount: number): TableSeatPosition {
  const requestedSeatCount = Number.isFinite(seatCount) ? Math.trunc(seatCount) : 2;
  const normalizedSeatCount = Math.min(9, Math.max(2, requestedSeatCount));
  const requestedSeatNo = Number.isFinite(seatNo) ? Math.trunc(seatNo) : 1;
  const normalizedSeatNo = Math.min(normalizedSeatCount, Math.max(1, requestedSeatNo));
  const [x, y] = tableSeatPositions[normalizedSeatCount][normalizedSeatNo - 1];

  return {
    '--seat-x': `${x}%`,
    '--seat-y': `${y}%`
  };
}
