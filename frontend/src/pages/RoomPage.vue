<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { actionLabel, statusLabel } from '../displayLabels';
import { isRedCard } from '../pokerVisuals';
import { emptyRoom, useAppState } from '../composables/useAppState';
import BuyInModal from '../components/BuyInModal.vue';
import TableDrawer from '../components/TableDrawer.vue';
import TableChatPanel from '../components/TableChatPanel.vue';
import TableScorePanel from '../components/TableScorePanel.vue';
import TableReplayPanel from '../components/TableReplayPanel.vue';
import TableSettingsPanel from '../components/TableSettingsPanel.vue';

const route = useRoute();
const router = useRouter();
const {
  room,
  currentRoomHand,
  myRoomSeat,
  myRoomHandPlayer,
  isMyTurn,
  loading,
  error,
  token,
  me,
  wallet,
  roomPresence,
  actionLog,
  chatMessages,
  roomLeaderboard,
  recentRoomHands,
  refreshProfile,
  refreshCurrentRoomHand,
  refreshRoom,
  doTakeSeat,
  doLeaveSeat,
  doLeaveRoom,
  doStartRoomHand,
  doRoomAction,
  connectRoomSocket,
  sendRoomChat,
  sendRoomEmoji,
  setShopReturnTo
} = useAppState();

const chatInput = ref('');
const activePanel = ref<'chat' | 'score' | 'replay' | 'settings' | null>(null);
const pendingSeatNo = ref<number | null>(null);
const shortDeckRulesOpen = ref(false);
const nowMs = ref(Date.now());
const actionClockOffsetMs = ref(0);
const actionAmount = ref(0);
const amountActions = ['bet', 'raise'];
const tableLatencyMs = ref(43);
const tableNotice = ref('');
let countdownTimer: number | null = null;

const occupiedSeatCount = computed(() => room.value?.seats.filter((seat) => seat.userId).length ?? 0);
const defaultBuyIn = computed(() => room.value?.seats.find((seat) => seat.buyInChips)?.buyInChips ?? 1000);
const isRoomOwner = computed(() => !!room.value && room.value.ownerUserId === me.value?.id);
const buyInModalOpen = computed(() => pendingSeatNo.value !== null);
const roomMinBuyIn = computed(() => room.value?.minBuyIn ?? defaultBuyIn.value ?? 1000);
const roomMaxBuyIn = computed(() => room.value?.maxBuyIn ?? Math.max(roomMinBuyIn.value, 6000));
const walletBalance = computed(() => wallet.value?.balance ?? me.value?.walletBalance ?? 0);
const firstOpenSeatNo = computed(() => {
	if (!room.value?.seatCount) return null;
	for (let seatNo = 1; seatNo <= room.value.seatCount; seatNo += 1) {
		if (!room.value.seats.find((seat) => seat.seatNo === seatNo && seat.userId)) return seatNo;
  }
  return null;
});

function seatPlayer(seatNo: number) {
	return currentRoomHand.value?.players.find((player) => player.seatNo === seatNo) ?? null;
}

function roomSeat(seatNo: number) {
	return room.value?.seats.find((seat) => seat.seatNo === seatNo) ?? null;
}

function presenceForUser(userId?: string | null) {
  if (!userId) return null;
  return roomPresence.value.find((item) => item.userId === userId) ?? null;
}

function displayCards(cards?: string[] | null) {
  if (!cards?.length) return [];
  return cards;
}

function cardRankLabel(card: string) {
  const normalized = card.trim().toUpperCase();
  if (!normalized) return '';
  const rank = normalized.slice(0, -1);
  return rank === 'T' ? '10' : rank;
}

function cardSuitSymbol(card: string) {
  const suit = card.trim().slice(-1).toLowerCase();
  if (suit === 's') return '♠';
  if (suit === 'h') return '♥';
  if (suit === 'd') return '♦';
  if (suit === 'c') return '♣';
  return suit.toUpperCase();
}

const remainingActionSeconds = computed(() => {
  const deadline = currentRoomHand.value?.actionDeadlineAt;
  if (!deadline) return null;
  const remaining = Math.ceil((new Date(deadline).getTime() - (nowMs.value + actionClockOffsetMs.value)) / 1000);
  return Math.max(0, remaining);
});

const maxActionAmount = computed(() => {
  const player = myRoomHandPlayer.value;
  if (!player) return 0;
  return Math.max(0, player.streetCommitted + player.stack);
});

const minActionAmount = computed(() => {
  const hand = currentRoomHand.value;
  const player = myRoomHandPlayer.value;
  if (!hand || !player) return 0;
  const minRaise = Math.max(1, hand.minRaise ?? 1);
  const rawMin = hand.availableActions.includes('raise') ? (hand.currentBet ?? 0) + minRaise : minRaise;
  return Math.min(maxActionAmount.value, Math.max(1, rawMin));
});

const canChooseActionAmount = computed(() => {
  const actions = currentRoomHand.value?.availableActions ?? [];
  return isMyTurn.value && actions.some((action) => amountActions.includes(action)) && maxActionAmount.value > 0;
});

function logText(entry: { kind: string; action?: string; seatNo?: number; source?: string }) {
  if (entry.kind === 'player_action') return `#${entry.seatNo ?? '-'} ${actionLabel(entry.action || '')}`;
  if (entry.kind === 'hand_started') return '新手牌开始';
  if (entry.kind === 'timeout_action') return `#${entry.seatNo ?? '-'} 超时`;
  return entry.kind;
}

function openPanel(panel: 'chat' | 'score' | 'replay' | 'settings') {
  activePanel.value = panel;
}

function closePanel() {
  activePanel.value = null;
}

function roomVariantLabel() {
  if (room.value?.variant === 'short_holdem' || room.value?.ruleSetId === 'short-deck') return '短牌';
  if (room.value?.variant === 'omaha') return '奥马哈';
  return '国际扑克';
}

function roomDisplayNo() {
  return room.value?.id.replace('room_', '').slice(-3) || '121';
}

function openBuyInModal(seatNo: number) {
  if (roomSeat(seatNo)?.userId) return;
  pendingSeatNo.value = seatNo;
}

function closeBuyInModal() {
  pendingSeatNo.value = null;
}

async function confirmBuyIn(amount: number) {
  if (!pendingSeatNo.value) return;
  await doTakeSeat(pendingSeatNo.value, amount);
  closeBuyInModal();
  await refreshRoom();
  await refreshCurrentRoomHand();
}

function goToShopFromBuyIn() {
  setShopReturnTo(router.currentRoute.value.fullPath);
  router.push('/shop');
}

async function startHandFromTable() {
  tableNotice.value = '';
  await doStartRoomHand();
  await refreshCurrentRoomHand();
  tableNotice.value = '牌局已开始';
}

async function inviteFriendFromTable() {
  if (!room.value) return;
  const inviteCode = room.value.inviteCode || room.value.id;
  const inviteText = `房间邀请码 ${inviteCode}`;
  try {
    await navigator.clipboard.writeText(inviteText);
    tableNotice.value = `邀请码已复制：${inviteCode}`;
  } catch {
    tableNotice.value = `邀请码：${inviteCode}`;
  }
}

async function sendChat() {
  const text = chatInput.value;
  chatInput.value = '';
  await sendRoomChat(text);
}

async function sendEmoji(emojiCode: string) {
  await sendRoomEmoji(emojiCode);
}

async function sitAtFirstOpenSeat() {
  if (!room.value || !firstOpenSeatNo.value) {
    if (room.value) router.push(`/room/${room.value.id}/players`);
    return;
  }
  openBuyInModal(firstOpenSeatNo.value);
}

async function leaveTable() {
  try {
    await doLeaveRoom();
  } finally {
    router.push('/lobby');
  }
}

async function submitRoomAction(action: string) {
  const amount = amountActions.includes(action) ? actionAmount.value : 0;
  await doRoomAction(action, amount);
}

watch([minActionAmount, maxActionAmount, canChooseActionAmount], () => {
  if (!canChooseActionAmount.value) {
    actionAmount.value = 0;
    return;
  }
  if (actionAmount.value < minActionAmount.value || actionAmount.value > maxActionAmount.value) {
    actionAmount.value = minActionAmount.value;
  }
}, { immediate: true });

watch(() => currentRoomHand.value?.serverTime, (serverTime) => {
  actionClockOffsetMs.value = serverTime ? new Date(serverTime).getTime() - Date.now() : 0;
});

onMounted(async () => {
  countdownTimer = window.setInterval(() => {
    nowMs.value = Date.now();
  }, 1000);
  if (typeof route.params.roomId === 'string' && token.value) {
    if (!room.value || room.value.id !== route.params.roomId) {
      room.value = emptyRoom(route.params.roomId);
    }
    await refreshProfile();
    await refreshRoom();
    await refreshCurrentRoomHand();
    await connectRoomSocket(room.value.id);
  }
});

onBeforeUnmount(() => {
  if (countdownTimer !== null) {
    window.clearInterval(countdownTimer);
    countdownTimer = null;
  }
});
</script>

<template>
  <main class="page-shell room-shell room-shell-minimal room-shell-fullscreen room-mobile-screen">
    <section class="room-stage panel mock-table-felt" v-if="room">
      <div class="responsible-gaming">绿色竞技 远离赌博 · 谨防诈骗 健康生活</div>
      <div class="latency-badge">网络延时<br><strong>{{ tableLatencyMs }}ms</strong></div>
      <header class="table-topbar minimal-topbar table-topbar-compact">
        <button type="button" class="topbar-float table-back-button" @click="router.back()">返回</button>
        <button type="button" class="topbar-float" data-testid="table-leave-button" @click="leaveTable">离开牌桌</button>
        <button type="button" class="topbar-float" @click="router.push(`/room/${room.id}/players`)">
          玩家 {{ occupiedSeatCount }}/{{ room.seatCount || room.members.length }}
        </button>
        <button type="button" class="topbar-float table-invite-code" @click="router.push(`/room/${room.id}/info`)">
          邀请码 {{ room.inviteCode || '加载中' }}
        </button>
        <button type="button" class="topbar-float" @click="router.push(`/room/${room.id}/info`)">房间</button>
        <button type="button" class="topbar-float" data-testid="table-lobby-button" @click="router.push('/lobby')">返回大厅</button>
      </header>

      <div class="table-area bare-table ultra-clean-table casino-table">
        <div class="owner-action-row owner-action-row-dock" v-if="!currentRoomHand">
          <button type="button" :disabled="!isRoomOwner">解散比赛</button>
          <button type="button" @click="inviteFriendFromTable">邀请好友</button>
          <button type="button" :disabled="!isRoomOwner || loading" @click="startHandFromTable">开 始</button>
        </div>
        <p v-if="tableNotice || error" class="table-feedback" :class="{ 'table-feedback-error': !!error }">{{ error || tableNotice }}</p>
        <section class="table-room-center table-room-watermark">
          <span>训练赛◆{{ roomVariantLabel() }} 第 {{ recentRoomHands[0]?.handNo ? recentRoomHands[0].handNo + 1 : 1 }} 手</span>
          <strong>{{ room.name || '德扑之星' }}</strong>
          <span>◎ {{ room.ante ?? 20 }}　级别:{{ room.level ?? 1 }}</span>
          <span>&lt; {{ roomDisplayNo() }} &gt;</span>
          <span>邀请码:{{ room.inviteCode || '加载中' }}</span>
        </section>
        <div class="minimal-strip subtle-strip" v-if="currentRoomHand">
          <span>{{ currentRoomHand.status }}</span>
          <span>底池 {{ currentRoomHand.pot }}</span>
          <span v-if="myRoomSeat">座位 #{{ myRoomSeat.seatNo }}</span>
          <span v-if="myRoomHandPlayer">{{ statusLabel(myRoomHandPlayer.status) }}</span>
          <span v-if="isMyTurn" class="turn-indicator">轮到我</span>
          <span v-if="remainingActionSeconds !== null">行动倒计时 {{ remainingActionSeconds }}s</span>
        </div>

        <div class="board-zone clean-board-zone table-center-stack table-center-stack-compact">
          <div class="community-cards">
            <span v-for="card in displayCards(currentRoomHand?.boardCards)" :key="card" class="board-card table-card-face" :class="{ red: isRedCard(card) }" :aria-label="card">
              <span class="card-rank">{{ cardRankLabel(card) }}</span>
              <span class="card-suit">{{ cardSuitSymbol(card) }}</span>
            </span>
            <span v-if="!displayCards(currentRoomHand?.boardCards).length" class="muted">等待公共牌</span>
          </div>
          <div class="pot-chip table-pot-chip">底池 {{ currentRoomHand?.pot ?? 0 }}</div>
        </div>

        <div class="seat-ring seat-ring-clean seat-ring-casino" v-if="room.seatCount">
          <button
            v-for="seatNo in room.seatCount"
            :key="seatNo"
            type="button"
            class="seat-node seat-node-clean casino-seat-node"
            :data-testid="`table-seat-${seatNo}`"
            :aria-label="roomSeat(seatNo)?.userId ? `座位 ${seatNo} ${roomSeat(seatNo)?.nickname}` : `坐下 座位 ${seatNo}`"
					:class="{ mine: myRoomSeat?.seatNo === seatNo, acting: currentRoomHand?.currentSeat === seatNo, empty: !roomSeat(seatNo)?.userId }"
					@click="openBuyInModal(seatNo)">
					<span class="seat-index">#{{ seatNo }}</span>
					<span class="seat-name">{{ roomSeat(seatNo)?.nickname || (myRoomSeat ? '坐下' : '空座') }}</span>
          <span v-if="roomSeat(seatNo)?.userId" class="seat-presence">{{ presenceForUser(roomSeat(seatNo)?.userId)?.status === 'online' ? '在线' : '离线' }}</span>
					<span class="seat-stack" v-if="seatPlayer(seatNo)">{{ seatPlayer(seatNo)?.stack }}</span>
				</button>
			</div>

			<div class="hero-panel hero-panel-clean" v-if="myRoomSeat">
				<div class="hero-cards">
					<span v-for="card in displayCards(myRoomHandPlayer?.holeCards)" :key="card" class="hole-card table-card-face hero-card-face" :class="{ red: isRedCard(card) }" :aria-label="card">
            <span class="card-rank">{{ cardRankLabel(card) }}</span>
            <span class="card-suit">{{ cardSuitSymbol(card) }}</span>
					</span>
					<span v-if="!displayCards(myRoomHandPlayer?.holeCards).length" class="hole-card back-card table-card-back" aria-label="card back">
            <span class="card-back-grid"></span>
					</span>
				</div>
				<div class="hero-meta">
					<span>{{ me?.nickname }} · 座位 #{{ myRoomSeat.seatNo }}</span>
					<span v-if="myRoomHandPlayer">剩余 {{ myRoomHandPlayer.stack }}</span>
					<span v-else>等待开局</span>
				</div>
			</div>

        <div class="hero-panel waiting-panel" v-else>
          <div class="hero-meta hero-meta-quiet spectator-cta">
            <span>当前为旁观视角</span>
            <button type="button" class="seat-cta-button" :disabled="loading || !firstOpenSeatNo" @click="sitAtFirstOpenSeat">
              {{ firstOpenSeatNo ? `坐下 #${firstOpenSeatNo}` : '已满员' }}
            </button>
            <button type="button" class="ghost seat-secondary-button" @click="router.push(`/room/${room.id}/players`)">选座位</button>
          </div>
        </div>

        <div class="actions hero-actions hero-actions-dock hero-actions-safe" v-if="currentRoomHand">
          <div class="action-amount-control" v-if="canChooseActionAmount">
            <label>下注金额 <strong>{{ actionAmount }}</strong></label>
            <input v-model.number="actionAmount" type="range" :min="minActionAmount" :max="maxActionAmount" :step="50" />
            <input v-model.number="actionAmount" type="number" :min="minActionAmount" :max="maxActionAmount" />
          </div>
          <button v-for="action in currentRoomHand.availableActions" :key="action" type="button" :disabled="loading || !isMyTurn" @click="submitRoomAction(action)">
            <template v-if="amountActions.includes(action) && canChooseActionAmount">执行 {{ actionLabel(action) }}</template>
            <template v-else>{{ actionLabel(action) }}</template>
          </button>
          <button type="button" class="ghost" :disabled="loading" @click="refreshCurrentRoomHand">刷新</button>
        </div>

      </div>

      <nav class="mock-bottom-toolbar mock-bottom-toolbar-safe" aria-label="底部工具栏">
        <button type="button" class="tool-settings" @click="openPanel('settings')">▦</button>
        <button type="button" @click="leaveTable">↩</button>
        <button type="button" class="tool-score" @click="openPanel('score')">战绩</button>
        <button type="button" class="tool-replay" @click="openPanel('replay')">牌谱</button>
        <button type="button" class="tool-chat" @click="openPanel('chat')">▤</button>
        <button type="button" class="tool-mic" disabled>麦克风关</button>
      </nav>

      <BuyInModal
        :open="buyInModalOpen"
        :min="roomMinBuyIn"
        :max="roomMaxBuyIn"
        :wallet-balance="walletBalance"
        @close="closeBuyInModal"
        @confirm="confirmBuyIn"
        @shop="goToShopFromBuyIn"
      />

      <TableDrawer :open="activePanel === 'chat'" placement="bottom" @close="closePanel">
        <TableChatPanel :messages="chatMessages" :loading="loading" @send="sendRoomChat" />
      </TableDrawer>

      <TableDrawer :open="activePanel === 'score'" placement="left" @close="closePanel">
        <TableScorePanel :leaderboard="roomLeaderboard" :members="room.members" :seats="room.seats" :duration-minutes="room.durationMinutes" />
      </TableDrawer>

      <TableDrawer :open="activePanel === 'replay'" placement="right" @close="closePanel">
        <TableReplayPanel :room-id="room.id" :hands="recentRoomHands" :token="token" />
      </TableDrawer>

      <TableDrawer :open="activePanel === 'settings'" placement="bottom" @close="closePanel">
        <TableSettingsPanel
          @stand="myRoomSeat && doLeaveSeat(myRoomSeat.seatNo)"
          @buy-in="myRoomSeat ? openBuyInModal(myRoomSeat.seatNo) : firstOpenSeatNo && openBuyInModal(firstOpenSeatNo)"
          @leave="leaveTable"
          @short-deck-rules="shortDeckRulesOpen = true"
        />
      </TableDrawer>

      <div v-if="shortDeckRulesOpen" class="modal-backdrop" role="dialog" aria-modal="true" aria-label="短牌规则说明">
        <section class="short-deck-rules-modal">
          <h2>短牌规则说明</h2>
          <p>短牌使用 36 张牌，只保留 6 到 A。</p>
          <p>A 可以作为高牌，也可以组成 A6789 顺子。</p>
          <p>本房间短牌规则为同花大于葫芦，底注为 {{ room.ante ?? 20 }}。</p>
          <p>当前玩法：{{ roomVariantLabel() }} · {{ room.seatCount || 0 }} 人桌</p>
          <button type="button" @click="shortDeckRulesOpen = false">关闭</button>
        </section>
      </div>

      <aside class="table-side-panel v11-table-tools" aria-label="牌桌工具">
        <section>
          <h2>动作日志</h2>
          <ol class="compact-list">
            <li v-for="entry in actionLog.slice(-6)" :key="entry.seq">
              #{{ entry.seq }} · {{ logText(entry) }}
            </li>
            <li v-if="!actionLog.length" class="muted">等待牌局动作</li>
          </ol>
        </section>
        <section>
          <h2>房间战绩榜</h2>
          <ol class="compact-list">
            <li v-for="item in roomLeaderboard.slice(0, 4)" :key="item.userId">
              {{ item.nickname }} · {{ item.netProfit >= 0 ? '+' : '' }}{{ item.netProfit }} · {{ item.handsPlayed }}手
            </li>
            <li v-if="!roomLeaderboard.length" class="muted">暂无结算战绩</li>
          </ol>
        </section>
        <section class="room-history-preview">
          <h2>牌桌历史</h2>
          <ol class="compact-list">
            <li v-for="hand in recentRoomHands.slice(0, 3)" :key="hand.handId">
              #{{ hand.handNo }} · {{ hand.winnerSummary || '未结算' }}
              <button type="button" class="text-link-button" @click="router.push(`/room/${room.id}/hands/${hand.handId}/replay`)">回放</button>
            </li>
            <li v-if="!recentRoomHands.length" class="muted">暂无牌桌历史</li>
          </ol>
        </section>
        <section class="chat-panel">
          <h2>聊天表情</h2>
          <ol class="compact-list chat-list">
            <li v-for="message in chatMessages.slice(-5)" :key="message.id">
              {{ message.nickname }}：{{ message.kind === 'emoji' ? message.emojiCode : message.text }}
            </li>
            <li v-if="!chatMessages.length" class="muted">还没有消息</li>
          </ol>
          <div class="chat-actions">
            <input v-model="chatInput" maxlength="200" placeholder="说点什么" @keyup.enter="sendChat" />
            <button type="button" :disabled="loading || !chatInput.trim()" @click="sendChat">发送</button>
            <button type="button" :disabled="loading" @click="sendEmoji('nice_hand')">Nice</button>
            <button type="button" :disabled="loading" @click="sendEmoji('wow')">Wow</button>
          </div>
        </section>
      </aside>
    </section>
  </main>
</template>
