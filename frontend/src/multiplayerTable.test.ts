import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import roomSource from './pages/RoomPage.vue?raw';
import roomInfoSource from './pages/RoomInfoPage.vue?raw';
import roomPlayersSource from './pages/RoomPlayersPage.vue?raw';
import lobbySource from './pages/LobbyPage.vue?raw';
import meSource from './pages/MePage.vue?raw';
import createMatchSource from './pages/CreateMatchPage.vue?raw';
import shopSource from './pages/ShopPage.vue?raw';
import buyInModalSource from './components/BuyInModal.vue?raw';
import tableChatPanelSource from './components/TableChatPanel.vue?raw';
import tableScorePanelSource from './components/TableScorePanel.vue?raw';
import tableReplayPanelSource from './components/TableReplayPanel.vue?raw';
import tableSettingsPanelSource from './components/TableSettingsPanel.vue?raw';
import appStateSource from './composables/useAppState.ts?raw';

const styleSource = readFileSync(resolve(process.cwd(), 'src/style.css'), 'utf-8');

describe('multiplayer table contract', () => {
  it('provides a mockup-driven create match flow', () => {
    [
      '创建比赛',
      '训练赛',
      'SNG',
      '国际扑克',
      '短牌',
      '奥马哈',
      '比赛牌局名字',
      'Ante设置',
      '单次最小带入',
      '带入记分牌上限',
      '训练时长',
      '单桌最大人数',
      '确 定 创 建',
      'doCreateRoom',
      'mode: mode.value',
      'variant: variant.value',
      'ruleSetForVariant',
      "router.push(`/room/${created.id}`)"
    ].forEach((token) => expect(createMatchSource).toContain(token));

    expect(createMatchSource).not.toContain("variant: 'short_holdem',");
    expect(lobbySource).toContain("router.push('/create-match')");
  });

  it('provides a gold shop backed by simulated recharge', () => {
    [
      '商城',
      '金币',
      '钻石',
      'VIP卡',
      '装扮',
      '道具',
      '课程',
      '普通卡',
      'fetchRechargeOptions',
      'doRecharge',
      'simulateRecharge',
      'pendingRecharge',
      'confirmRecharge',
      '确认模拟充值',
      'consumeShopReturnTo',
      "router.push(returnPath.value || '/me')"
    ].forEach((token) => expect(shopSource).toContain(token));

    expect(shopSource).not.toContain('window.confirm');

    expect(meSource).toContain("router.push('/shop')");
  });

  it('opens a table buy-in modal before taking a seat', () => {
    [
      '补充记分牌',
      '在下一手开始前，为您补充记分牌',
      '带入记分牌',
      '消耗',
      '可用',
      '前往商城购买金币',
      "emit('confirm'",
      "emit('shop')"
    ].forEach((token) => expect(buyInModalSource).toContain(token));

    [
      'BuyInModal',
      'pendingSeatNo',
      'openBuyInModal',
      ':data-testid="`table-seat-${seatNo}`"',
      ':aria-label="roomSeat(seatNo)?.userId ? `座位 ${seatNo} ${roomSeat(seatNo)?.nickname}` : `坐下 座位 ${seatNo}`"',
      'confirmBuyIn',
      'setShopReturnTo',
      "router.push('/shop')"
    ].forEach((token) => expect(roomSource).toContain(token));

    expect(styleSource).toContain('pointer-events: none;');
    expect(styleSource).toContain('calc(var(--table-top-safe) + var(--table-seat-ring-y))');
    expect(styleSource).toContain('var(--table-seat-ring-x)');
    expect(styleSource).toContain('var(--table-seat-bottom-safe);');
  });

  it('keeps waiting table owner controls clickable above the seat ring', () => {
    expect(roomSource).toContain('owner-action-row owner-action-row-dock');
    expect(styleSource).toContain('pointer-events: none;');
    expect(styleSource).toContain('transform: translate(-50%, -50%);\n  pointer-events: auto;');
    expect(styleSource).toContain('.owner-action-row-dock {\n  z-index: var(--table-layer-actions);');
  });

  it('wires table invite and start controls to visible state changes', () => {
    [
      'tableNotice',
      'async function inviteFriendFromTable',
      'copyTableInvite(inviteCode, navigator.clipboard)',
      '@click="inviteFriendFromTable"',
      'await refreshCurrentRoomHand();',
      'table-feedback',
      'error'
    ].forEach((token) => expect(roomSource).toContain(token));

    [
      'startRoomFromInfo',
      'await refreshProfile();',
      'await connectRoomSocket(room.value.id);',
      'await doStartRoomHand();',
      'router.push(`/room/${room.value.id}`)'
    ].forEach((token) => expect(roomInfoSource).toContain(token));

    expect(roomInfoSource).not.toContain('await refreshCurrentRoomHand();');
  });

  it('renders large custom poker cards on the multiplayer table', () => {
    [
      "import { cardFaceVisual, holeCardVisuals } from '../pokerVisuals';",
      'cardFaceVisual(card).rankLabel',
      'cardFaceVisual(card).suitSymbol',
      'cardFaceVisual(card).colorClass',
      'cardFaceVisual(card).ariaLabel',
      'class="board-card table-card-face"',
      'class="card-rank"',
      'class="card-suit"',
      'class="hole-card table-card-face hero-card-face"'
    ].forEach((token) => expect(roomSource).toContain(token));

    expect(roomSource).not.toContain('cardImagePath(card)');

    [
      '.table-card-face {',
      '.card-rank {',
      '.card-suit {',
      'font-size: 34px;',
      '.hero-card-face {',
      'width: max(var(--table-hero-card-min-width), 58px);',
      '.hero-card-face .card-suit'
    ].forEach((token) => expect(styleSource).toContain(token));
  });

  it('keeps table action, board, seats, and bottom tools in separate safe zones', () => {
    [
      'table-topbar-compact',
      'table-center-stack-compact',
      'table-room-watermark',
      'hero-actions-safe',
      'mock-bottom-toolbar-safe'
    ].forEach((token) => expect(roomSource).toContain(token));

    [
      '--table-top-safe',
      '--table-bottom-safe',
      '.hero-actions-safe',
      'bottom: var(--table-actions-bottom-safe);',
      '.table-center-stack-compact',
      'top: var(--table-center-top);',
      '.casino-seat-node .seat-name',
      'text-overflow: ellipsis;',
      '.mock-bottom-toolbar-safe'
    ].forEach((token) => expect(styleSource).toContain(token));
  });

  it('opens short deck rules from table settings instead of leaving a dead button', () => {
    expect(tableSettingsPanelSource).toContain('short-deck-rules');
    expect(tableSettingsPanelSource).toContain("emit('shortDeckRules')");
    expect(roomSource).toContain('shortDeckRulesOpen');
    expect(roomSource).toContain('短牌规则说明');
    expect(roomSource).toContain('36 张牌');
    expect(roomSource).toContain('A6789');
  });

  it('uses room leave flow for exiting the match and reacts to settlement room updates', () => {
    expect(appStateSource).toContain('leaveRoom');
    expect(appStateSource).toContain('async function doLeaveRoom()');
    expect(appStateSource).toContain("roomSocket?.send('room.leave'");
    expect(roomSource).toContain('doLeaveRoom');
    expect(roomSource).toContain('await doLeaveRoom()');
    expect(appStateSource).toContain("roomSocket.on('room.updated', applyRoomSocketPayload)");
  });

  it('does not block room navigation when socket connection is unavailable after create or join', () => {
    [
      'connectRoomSocketBestEffort',
      'await connectRoomSocketBestEffort(room.value.id)',
      'error.value = previousError'
    ].forEach((token) => expect(appStateSource).toContain(token));

    expect(appStateSource).not.toContain('await connectRoomSocket(room.value.id);');
  });

  it('provides in-table chat score replay and settings panels', () => {
    ['常用语', '聊天记录', '请输入聊天内容，上限40个汉字', '发送', 'sendRoomChat'].forEach((token) => expect(tableChatPanelSource).toContain(token));
    ['当前战绩', '剩余时间', '昵称', '带入', '手数', '战绩', '观众'].forEach((token) => expect(tableScorePanelSource).toContain(token));
    ['牌谱回顾', '回放', '暂无已结算牌谱', '重试加载历史', '收藏（暂未开放）', '投诉（暂未开放）', 'fetchRoomHands', 'fetchRoomHandReplay'].forEach((token) => expect(tableReplayPanelSource).toContain(token));
    ['桌面设置（暂未开放）', '站起围观', '带入记分牌', '比赛设置（暂未开放）', '短牌规则', '保位离座（暂未开放）', '降落伞（暂未开放）', '退出比赛'].forEach((token) => expect(tableSettingsPanelSource).toContain(token));
  });

  it('shows concrete replay steps inside the table replay drawer', () => {
    [
      'watch(',
      'replayStep',
      'stepActionText',
      '公共牌',
      '玩家明细',
      '步骤 #{{ replayStep.seq }} · {{ stepIndex + 1 }}/{{ replay.steps.length }}',
      '上一动作',
      '下一动作',
      'replayStep.players'
    ].forEach((token) => expect(tableReplayPanelSource).toContain(token));

    expect(tableReplayPanelSource).not.toContain('<p v-if="replay">步骤 {{ replay.steps.length }}</p>');
  });

  it('keeps room subpages reloadable by restoring token and room context from route', () => {
    [
      "sessionStorage.getItem",
      "sessionStorage.setItem",
      "sessionStorage.removeItem",
      "setToken(payload.token)",
    ].forEach((token) => expect(appStateSource).toContain(token));

    [
      'ensureRouteRoom',
      "room.value = emptyRoom(route.params.roomId)",
      'await refreshRoom()',
    ].forEach((token) => expect(roomInfoSource).toContain(token));

    [
      'ensureRouteRoom',
      "room.value = emptyRoom(route.params.roomId)",
      'await refreshRoom()',
    ].forEach((token) => expect(roomPlayersSource).toContain(token));

    expect(roomInfoSource).not.toContain("router.push('/lobby')");
    expect(roomPlayersSource).not.toContain("router.push('/lobby')");
  });

  it('does not surface an expected missing current hand as a lobby error', () => {
    [
      'safeRefreshCurrentRoomHand',
      'current hand not found',
      'currentRoomHand.value = null',
      'refreshCurrentRoomHand: safeRefreshCurrentRoomHand'
    ].forEach((token) => expect(appStateSource).toContain(token));
  });

  it('clears a stale persisted token when profile refresh is unauthorized', () => {
    [
      'authentication required',
      'doLogout()',
      'throw err'
    ].forEach((token) => expect(appStateSource).toContain(token));
  });

  it('clears stale room errors when returning to the lobby', () => {
    expect(appStateSource).toContain('function clearError()');
    expect(lobbySource).toContain('clearError');
    expect(lobbySource).toContain('clearError();');
  });

  it('keeps the main room page focused on table actions and table state', () => {
    [
      'room-shell-fullscreen',
      '返回',
      'router.back()',
      'refreshCurrentRoomHand',
      'doRoomAction',
      'currentRoomHand',
      'myRoomSeat',
		'myRoomHandPlayer',
		'doTakeSeat',
		'sitAtFirstOpenSeat',
		'roomSeat(seatNo)?.userId',
		'v-if="myRoomSeat"',
		'router.push(`/room/${room.id}/players`)',
      'router.push(`/room/${room.id}/info`)',
      'board-zone',
      'seat-ring',
      'casino-table',
      'seat-ring-casino',
      '坐下 #',
      '选座位',
      'hero-actions-dock'
    ].forEach((token) => expect(roomSource).toContain(token));

    [
      '当前战绩',
      '观众（',
      'player-board-table',
      '座位操作'
    ].forEach((token) => expect(roomSource).not.toContain(token));
  });

  it('moves room metadata and owner controls into room info page', () => {
    [
      '房间信息',
      '邀请码',
      '状态',
      '房主',
      '房主开局'
    ].forEach((token) => expect(roomInfoSource).toContain(token));
  });

  it('moves player list and spectator info into room players page', () => {
    [
      '当前战绩',
      '人数',
      '观众（',
      'player-board-table',
      'live-score-table',
      'audience-grid',
      '房主坐下',
      '选择一个空位即可上桌',
      '坐下 #'
    ].forEach((token) => expect(roomPlayersSource).toContain(token));
  });

  it('uses socket commands for formal room start and player actions', () => {
    [
      'createRoomSocketClient',
      "room.start_hand",
      "room.action",
      "room.snapshot",
      "hand.started",
      "hand.updated",
      "hand.settled",
      "wallet.updated",
      'connectRoomSocket'
    ].forEach((token) => expect(appStateSource).toContain(token));

    expect(appStateSource).not.toContain('startRoomHand,');
    expect(appStateSource).not.toContain('submitRoomAction,');
    expect(appStateSource).not.toContain('currentRoomHand.value = await startRoomHand');
    expect(appStateSource).not.toContain('currentRoomHand.value = await submitRoomAction');
  });

  it('surfaces V1.1 realtime table experience state on the room page', () => {
    [
      'roomPresence',
      'actionLog',
      'chatMessages',
      'roomLeaderboard',
      'sendRoomChat',
      'hand.log.appended',
      'player.presence.updated',
      'chat.message',
      'room.leaderboard.updated',
      "chat.send"
    ].forEach((token) => expect(appStateSource).toContain(token));

    [
      'remainingActionSeconds',
      '行动倒计时',
      'nowMs',
      'window.setInterval',
      'window.clearInterval',
      '在线',
      '离线',
      '动作日志',
      '房间战绩榜',
      '聊天表情',
      'chatInput',
      'sendChat',
      'sendEmoji'
    ].forEach((token) => expect(roomSource).toContain(token));

    expect(roomSource).not.toContain('startRoomPolling');
    expect(roomSource).not.toContain('stopRoomPolling');
  });

  it('lets players choose an action amount and sends it with socket actions', () => {
    [
      'actionAmount',
      'minActionAmount',
      'maxActionAmount',
      'canChooseActionAmount',
      'type="range"',
      'v-model.number="actionAmount"',
      '执行 {{ actionLabel(action) }}',
      'submitRoomAction(action)'
    ].forEach((token) => expect(roomSource).toContain(token));

    expect(appStateSource).toContain('async function doRoomAction(action: string, amount = 0)');
    expect(appStateSource).toContain("roomSocket?.send('room.action', room.value!.id, { action, amount })");
    expect(appStateSource).not.toContain("roomSocket?.send('room.action', room.value!.id, { action, amount: 0 })");
  });

  it('handles a disconnected action command without an unhandled page rejection', () => {
    [
      'async function submitRoomAction(action: string)',
      'try {',
      'await doRoomAction(action, amount);',
      'catch {',
      "tableNotice.value = '操作未确认，请根据最新牌局状态重试';"
    ].forEach((token) => expect(roomSource).toContain(token));
  });

  it('provides table exits and room-specific history from the table surface', () => {
    [
      "router.push('/lobby')",
      "tableNotice.value = '退出比赛失败，请重试'",
      'data-testid="table-leave-button"',
      'data-testid="table-lobby-button"',
      '离开牌桌',
      '返回大厅',
      '牌桌历史',
      'room-history-preview',
      'recentRoomHands.slice(0, 3)',
      '`/room/${room.id}/hands/${hand.handId}/replay`'
    ].forEach((token) => expect(roomSource).toContain(token));
  });

  it('prevents already seated users from taking another seat from the players page', () => {
    [
      'disabled: true',
      '你已在其他座位',
      'seat.userId !== myRoomSeat.value?.userId'
    ].forEach((token) => expect(roomPlayersSource).toContain(token));
  });

  it('keeps V1.1 log leaderboard and chat tools outside the table surface', () => {
    const tableSurfaceClose = '</div>\n\n      <nav class="mock-bottom-toolbar mock-bottom-toolbar-safe" aria-label="底部工具栏">';
    const tableTools = '<aside class="table-side-panel v11-table-tools" aria-label="牌桌工具">';

    expect(roomSource).toContain(tableSurfaceClose);
    expect(roomSource).toContain(tableTools);
    expect(roomSource.indexOf(tableTools)).toBeGreaterThan(roomSource.indexOf(tableSurfaceClose));
    expect(styleSource).toContain('overflow-y: auto;');
    expect(styleSource).toContain('grid-template-columns: repeat(3, minmax(0, 1fr));');
    expect(styleSource).toContain('grid-template-columns: 1fr;');
    expect(styleSource).not.toContain('.room-shell-fullscreen {\n  min-height: 100dvh;\n  padding: 0;\n  overflow: hidden;');
  });
});
