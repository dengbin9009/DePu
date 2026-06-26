<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { actionLabel, handClassLabel, statusLabel } from '../displayLabels';
import { cardBackImagePath, cardImagePath } from '../cardAssets';
import { isRedCard } from '../pokerVisuals';
import { useAppState } from '../composables/useAppState';

const route = useRoute();
const {
  room,
  currentRoomHand,
  myRoomSeat,
  myRoomHandPlayer,
  isMyTurn,
  recentRoomHands,
  roomHistory,
  loading,
  token,
  me,
  refreshProfile,
  refreshCurrentRoomHand,
  refreshRoom,
  doTakeSeat,
  doLeaveSeat,
  doStartRoomHand,
  doRoomAction,
  startRoomPolling,
  stopRoomPolling
} = useAppState();

const roomBuyIn = ref(1000);
const showDrawer = ref(true);

const currentRoomSummary = computed(() => recentRoomHands.value[0] ?? null);
const spectatorMembers = computed(() => {
  if (!room.value) return [];
  return room.value.members.filter((member) => !room.value?.seats.some((seat) => seat.userId === member.userId));
});

function formatProfit(value?: number | null) {
  const amount = value ?? 0;
  return `${amount >= 0 ? '+' : ''}${amount}`;
}

function memberProfit(userId: string) {
  return currentRoomSummary.value?.participants.find((p) => p.userId === userId)?.profit ?? 0;
}

function memberHands(userId: string) {
  return roomHistory.value.filter((item) => item.roomId === room.value?.id && item.nickname && room.value?.members.some((m) => m.userId === userId && m.nickname === item.nickname)).length;
}

function seatPlayer(seatNo: number) {
  return currentRoomHand.value?.players.find((player) => player.seatNo === seatNo) ?? null;
}

function playerHandClass(seatNo: number) {
  return currentRoomSummary.value?.participants.find((participant) => participant.seatNo === seatNo)?.handClass ?? '';
}

function roomOwnerNickname() {
  if (!room.value) return '';
  return room.value.members.find((member) => member.userId === room.value?.ownerUserId)?.nickname || room.value.ownerUserId;
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

onMounted(async () => {
  if (typeof route.params.roomId === 'string' && token.value) {
    if (!room.value || room.value.id !== route.params.roomId) {
      room.value = { id: route.params.roomId, inviteCode: '', ownerUserId: '', status: 'waiting', members: [], seats: [] } as any;
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
  <main class="page-shell room-shell">
    <section class="room-header panel mobile-panel" v-if="room">
      <div>
        <strong>房间 {{ room.id }}</strong>
        <p>邀请码 {{ room.inviteCode }} · 状态 {{ room.status }}</p>
        <p v-if="myRoomSeat">我的座位 #{{ myRoomSeat.seatNo }} · 买入 {{ myRoomSeat.buyInChips }}</p>
        <p v-else>我当前还未入座</p>
      </div>
      <button type="button" class="ghost" @click="showDrawer = !showDrawer">{{ showDrawer ? '收起玩家面板' : '展开玩家面板' }}</button>
    </section>

    <section class="room-content" v-if="room">
      <aside v-if="showDrawer" class="side-drawer panel">
        <h2>当前战绩</h2>
        <table class="score-table">
          <thead><tr><th>昵称</th><th>带入</th><th>手数</th><th>战绩</th></tr></thead>
          <tbody>
            <tr v-for="member in room.members" :key="member.userId">
              <td>{{ member.nickname }}</td>
              <td>{{ room.seats.find(seat => seat.userId === member.userId)?.buyInChips ?? '-' }}</td>
              <td>{{ memberHands(member.userId) || '-' }}</td>
              <td :class="{ profit: memberProfit(member.userId) >= 0, loss: memberProfit(member.userId) < 0 }">{{ formatProfit(memberProfit(member.userId)) }}</td>
            </tr>
          </tbody>
        </table>
        <h3>观众（{{ spectatorMembers.length }}）</h3>
        <div class="spectators">
          <span v-for="member in spectatorMembers" :key="member.userId" class="spectator-chip">{{ member.nickname }}</span>
          <span v-if="!spectatorMembers.length" class="spectator-chip muted">暂无观众</span>
        </div>
      </aside>

      <section class="table-area panel">
        <div class="rules-strip" v-if="currentRoomHand">
          <span>当前手牌 {{ currentRoomHand.handId }} · 状态 {{ currentRoomHand.status }} · 当前座位 {{ currentRoomHand.currentSeat }}</span>
          <span>底池 {{ currentRoomHand.pot }}</span>
          <span v-if="myRoomHandPlayer">我本手状态 {{ statusLabel(myRoomHandPlayer.status) }} · 剩余 {{ myRoomHandPlayer.stack }} · 已投入 {{ myRoomHandPlayer.handCommitted }}</span>
          <span v-if="isMyTurn">现在轮到我操作</span>
          <span v-else-if="myRoomSeat">当前不是我的回合</span>
          <span v-else>我当前还未入座</span>
        </div>

        <div class="mobile-table-screen multiplayer-table-screen">
          <div class="table-status-bar">
            <span>房主 {{ roomOwnerNickname() }}</span>
            <span>人数 {{ room.members.length }}/{{ room.seats.length }}</span>
            <span v-if="me">当前用户 {{ me.nickname }}</span>
          </div>

          <div class="table-felt multiplayer-felt">
            <div class="community-core room-community-core">
              <div class="board">
                <div class="board-cards">
                  <template v-if="currentRoomHand?.boardCards?.length">
                    <span v-for="card in currentRoomHand.boardCards" :key="card" class="card" :class="{ red: isRedCard(card) }">
                      <img v-if="cardImagePath(card)" :src="cardImagePath(card)!" :alt="card" @error="hideBrokenCardImage" />
                      <template v-else>{{ card }}</template>
                    </span>
                  </template>
                  <template v-else>
                    <span v-for="index in 5" :key="index" class="card opponent-card placeholder-card">
                      <img :src="cardBackImagePath()" :alt="`公共牌背面 ${index}`" @error="hideBrokenCardImage" />
                    </span>
                  </template>
                </div>
              </div>
              <div class="pot-stack">底池 {{ currentRoomHand?.pot ?? 0 }}</div>
            </div>

            <div class="room-seat-ring">
              <button
                v-for="seat in room.seats"
                :key="seat.seatNo"
                type="button"
                class="seat seat-button"
                :class="{
                  active: currentRoomHand?.currentSeat === seat.seatNo,
                  occupied: Boolean(seat.userId),
                  mine: myRoomSeat?.seatNo === seat.seatNo
                }"
                :disabled="loading"
                @click="seat.userId ? (myRoomSeat?.seatNo === seat.seatNo ? doLeaveSeat(seat.seatNo) : undefined) : doTakeSeat(seat.seatNo, roomBuyIn)"
              >
                <span class="seat-label">#{{ seat.seatNo }}</span>
                <span class="seat-name">{{ seat.nickname || '坐下' }}</span>
                <span class="seat-stack">{{ seat.buyInChips ? `带入 ${seat.buyInChips}` : '旁观 / 坐下 / 已入座' }}</span>
                <span class="seat-hand-rank">{{ handClassLabel(playerHandClass(seat.seatNo)) }}</span>
              </button>
            </div>

            <div class="hero-hand room-hero-zone" v-if="myRoomHandPlayer">
              <div class="hero-cards">
                <span v-for="card in displayCards(myRoomHandPlayer.holeCards)" :key="card" class="card" :class="{ red: isRedCard(card) }">
                  <img v-if="cardImagePath(card)" :src="cardImagePath(card)!" :alt="card" @error="hideBrokenCardImage" />
                  <template v-else>{{ card }}</template>
                </span>
                <span v-if="!displayCards(myRoomHandPlayer.holeCards).length" class="hero-status">等待发牌</span>
              </div>
              <div class="hero-status">{{ me?.nickname }} · 剩余 {{ myRoomHandPlayer.stack }}</div>
              <div class="hero-rank">{{ handClassLabel(playerHandClass(myRoomHandPlayer.seatNo)) }}</div>
            </div>
          </div>
        </div>

        <div class="seat-actions">
          <label>买入 <input v-model.number="roomBuyIn" min="1" type="number" /></label>
          <button type="button" :disabled="loading" @click="refreshRoom">刷新房间</button>
          <button type="button" :disabled="loading" @click="doStartRoomHand">房主开局</button>
          <button type="button" :disabled="loading" @click="refreshCurrentRoomHand">刷新当前手牌</button>
        </div>

        <div class="actions hero-actions" v-if="currentRoomHand">
          <button v-for="action in currentRoomHand.availableActions" :key="action" type="button" :disabled="loading || !isMyTurn" @click="doRoomAction(action)">
            {{ actionLabel(action) }}
          </button>
        </div>
      </section>
    </section>
  </main>
</template>
