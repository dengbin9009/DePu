import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const frontendRoot = resolve(process.cwd());
const runnerPath = resolve(frontendRoot, 'scripts/table-hardening-browser-acceptance.mjs');
const responsiveRunnerPath = resolve(frontendRoot, 'scripts/table-responsive-browser-acceptance.mjs');
const replayRunnerPath = resolve(frontendRoot, 'scripts/table-replay-browser-acceptance.mjs');
const multiplayerRunnerPath = resolve(frontendRoot, 'scripts/table-multiplayer-browser-acceptance.mjs');
const reconnectRunnerPath = resolve(frontendRoot, 'scripts/table-reconnect-browser-acceptance.mjs');
const packagePath = resolve(frontendRoot, 'package.json');

describe('table hardening browser acceptance skeleton', () => {
  it('defines the fixed viewports and structured browser artifacts', () => {
    expect(existsSync(runnerPath)).toBe(true);

    const runnerSource = readFileSync(runnerPath, 'utf8');
    const packageSource = readFileSync(packagePath, 'utf8');

    for (const viewport of ['360x640', '390x844', '430x932', '768x1024', '1280x720']) {
      expect(runnerSource).toContain(viewport);
    }

    for (const token of [
      'getBoundingClientRect',
      'elementFromPoint',
      'cardReadability',
      'cardFaceReadabilityProbe',
      'contrastRatio',
      'rankFits',
      'suitFits',
      "['10h', 'ad', 'ks', 'qc']",
      "new Set(['360x640', '1280x720'])",
      "page.on('console'",
      "page.on('pageerror'",
      'page.screenshot',
      'report.json',
      'consoleErrors',
      'rectangles',
      'clickHits',
      'blockingOverlaps',
      'viewportOverflows',
      'missedClicks.length',
      'blockingOverlaps.length',
      'viewportOverflows.length'
    ]) {
      expect(runnerSource).toContain(token);
    }

    expect(packageSource).toContain('test:browser:table-hardening');
  });

  it('covers a completed browser hand, replay navigation, and MySQL archive consistency', () => {
    expect(existsSync(replayRunnerPath)).toBe(true);

    const runnerSource = readFileSync(replayRunnerPath, 'utf8');
    const packageSource = readFileSync(packagePath, 'utf8');

    for (const token of [
      'DEPU_DSN',
      'DEPU_TEST_DATABASE',
      "getByRole('button', { name: '开 始' })",
      "getByRole('button', { name: '打开牌谱' })",
      "getByRole('button', { name: '下一动作' })",
      'hand_results',
      'hand_participants',
      'from actions',
      'archiveConsistency',
      'replay-acceptance-report.json',
      'page.screenshot'
    ]) {
      expect(runnerSource).toContain(token);
    }

    expect(packageSource).toContain('test:browser:table-replay');
  });

  it('covers two-account invitation, buy-in, socket actions, countdown, and polling audit', () => {
    expect(existsSync(multiplayerRunnerPath)).toBe(true);

    const runnerSource = readFileSync(multiplayerRunnerPath, 'utf8');
    const packageSource = readFileSync(packagePath, 'utf8');

    for (const token of [
      'DEPU_DSN',
      'DEPU_TEST_DATABASE',
      "getByLabel('邀请码')",
      "getByRole('button', { name: '加入房间' })",
      "getByRole('button', { name: '确定' })",
      "getByRole('button', { name: '开 始' })",
      'room.start_hand',
      'room.action',
      'not_your_turn',
      '行动倒计时',
      'current-hand',
      'requestAudit',
      'socketSynchronization',
      'multiplayer-acceptance-report.json',
      'page.screenshot',
    ]) {
      expect(runnerSource).toContain(token);
    }

    expect(packageSource).toContain('test:browser:table-multiplayer');
  });

  it('covers three consecutive hands, automatic starts, insufficient players, and authoritative records', () => {
    expect(existsSync(multiplayerRunnerPath)).toBe(true);

    const runnerSource = readFileSync(multiplayerRunnerPath, 'utf8');

    for (const token of [
      'targetHandCount = 3',
      'completedHands',
      'automaticStarts',
      'insufficientPlayers',
      'duplicateHandIds',
      'walletConsistency',
      'leaderboardConsistency',
      'historyConsistency',
      'settledHands === targetHandCount',
    ]) {
      expect(runnerSource).toContain(token);
    }
  });

  it('covers reconnect snapshots in waiting, playing, and settlement windows without command replay', () => {
    expect(existsSync(reconnectRunnerPath)).toBe(true);

    const runnerSource = readFileSync(reconnectRunnerPath, 'utf8');
    const packageSource = readFileSync(packagePath, 'utf8');

    for (const token of [
      'waitingReconnect',
      'playingReconnect',
      'settlementReconnect',
      'room.snapshot',
      'room.start_hand',
      'room.action',
      'unacknowledgedCommands',
      'replayedCommands',
      'reconnect-acceptance-report.json',
      'page.screenshot',
    ]) {
      expect(runnerSource).toContain(token);
    }

    expect(packageSource).toContain('test:browser:table-reconnect');
  });

  it('covers the full responsive table scenario matrix with screenshots and structured hit results', () => {
    expect(existsSync(responsiveRunnerPath)).toBe(true);

    const runnerSource = readFileSync(responsiveRunnerPath, 'utf8');
    const packageSource = readFileSync(packagePath, 'utf8');

    for (const viewport of ['360x640', '390x844', '430x932', '768x1024', '1280x720']) {
      expect(runnerSource).toContain(viewport);
    }

    for (const scenario of [
      'waiting-nine-seats',
      'buy-in-modal',
      'playing-amount-controls',
      'playing-five-board-cards',
      'drawer-settings',
      'drawer-chat',
      'drawer-score',
      'drawer-replay',
    ]) {
      expect(runnerSource).toContain(scenario);
    }

    for (const token of [
      'DEPU_DSN',
      'DEPU_TEST_DATABASE',
      'blockingOverlaps',
      'viewportOverflows',
      'clickHits',
      'page.screenshot',
      'responsive-acceptance-report.json',
    ]) {
      expect(runnerSource).toContain(token);
    }

    expect(packageSource).toContain('test:browser:table-responsive');
  });
});
