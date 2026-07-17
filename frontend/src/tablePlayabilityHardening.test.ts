import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import roomSource from './pages/RoomPage.vue?raw';
import roomInfoSource from './pages/RoomInfoPage.vue?raw';
import appStateSource from './composables/useAppState.ts?raw';
import chatPanelSource from './components/TableChatPanel.vue?raw';
import replayPanelSource from './components/TableReplayPanel.vue?raw';
import settingsPanelSource from './components/TableSettingsPanel.vue?raw';
import tableDrawerSource from './components/TableDrawer.vue?raw';
import buyInModalSource from './components/BuyInModal.vue?raw';
import { roomStartHandUnavailableReason, startHandErrorMessage } from './composables/useAppState';
import tableInviteSource from './tableInvite.ts?raw';

const styleSource = readFileSync(resolve(process.cwd(), 'src/style.css'), 'utf8');

interface ContractSources {
  room: string;
  roomInfo: string;
  appState: string;
  chatPanel: string;
  replayPanel: string;
  settingsPanel: string;
  tableDrawer: string;
  buyInModal: string;
  tableInvite: string;
  style: string;
}

const currentSources: ContractSources = {
  room: roomSource,
  roomInfo: roomInfoSource,
  appState: appStateSource,
  chatPanel: chatPanelSource,
  replayPanel: replayPanelSource,
  settingsPanel: settingsPanelSource,
  tableDrawer: tableDrawerSource,
  buyInModal: buyInModalSource,
  tableInvite: tableInviteSource,
  style: styleSource
};

function requireToken(source: string, token: string, contract: string) {
  if (!source.includes(token)) throw new Error(`${contract} 缺少 ${token}`);
}

function requirePattern(source: string, pattern: RegExp, contract: string) {
  if (!pattern.test(source)) throw new Error(`${contract} 缺少 ${pattern}`);
}

function tableLayerValue(source: string, property: string, contract: string) {
  const match = source.match(new RegExp(`${property}:\\s*(\\d+);`));
  if (!match) throw new Error(`${contract} 缺少 ${property}`);
  return Number(match[1]);
}

function assertTableSafetyContract(sources: ContractSources) {
  const contract = '牌桌安全区契约';
  [
    'table-topbar-compact',
    'table-center-stack-compact',
    'seat-ring-casino',
    'hero-panel-clean',
    'hero-actions-safe',
    'mock-bottom-toolbar-safe'
  ].forEach((token) => requireToken(sources.room, token, contract));

  [
    '--table-top-safe:',
    '--table-bottom-safe:',
    '--table-center-top:',
    '--table-seat-bottom-safe:',
    '--table-hero-bottom-safe:',
    '--table-actions-bottom-safe:',
    '--table-layer-stage:',
    '--table-layer-center:',
    '--table-layer-seats:',
    '--table-layer-toolbar:',
    '--table-layer-hero:',
    '--table-layer-actions:',
    '--table-layer-topbar:'
  ].forEach((token) => requireToken(sources.style, token, contract));

  const layers = [
    '--table-layer-stage',
    '--table-layer-center',
    '--table-layer-seats',
    '--table-layer-toolbar',
    '--table-layer-hero',
    '--table-layer-actions',
    '--table-layer-topbar'
  ].map((property) => tableLayerValue(sources.style, property, contract));
  expect(layers).toEqual([...layers].sort((left, right) => left - right));
  requirePattern(
    sources.style,
    /\.casino-table\s*\{[^}]*z-index:\s*var\(--table-layer-stage\);[^}]*isolation:\s*isolate;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.seat-ring-casino\s*\{[^}]*inset:\s*calc\(var\(--table-top-safe\) \+ var\(--table-seat-ring-y\)\)[^;]*var\(--table-seat-ring-x\)[^;]*var\(--table-seat-bottom-safe\);[^}]*z-index:\s*var\(--table-layer-seats\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-center-stack-compact\s*\{[^}]*top:\s*var\(--table-center-top\);[^}]*z-index:\s*var\(--table-layer-center\);[^}]*pointer-events:\s*none;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.hero-panel-clean\s*\{[^}]*bottom:\s*var\(--table-hero-bottom-safe\);[^}]*z-index:\s*var\(--table-layer-hero\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.hero-actions-safe\s*\{[^}]*bottom:\s*var\(--table-actions-bottom-safe\);[^}]*z-index:\s*var\(--table-layer-actions\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.owner-action-row\s*\{[^}]*z-index:\s*var\(--table-layer-actions\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.mock-bottom-toolbar-safe\s*\{[^}]*min-height:\s*var\(--table-bottom-safe\);[^}]*z-index:\s*var\(--table-layer-toolbar\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-topbar-compact\s*\{[^}]*z-index:\s*var\(--table-layer-topbar\);/s,
    contract
  );
}

function assertCompactTopbarContract(sources: ContractSources) {
  const contract = '顶部紧凑布局契约';
  [
    'class="topbar-float table-player-count"',
    'class="topbar-float table-invite-code"',
    'class="topbar-float table-room-name"',
    ':title="`玩家 ${occupiedSeatCount}/${room.seatCount || room.members.length}`"',
    ':title="`邀请码 ${room.inviteCode || \'加载中\'}`"',
    ':title="room.name || \'房间信息\'"'
  ].forEach((token) => requireToken(sources.room, token, contract));

  [
    '--table-topbar-height:',
    '--table-status-top:',
    '--table-status-height:',
    '--table-top-safe: calc(var(--table-status-top) + var(--table-status-height) + 8px);'
  ].forEach((token) => requireToken(sources.style, token, contract));

  requirePattern(
    sources.style,
    /\.table-topbar-compact\s*\{[^}]*grid-template-columns:\s*repeat\(6,\s*minmax\(0,\s*1fr\)\);[^}]*grid-template-rows:\s*var\(--table-topbar-height\);[^}]*overflow:\s*hidden;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.subtle-strip\s*\{[^}]*top:\s*var\(--table-status-top\);[^}]*min-height:\s*var\(--table-status-height\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-player-count,\s*\.table-invite-code,\s*\.table-room-name\s*\{[^}]*min-width:\s*0;[^}]*overflow:\s*hidden;[^}]*text-overflow:\s*ellipsis;[^}]*white-space:\s*nowrap;/s,
    contract
  );
}

function assertCenterLayerContract(sources: ContractSources) {
  const contract = '公共牌中心层契约';
  [
    'class="table-room-center table-room-watermark table-watermark-low"',
    'class="board-zone clean-board-zone table-center-stack table-center-stack-compact table-center-layer"',
    'class="community-cards table-community-grid"',
    'class="pot-chip table-pot-chip table-pot-below-board"'
  ].forEach((token) => requireToken(sources.room, token, contract));

  [
    '--table-center-width:',
    '--table-center-card-gap:',
    '--table-watermark-top:'
  ].forEach((token) => requireToken(sources.style, token, contract));

  requirePattern(
    sources.style,
    /\.table-center-layer\s*\{[^}]*width:\s*min\(var\(--table-center-width\),\s*calc\(100% - 24px\)\);[^}]*grid-template-rows:\s*minmax\(64px,\s*auto\) auto;[^}]*align-content:\s*start;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-community-grid\s*\{[^}]*display:\s*grid;[^}]*grid-template-columns:\s*repeat\(5,\s*minmax\(0,\s*1fr\)\);[^}]*gap:\s*var\(--table-center-card-gap\);[^}]*width:\s*100%;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-community-grid > \.muted\s*\{[^}]*grid-column:\s*1 \/ -1;[^}]*align-self:\s*center;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-pot-below-board\s*\{[^}]*position:\s*relative;[^}]*justify-self:\s*center;[^}]*margin-top:\s*0;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-watermark-low\s*\{[^}]*top:\s*var\(--table-watermark-top\);[^}]*z-index:\s*0;[^}]*max-height:[^;]+;[^}]*overflow:\s*hidden;[^}]*opacity:\s*0\.24;/s,
    contract
  );
  requirePattern(
    sources.style,
    /@media \(max-width:\s*760px\)\s*\{[^}]*\.room-mobile-screen\s*\{[^}]*--table-center-width:\s*230px;[^}]*--table-center-card-gap:\s*4px;/s,
    contract
  );
  requirePattern(
    sources.style,
    /@media \(max-width:\s*760px\)[\s\S]*?\.table-center-stack-compact \.board-card\s*\{[^}]*width:\s*var\(--table-community-card-min-width\);[^}]*height:\s*auto;/s,
    contract
  );
}

function assertBottomInteractionContract(sources: ContractSources) {
  const contract = '底部交互安全区契约';
  [
    'class="table-bottom-interaction table-bottom-interaction-safe"',
    'class="hero-panel hero-panel-clean"',
    'class="actions hero-actions hero-actions-dock hero-actions-safe"',
    'class="action-amount-control" v-if="canChooseActionAmount"'
  ].forEach((token) => requireToken(sources.room, token, contract));

  [
    '--table-interaction-gap:',
    '--table-interaction-hero-width:',
    '--table-action-button-min-width:'
  ].forEach((token) => requireToken(sources.style, token, contract));

  requirePattern(
    sources.style,
    /\.table-bottom-interaction-safe\s*\{[^}]*position:\s*absolute;[^}]*bottom:\s*calc\(var\(--table-bottom-safe\) \+ var\(--table-interaction-gap\)\);[^}]*display:\s*grid;[^}]*grid-template-columns:\s*var\(--table-interaction-hero-width\) minmax\(0,\s*1fr\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-bottom-interaction-safe \.hero-panel-clean\s*\{[^}]*position:\s*static;[^}]*transform:\s*none;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-bottom-interaction-safe \.hero-actions-safe\s*\{[^}]*position:\s*static;[^}]*display:\s*grid;[^}]*grid-template-columns:\s*repeat\(auto-fit,\s*minmax\(min\(100%,\s*var\(--table-action-button-min-width\)\),\s*1fr\)\);[^}]*overflow:\s*visible;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-bottom-interaction-safe \.action-amount-control\s*\{[^}]*grid-column:\s*1 \/ -1;[^}]*width:\s*100%;/s,
    contract
  );
  requirePattern(
    sources.style,
    /@media \(max-width:\s*760px\)[\s\S]*?\.room-mobile-screen\s*\{[^}]*--table-interaction-hero-width:\s*116px;[^}]*--table-action-button-min-width:\s*64px;/s,
    contract
  );
}

function assertOverlayInteractionContract(sources: ContractSources) {
  const contract = '抽屉与模态框交互契约';

  [
    'role="dialog"',
    'aria-modal="true"',
    'class="table-drawer-content"',
    'class="drawer-close" aria-label="关闭抽屉"'
  ].forEach((token) => requireToken(sources.tableDrawer, token, contract));

  [
    'class="modal-backdrop" @click.self="emit(\'close\')"',
    'class="buy-in-modal" role="dialog" aria-modal="true"'
  ].forEach((token) => requireToken(sources.buyInModal, token, contract));

  ['--table-layer-drawer:', '--table-layer-modal:'].forEach((token) => requireToken(sources.style, token, contract));

  const drawerLayer = tableLayerValue(sources.style, '--table-layer-drawer', contract);
  const modalLayer = tableLayerValue(sources.style, '--table-layer-modal', contract);
  expect(modalLayer).toBeGreaterThan(drawerLayer);

  requirePattern(
    sources.style,
    /\.table-drawer-backdrop\s*\{[^}]*z-index:\s*var\(--table-layer-drawer,\s*35\);[^}]*overscroll-behavior:\s*contain;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-drawer\s*\{[^}]*display:\s*grid;[^}]*grid-template-rows:\s*auto minmax\(0,\s*1fr\);[^}]*max-height:\s*100dvh;[^}]*overflow:\s*hidden;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-drawer-content\s*\{[^}]*min-height:\s*0;[^}]*overflow-y:\s*auto;[^}]*overscroll-behavior:\s*contain;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.drawer-close\s*\{[^}]*position:\s*sticky;[^}]*z-index:\s*1;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.modal-backdrop\s*\{[^}]*z-index:\s*var\(--table-layer-modal,\s*40\);[^}]*overflow-y:\s*auto;[^}]*overscroll-behavior:\s*contain;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.buy-in-modal,\s*\.short-deck-rules-modal\s*\{[^}]*max-height:\s*calc\(100dvh - 36px\);[^}]*overflow-y:\s*auto;[^}]*overscroll-behavior:\s*contain;/s,
    contract
  );
}

function assertCriticalFeedbackContract(sources: ContractSources) {
  const contract = '关键按钮反馈契约';
  [
    'async function inviteFriendFromTable()',
    'await copyTableInvite(inviteCode, navigator.clipboard)',
    'data-testid="table-invite-feedback"',
    ':disabled="loading || !!startHandUnavailableReason"',
    '@click="startHandFromTable"',
    'await doStartRoomHand();',
    'const startHandUnavailableReason = computed(',
    ':disabled="loading || !!startHandUnavailableReason"',
    'v-if="tableNotice"',
    'catch {',
    'class="table-feedback"',
    '{{ tableNotice }}'
  ].forEach((token) => requireToken(sources.room, token, contract));

  [
    'export function tableInviteText(inviteCode: string)',
    '邀请你加入 DePu 房间，邀请码：${inviteCode}',
    'await clipboard.writeText(tableInviteText(inviteCode))',
    'return `邀请码已复制：${inviteCode}`',
    'return `邀请码：${inviteCode}`'
  ].forEach((token) => requireToken(sources.tableInvite, token, contract));

  [
    'async function startRoomFromInfo()',
    'await doStartRoomHand();',
    'router.push(`/room/${room.value.id}`)',
    'const startHandUnavailableReason = computed(',
    ':disabled="loading || !!startHandUnavailableReason"',
    'v-if="startHandUnavailableReason"',
    'catch {',
    'v-if="error"'
  ].forEach((token) => requireToken(sources.roomInfo, token, contract));

  [
    'export function startHandErrorMessage(startError: unknown)',
    '只有房主可以开始牌局',
    '入座人数不足，无法开始牌局',
    '房间状态已变化',
    '网络或实时连接异常'
  ].forEach((token) => requireToken(sources.appState, token, contract));

  [
    '<button type="button" disabled title="保位离座暂未开放">保位离座（暂未开放）</button>',
    '<button type="button" disabled title="降落伞功能暂未开放">降落伞（暂未开放）</button>'
  ]
    .forEach((token) => requireToken(sources.settingsPanel, token, contract));
}

function assertVisibleButtonBehaviorContract(sources: ContractSources) {
  const contract = '牌桌可见按钮行为契约';
  [
    'async function leaveTable()',
    'closePanel();',
    "tableNotice.value = '退出比赛失败，请重试';",
    '<button type="button" disabled title="当前版本不支持手动解散比赛">解散比赛（暂未开放）</button>',
    ':disabled="!!roomSeat(seatNo)?.userId || !!myRoomSeat || loading"',
    'aria-label="打开设置"',
    'aria-label="离开牌桌"',
    'aria-label="打开战绩"',
    'aria-label="打开牌谱"',
    'aria-label="打开聊天"',
    'disabled title="语音功能暂未开放">麦克风（暂未开放）</button>',
    '@stand="standFromTable"',
    '@buy-in="buyInFromSettings"',
    ':can-stand="!!myRoomSeat"',
    ':can-buy-in="!myRoomSeat && !!firstOpenSeatNo"'
  ].forEach((token) => requireToken(sources.room, token, contract));

  expect(sources.room).not.toContain('async function leaveTable() {\n  try {\n    await doLeaveRoom();\n  } finally {');

  [
    'defineProps<{ canStand: boolean; canBuyIn: boolean; loading: boolean }>();',
    '<button type="button" disabled title="桌面设置暂未开放">桌面设置（暂未开放）</button>',
    ':disabled="loading || !canStand"',
    ':title="canStand ? \'站起围观\' : \'当前未入座\'"',
    ':disabled="loading || !canBuyIn"',
    ':title="canBuyIn ? \'带入记分牌\' : \'已入座玩家暂不支持中途补充记分牌\'"',
    '<button type="button" disabled title="比赛设置暂未开放">比赛设置（暂未开放）</button>',
    '<button type="button" disabled title="保位离座暂未开放">保位离座（暂未开放）</button>',
    '<button type="button" disabled title="降落伞功能暂未开放">降落伞（暂未开放）</button>',
    '<button type="button" :disabled="loading" @click="emit(\'leave\')">退出比赛</button>'
  ].forEach((token) => requireToken(sources.settingsPanel, token, contract));

  requireToken(
    sources.chatPanel,
    '<button type="button" class="quick-chat-tab" disabled title="常用语功能暂未开放">常用语（暂未开放）</button>',
    contract
  );

  [
    '<button type="button" disabled title="收藏功能暂未开放">收藏（暂未开放）</button>',
    '<button type="button" disabled title="投诉功能暂未开放">投诉（暂未开放）</button>'
  ].forEach((token) => requireToken(sources.replayPanel, token, contract));
}

function assertUnifiedCardContract(sources: ContractSources) {
  const contract = '统一牌面契约';
  [
    "import { cardFaceVisual, holeCardVisuals } from '../pokerVisuals';",
    'cardFaceVisual(card).rankLabel',
    'cardFaceVisual(card).suitSymbol',
    'cardFaceVisual(card).colorClass',
    'cardFaceVisual(card).ariaLabel',
    'class="board-card table-card-face"',
    'class="hole-card table-card-face hero-card-face"',
    'class="card-rank"',
    'class="card-suit"',
    'class="hole-card back-card table-card-back hero-card-face"'
  ].forEach((token) => requireToken(sources.room, token, contract));

  requirePattern(sources.style, /\.table-card-face\s*\{[^}]*position:\s*relative;/s, contract);
  requirePattern(sources.style, /\.card-rank\s*\{[^}]*top:[^;]+;[^}]*left:[^;]+;/s, contract);
  requirePattern(sources.style, /\.card-suit\s*\{[^}]*right:[^;]+;[^}]*bottom:[^;]+;[^}]*font-size:\s*34px;/s, contract);
  requirePattern(
    sources.style,
    /\.hero-card-face\s*\{[^}]*width:\s*max\(var\(--table-hero-card-min-width\),\s*58px\);[^}]*height:\s*auto;/s,
    contract
  );
  requireToken(sources.style, '.table-card-face.red {', contract);
}

function assertReadableCardSizingContract(sources: ContractSources) {
  const contract = '正式牌桌牌面尺寸契约';

  [
    '--table-card-aspect-ratio: 5 / 7;',
    '--table-community-card-min-width: 42px;',
    '--table-hero-card-min-width: 52px;'
  ].forEach((token) => requireToken(sources.style, token, contract));

  requirePattern(
    sources.style,
    /\.table-card-face\s*\{[^}]*aspect-ratio:\s*var\(--table-card-aspect-ratio\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.table-center-stack-compact \.board-card\s*\{[^}]*width:\s*max\(var\(--table-community-card-min-width\),\s*46px\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.hero-card-face\s*\{[^}]*width:\s*max\(var\(--table-hero-card-min-width\),\s*58px\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /@media \(max-width:\s*760px\)[\s\S]*?\.table-center-stack-compact \.board-card\s*\{[^}]*width:\s*var\(--table-community-card-min-width\);/s,
    contract
  );
  requirePattern(
    sources.style,
    /@media \(max-width:\s*760px\)[\s\S]*?\.table-bottom-interaction-safe \.hero-card-face\s*\{[^}]*width:\s*var\(--table-hero-card-min-width\);/s,
    contract
  );
}

function assertPrivateCardBackContract(sources: ContractSources) {
  const contract = '统一牌背隐私契约';
  [
    "import { cardFaceVisual, holeCardVisuals } from '../pokerVisuals';",
    'const heroHoleCards = computed(() => holeCardVisuals(myRoomHandPlayer.value?.holeCards));',
    'v-for="(card, index) in heroHoleCards"',
    "card.kind === 'face'",
    'class="hole-card back-card table-card-back hero-card-face"',
    'class="card-back-grid" aria-hidden="true"'
  ].forEach((token) => requireToken(sources.room, token, contract));

  expect(sources.room).not.toContain('<img');
  requirePattern(
    sources.style,
    /\.table-card-back\s*\{[^}]*background:\s*[\s\S]*linear-gradient[^}]*;/s,
    contract
  );
  requirePattern(
    sources.style,
    /\.card-back-grid\s*\{[^}]*background:\s*[\s\S]*repeating-linear-gradient[^}]*;/s,
    contract
  );
}

function assertReplayStepContract(sources: ContractSources) {
  const contract = '牌谱步骤契约';
  [
    'onMounted(() => {',
    'void refreshHistory();',
    'if (items.length) await loadReplay(0, items);',
    'const replayStep = computed',
    'stepIndex.value = 0;',
    '步骤 #{{ replayStep.seq }} · {{ stepIndex + 1 }}/{{ replay.steps.length }}',
    'stageLabel(replayStep.stage)',
    '底池 {{ replayStep.pot }}',
    '上一动作：{{ stepActionText(replayStep) }}',
    'v-for="card in displayCards(replayStep.boardCards)"',
    'class="table-card-face replay-card-face"',
    ':data-card="card"',
    'cardFaceVisual(card).rankLabel',
    'cardFaceVisual(card).suitSymbol',
    'v-for="player in replayStep.players"',
    'displayCards(player.holeCards)',
    '@click="previousStep"',
    '@click="nextStep"'
  ].forEach((token) => requireToken(sources.replayPanel, token, contract));
}

describe('table playability hardening behavior contract', () => {
  it('covers safety zones, critical feedback, unified cards, and replay steps', () => {
    assertTableSafetyContract(currentSources);
    assertCompactTopbarContract(currentSources);
    assertCenterLayerContract(currentSources);
    assertOverlayInteractionContract(currentSources);
    assertCriticalFeedbackContract(currentSources);
    assertVisibleButtonBehaviorContract(currentSources);
    assertUnifiedCardContract(currentSources);
    assertReplayStepContract(currentSources);
  });

  it('keeps formal hero and community cards readable at stable proportions', () => {
    assertReadableCardSizingContract(currentSources);
  });

  it('uses code-native card backs without exposing unavailable private cards', () => {
    assertPrivateCardBackContract(currentSources);
  });

  it('detects a missing bottom safe-area reservation', () => {
    const regressed = {
      ...currentSources,
      style: currentSources.style.replace('--table-bottom-safe: 106px;', '')
    };

    expect(() => assertTableSafetyContract(regressed)).toThrow('牌桌安全区契约');
  });

  it('detects topbar metadata overflow regressions', () => {
    const regressed = {
      ...currentSources,
      style: currentSources.style.replace(
        '.table-player-count,\n.table-invite-code,\n.table-room-name {\n  min-width: 0;',
        '.table-player-count,\n.table-invite-code,\n.table-room-name {'
      )
    };

    expect(() => assertCompactTopbarContract(regressed)).toThrow('顶部紧凑布局契约');
  });

  it('detects center board and watermark overlap regressions', () => {
    const regressed = {
      ...currentSources,
      style: currentSources.style.replace('opacity: 0.24;', 'opacity: 0.58;')
    };

    expect(() => assertCenterLayerContract(regressed)).toThrow('公共牌中心层契约');
  });

  it('keeps hero cards, amount controls, and variable actions above the toolbar', () => {
    assertBottomInteractionContract(currentSources);
  });

  it('detects action layouts that are not anchored to the bottom safety area', () => {
    const regressed = {
      ...currentSources,
      style: currentSources.style.replace(
        'bottom: calc(var(--table-bottom-safe) + var(--table-interaction-gap));',
        'bottom: 0;'
      )
    };

    expect(() => assertBottomInteractionContract(regressed)).toThrow('底部交互安全区契约');
  });

  it('detects overlays that can lose foreground clicks or scroll controls off-screen', () => {
    const regressed = {
      ...currentSources,
      style: currentSources.style.replace('z-index: var(--table-layer-modal, 40);', 'z-index: 1;')
    };

    expect(() => assertOverlayInteractionContract(regressed)).toThrow('抽屉与模态框交互契约');
  });

  it('detects silent invite fallback regressions', () => {
    const regressed = {
      ...currentSources,
      tableInvite: currentSources.tableInvite.replace('return `邀请码：${inviteCode}`;', '')
    };

    expect(() => assertCriticalFeedbackContract(regressed)).toThrow('关键按钮反馈契约');
  });

  it('detects enabled placeholder buttons without behavior', () => {
    const regressed = {
      ...currentSources,
      settingsPanel: currentSources.settingsPanel.replace(
        '<button type="button" disabled title="比赛设置暂未开放">比赛设置（暂未开放）</button>',
        '<button type="button">比赛设置</button>'
      )
    };

    expect(() => assertVisibleButtonBehaviorContract(regressed)).toThrow('牌桌可见按钮行为契约');
  });

  it('explains start-hand preconditions and command failures', () => {
    const waitingRoom = {
      id: 'room_feedback',
      inviteCode: 'ABC123',
      ownerUserId: 'owner_1',
      status: 'waiting' as const,
      minPlayersToStart: 2,
      members: [],
      seats: [{ seatNo: 1, seatStatus: 'occupied' as const, userId: 'owner_1' }]
    };

    expect(roomStartHandUnavailableReason(waitingRoom, 'player_2')).toBe('只有房主可以开始牌局');
    expect(roomStartHandUnavailableReason(waitingRoom, 'owner_1')).toContain('入座人数不足');
    expect(roomStartHandUnavailableReason({ ...waitingRoom, status: 'playing' }, 'owner_1')).toContain('房间状态已变化');
    expect(startHandErrorMessage(new Error('only room owner can start'))).toBe('只有房主可以开始牌局');
    expect(startHandErrorMessage(new Error('not enough players to start'))).toBe('入座人数不足，无法开始牌局');
    expect(startHandErrorMessage(new Error('room is not waiting'))).toContain('房间状态已变化');
    expect(startHandErrorMessage(new Error('socket closed'))).toContain('网络或实时连接异常');
  });

  it('detects card face regressions', () => {
    const regressed = {
      ...currentSources,
      room: currentSources.room.split('class="card-suit"').join('class="card-symbol"')
    };

    expect(() => assertUnifiedCardContract(regressed)).toThrow('统一牌面契约');
  });

  it('detects replay detail regressions', () => {
    const regressed = {
      ...currentSources,
      replayPanel: currentSources.replayPanel.replace('上一动作：{{ stepActionText(replayStep) }}', '')
    };

    expect(() => assertReplayStepContract(regressed)).toThrow('牌谱步骤契约');
  });
});
