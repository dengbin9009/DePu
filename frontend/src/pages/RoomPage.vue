<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { actionLabel, statusLabel } from '../displayLabels';
import { cardBackImagePath, cardImagePath } from '../cardAssets';
import { isRedCard } from '../pokerVisuals';
import { emptyRoom, useAppState } from '../composables/useAppState';

const route = useRoute();
const router = useRouter();
const {
  room,
  currentRoomHand,
  myRoomSeat,
  myRoomHandPlayer,
  isMyTurn,
  loading,
  token,
  me,
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
  doRoomAction,
  connectRoomSocket,
  sendRoomChat,
  sendRoomEmoji
} = useAppState();

const chatInput = ref('');
const nowMs = ref(Date.now());
const actionClockOffsetMs = ref(0);
const actionAmount = ref(0);
const amountActions = ['bet', 'raise'];
let countdownTimer: number | null = null;

const occupiedSeatCount = computed(() => room.value?.seats.filter((seat) => seat.userId).length ?? 0);
const defaultBuyIn = computed(() => room.value?.seats.find((seat) => seat.buyInChips)?.buyInChips ?? 1000);
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

function hideBrokenCardImage(event: Event) {
  if (event.target instanceof HTMLImageElement) {
    event.target.style.display = 'none';
  }
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
  await doTakeSeat(firstOpenSeatNo.value, defaultBuyIn.value);
  await refreshRoom();
  await refreshCurrentRoomHand();
}

async function leaveTable() {
  try {
    if (myRoomSeat.value) {
      await doLeaveSeat(myRoomSeat.value.seatNo);
      await refreshRoom();
    }
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
  <main class="page-shell room-shell room-shell-minimal room-shell-fullscreen">
    <section class="room-stage panel" v-if="room">
      <header class="table-topbar minimal-topbar">
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
        <div class="minimal-strip subtle-strip" v-if="currentRoomHand">
          <span>{{ currentRoomHand.status }}</span>
          <span>底池 {{ currentRoomHand.pot }}</span>
          <span v-if="myRoomSeat">座位 #{{ myRoomSeat.seatNo }}</span>
          <span v-if="myRoomHandPlayer">{{ statusLabel(myRoomHandPlayer.status) }}</span>
          <span v-if="isMyTurn" class="turn-indicator">轮到我</span>
          <span v-if="remainingActionSeconds !== null">行动倒计时 {{ remainingActionSeconds }}s</span>
        </div>

        <div class="board-zone clean-board-zone table-center-stack">
          <div class="community-cards">
            <span v-for="card in displayCards(currentRoomHand?.boardCards)" :key="card" class="board-card" :class="{ red: isRedCard(card) }">
              <img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="card" class="playing-card-image" @error="hideBrokenCardImage" />
              <template v-else>{{ card }}</template>
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
					:class="{ mine: myRoomSeat?.seatNo === seatNo, acting: currentRoomHand?.currentSeat === seatNo, empty: !roomSeat(seatNo)?.userId }"
					@click="router.push(`/room/${room.id}/players`)">
					<span class="seat-index">#{{ seatNo }}</span>
					<span class="seat-name">{{ roomSeat(seatNo)?.nickname || '空位' }}</span>
          <span v-if="roomSeat(seatNo)?.userId" class="seat-presence">{{ presenceForUser(roomSeat(seatNo)?.userId)?.status === 'online' ? '在线' : '离线' }}</span>
					<span class="seat-stack" v-if="seatPlayer(seatNo)">{{ seatPlayer(seatNo)?.stack }}</span>
				</button>
			</div>

			<div class="hero-panel hero-panel-clean" v-if="myRoomSeat">
				<div class="hero-cards">
					<span v-for="card in displayCards(myRoomHandPlayer?.holeCards)" :key="card" class="hole-card" :class="{ red: isRedCard(card) }">
						<img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="card" class="playing-card-image hero-card-image" @error="hideBrokenCardImage" />
						<template v-else>{{ card }}</template>
					</span>
					<span v-if="!displayCards(myRoomHandPlayer?.holeCards).length" class="hole-card back-card">
						<img :src="cardBackImagePath" alt="card back" class="playing-card-image hero-card-image" @error="hideBrokenCardImage" />
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

        <div class="actions hero-actions hero-actions-dock" v-if="currentRoomHand">
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
