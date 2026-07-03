<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue';
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
  refreshProfile,
  refreshCurrentRoomHand,
  refreshRoom,
  doTakeSeat,
  doRoomAction,
  startRoomPolling,
  stopRoomPolling
} = useAppState();

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

function displayCards(cards?: string[] | null) {
  if (!cards?.length) return [];
  return cards;
}

function hideBrokenCardImage(event: Event) {
  if (event.target instanceof HTMLImageElement) {
    event.target.style.display = 'none';
  }
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

onMounted(async () => {
  if (typeof route.params.roomId === 'string' && token.value) {
    if (!room.value || room.value.id !== route.params.roomId) {
      room.value = emptyRoom(route.params.roomId);
    }
    await refreshProfile();
    await refreshRoom();
    await refreshCurrentRoomHand();
    startRoomPolling();
  }
});

onBeforeUnmount(() => {
  stopRoomPolling();
});
</script>

<template>
  <main class="page-shell room-shell room-shell-minimal room-shell-fullscreen">
    <section class="room-stage panel" v-if="room">
      <header class="table-topbar minimal-topbar">
        <button type="button" class="topbar-float table-back-button" @click="router.back()">返回</button>
        <button type="button" class="topbar-float" @click="router.push(`/room/${room.id}/players`)">
          玩家 {{ occupiedSeatCount }}/{{ room.seatCount || room.members.length }}
        </button>
        <button type="button" class="topbar-float table-invite-code" @click="router.push(`/room/${room.id}/info`)">
          邀请码 {{ room.inviteCode || '加载中' }}
        </button>
        <button type="button" class="topbar-float" @click="router.push(`/room/${room.id}/info`)">房间</button>
      </header>

      <div class="table-area bare-table ultra-clean-table casino-table">
        <div class="minimal-strip subtle-strip" v-if="currentRoomHand">
          <span>{{ currentRoomHand.status }}</span>
          <span>底池 {{ currentRoomHand.pot }}</span>
          <span v-if="myRoomSeat">座位 #{{ myRoomSeat.seatNo }}</span>
          <span v-if="myRoomHandPlayer">{{ statusLabel(myRoomHandPlayer.status) }}</span>
          <span v-if="isMyTurn" class="turn-indicator">轮到我</span>
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
          <button v-for="action in currentRoomHand.availableActions" :key="action" type="button" :disabled="loading || !isMyTurn" @click="doRoomAction(action)">
            {{ actionLabel(action) }}
          </button>
          <button type="button" class="ghost" :disabled="loading" @click="refreshCurrentRoomHand">刷新</button>
        </div>
      </div>
    </section>
  </main>
</template>
