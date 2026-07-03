import { describe, expect, it } from 'vitest';
import appSource from './App.vue?raw';
import rulesTestSource from './pages/RulesTestPage.vue?raw';
import roomSource from './pages/RoomPage.vue?raw';
import meSource from './pages/MePage.vue?raw';
import historySource from './pages/HistoryPage.vue?raw';
import { tableVisualState, visibleOpponentSeats } from './pokerVisuals';
import type { GameSnapshot } from './types/game';

function snapshotWithSeats(count: number): GameSnapshot {
	return {
		id: `game-${count}`,
		rulesetId: 'long-holdem',
		bettingStructure: { type: 'blinds', smallBlind: 50, bigBlind: 100 },
		stage: 'flop',
		buttonSeat: 1,
		currentSeat: Math.min(2, count),
		currentBet: 100,
		minRaise: 100,
		board: ['As', 'Kh', 'Qd'],
		seats: Array.from({ length: count }, (_, i) => ({
			seatNo: i + 1,
			name: `P${i + 1}`,
			stack: 1000,
			holeCards: ['Js', 'Tc'],
			status: 'active',
			streetCommitted: 0,
			handCommitted: 0
		})),
		pots: [{ id: 'pot-1', amount: 150, eligibleSeats: [1, 2] }],
		showdown: [],
		legalActions: ['fold', 'call', 'raise'],
		isReplay: false,
		debugLocked: false,
		version: 1
	};
}

describe('App visual contract', () => {
	it('keeps the app shell with router outlet and bottom tabs', () => {
		for (const token of ['router-view', 'BottomTabBar', "route.path !== '/login'", "route.path !== '/rules-test'", "!route.path.startsWith('/room/')"]) {
			expect(appSource).toContain(token);
		}
	});

	it('keeps the rules test table, board, pot, actions, history, and debug areas in the rules test page', () => {
		for (const token of [
			'v-if="game" class="table-zone panel mobile-panel"',
			'class="phone-stage"',
			'class="mobile-table-screen"',
			'class="table-status-bar"',
			'class="community-core"',
			'class="hero-hand"',
			'class="hero-status"',
			'class="seat-cards"',
			'class="seat-hand-rank"',
			'class="hero-rank"',
			'class="card opponent-card"',
			'visibleOpponentSeats',
			'seatHandClass((seat.seat.currentHand as any)?.handClass)',
			'seatHandClass(heroSeat()?.currentHand?.handClass)',
			'has-game',
			'class="table-felt"',
			'class="board"',
			'class="pot-stack"',
			'class="seat"',
			'class="actions hero-actions"',
			'class="bet-amount-panel"',
			'class="bet-slider"',
			'class="bet-presets"',
			'actionButtonLabel(action)',
			'setBetPreset',
			'调试发牌',
			'历史',
			'摊牌',
			'potAwards',
			'replayTransition',
			'cardImagePath',
			'<img v-if="cardImagePath(card)"',
			'faceDownBoardCards',
			'cardBackImagePath'
		]) {
			expect(rulesTestSource).toContain(token);
		}
	});

	it('supports the required 2/4/6/9/10 seat table scenarios', () => {
		for (const count of [2, 4, 6, 9, 10]) {
			const visual = tableVisualState(snapshotWithSeats(count));
			expect(visual.seatPositions).toHaveLength(count);
			expect(visual.seatPositions.some((seat) => seat.active)).toBe(true);
			expect(visual.potTotal).toBe(150);
			expect(visual.seatPositions.every((seat) => seat.x >= 0 && seat.x <= 100 && seat.y >= 0 && seat.y <= 100)).toBe(true);
			expect(visual.seatPositions.every((seat) => seat.compact)).toBe(count >= 6);
		}
	});

	it('keeps non-hero seats away from the bottom hero hand area', () => {
		const visual = tableVisualState(snapshotWithSeats(4));
		const hero = visual.seatPositions[0];
		const opponents = visibleOpponentSeats(visual.seatPositions, hero.seat.seatNo);

		expect(opponents).toHaveLength(3);
		expect(opponents.every((seat) => seat.y <= 42)).toBe(true);
		expect(opponents.some((seat) => seat.x < 35)).toBe(true);
		expect(opponents.some((seat) => seat.x > 65)).toBe(true);
		expect(opponents.some((seat) => seat.x >= 42 && seat.x <= 58 && seat.y <= 20)).toBe(true);
	});

	it('keeps multiplayer profile, wallet, room history, and player perspective sections in dedicated pages', () => {
		for (const token of ['金币充值', '当前为模拟充值', 'simulateRecharge', '钱包流水', 'walletTransactionLabel', 'formatDateTime']) {
			expect(meSource).toContain(token);
		}
		for (const token of ['退出登录', 'doLogout', "router.push('/login')"]) {
			expect(meSource).toContain(token);
		}
		for (const token of ['房间最近牌局', '个人战绩', 'recentRoomHands']) {
			expect(historySource).toContain(token);
		}
		for (const token of ['myRoomSeat', 'isMyTurn', 'myRoomHandPlayer', '坐下 #', '选座位', '等待开局']) {
			expect(roomSource).toContain(token);
		}
	});
});
