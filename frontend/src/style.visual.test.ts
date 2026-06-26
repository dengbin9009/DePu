import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';

const cssSource = readFileSync('src/style.css', 'utf8');

function durationMs(token: string): number {
	const value = Number.parseInt(token, 10);
	return token.endsWith('ms') ? value : value * 1000;
}

describe('table style contract', () => {
	it('defines desktop, compact, mobile, and reduced-motion rules', () => {
		expect(cssSource).toContain('@media (min-width: 1280px)');
		expect(cssSource).toContain('@media (max-width: 760px)');
		expect(cssSource).toContain('.phone-stage');
		expect(cssSource).toContain('.mobile-table-screen');
		expect(cssSource).toContain('.community-core');
		expect(cssSource).toContain('.hero-hand');
		expect(cssSource).toContain('.hero-status');
		expect(cssSource).toContain('.bet-amount-panel');
		expect(cssSource).toContain('.bet-slider');
		expect(cssSource).toContain('.bet-presets');
		expect(cssSource).toContain('.seat-cards');
		expect(cssSource).toContain('.opponent-card');
		expect(cssSource).toContain('aspect-ratio: 9 / 16');
		expect(cssSource).toContain('.app-shell.has-game');
		expect(cssSource).toContain('.table-zone.compact');
		expect(cssSource).toContain('overflow-y: auto');
		expect(cssSource).toContain('@media (prefers-reduced-motion: reduce)');
	});

	it('keeps reveal and replay animations under one second', () => {
		const durations = [...cssSource.matchAll(/(?:animation|transition)(?:-duration)?:[^;]*?(\d+(?:ms|s))/g)].map((match) => durationMs(match[1]));
		expect(durations.length).toBeGreaterThan(0);
		expect(Math.max(...durations)).toBeLessThanOrEqual(1000);
		expect(cssSource).toContain('@keyframes reveal-board');
		expect(cssSource).toContain('@keyframes replay-pulse');
	});
});
