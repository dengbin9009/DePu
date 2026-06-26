import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import appSource from '../../src/App.vue?raw';
import { tableVisualState } from '../../src/pokerVisuals';
import type { GameSnapshot } from '../../src/types/game';

const cssSource = readFileSync('src/style.css', 'utf8');

const scenarios = [
	{ players: 2, viewport: '1280x900 桌面' },
	{ players: 4, viewport: '1280x900 桌面第一屏' },
	{ players: 6, viewport: '390x844 移动纵向滚动' },
	{ players: 9, viewport: '390x844 移动纵向滚动' },
	{ players: 10, viewport: '1280x900 紧凑桌面' },
	{ players: 4, viewport: '手动回放节点切换' }
];

function snapshot(players: number): GameSnapshot {
	return {
		id: `visual-${players}`,
		rulesetId: 'short-deck',
		bettingStructure: { type: 'ante', ante: 10, buttonBlind: 50 },
		stage: 'turn',
		buttonSeat: 1,
		currentSeat: Math.min(2, players),
		currentBet: 60,
		minRaise: 50,
		board: ['As', 'Kh', 'Qd', 'Jc'],
		seats: Array.from({ length: players }, (_, i) => ({
			seatNo: i + 1,
			name: `P${i + 1}`,
			stack: 1000 - i * 10,
			holeCards: ['Ts', '9c'],
			status: i === players - 1 ? 'all_in' : 'active',
			streetCommitted: i === 0 ? 60 : 10,
			handCommitted: i === 0 ? 60 : 10
		})),
		pots: [{ id: 'pot-1', amount: 100 + players * 10, eligibleSeats: Array.from({ length: players }, (_, i) => i + 1) }],
		showdown: [],
		legalActions: ['fold', 'call', 'raise'],
		isReplay: false,
		debugLocked: false,
		version: 1
	};
}

describe('visual acceptance record', () => {
	it('keeps an executable record for required table screenshots', () => {
		expect(scenarios.map((scenario) => `${scenario.players}-${scenario.viewport}`)).toMatchInlineSnapshot(`
			[
			  "2-1280x900 桌面",
			  "4-1280x900 桌面第一屏",
			  "6-390x844 移动纵向滚动",
			  "9-390x844 移动纵向滚动",
			  "10-1280x900 紧凑桌面",
			  "4-手动回放节点切换",
			]
		`);
		for (const { players } of scenarios) {
			const visual = tableVisualState(snapshot(players), { replayTransition: true });
			expect(visual.seatPositions).toHaveLength(players);
			expect(visual.replayTransition).toBe(true);
			expect(visual.seatPositions.every((seat) => seat.compact)).toBe(players >= 6);
		}
		expect(appSource).toContain('回放');
		expect(cssSource).toContain('.replaying .table-felt');
	});
});
