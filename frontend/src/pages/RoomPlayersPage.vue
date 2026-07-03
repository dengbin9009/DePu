<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { emptyRoom, useAppState } from '../composables/useAppState';

const route = useRoute();
const router = useRouter();
const { room, recentRoomHands, roomHistory, token, error, refreshRoom, doTakeSeat, doLeaveSeat, myRoomSeat } = useAppState();

const buyInInput = ref(1000);
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

function memberSeat(userId: string) {
	return room.value?.seats.find((seat) => seat.userId === userId) ?? null;
}

function firstOpenSeatNo() {
	if (!room.value?.seatCount) return null;
	for (let seatNo = 1; seatNo <= room.value.seatCount; seatNo += 1) {
		if (!room.value.seats.find((seat) => seat.seatNo === seatNo && seat.userId)) return seatNo;
	}
	return null;
}

function seatLabel(seatNo: number) {
	const seat = room.value?.seats.find((item) => item.seatNo === seatNo);
	if (!seat?.userId) return `坐下 #${seatNo}`;
	if (seat.userId === myRoomSeat.value?.userId) return `离座 #${seatNo} ${seat.nickname || ''}`;
	return `已占 #${seatNo} ${seat.nickname || ''}`;
}

async function sitCurrentUserAtFirstOpenSeat() {
	const seatNo = firstOpenSeatNo();
	if (!seatNo) return;
	await doTakeSeat(seatNo, buyInInput.value);
	await refreshRoom();
}

async function toggleSeat(seatNo: number) {
	const seat = room.value?.seats.find((item) => item.seatNo === seatNo);
	if (seat?.userId === myRoomSeat.value?.userId) {
		await doLeaveSeat(seatNo);
		await refreshRoom();
		return;
	}
	if (!seat?.userId) {
		await doTakeSeat(seatNo, buyInInput.value);
		await refreshRoom();
	}
}

async function ensureRouteRoom() {
  if (typeof route.params.roomId === 'string' && token.value) {
    if (!room.value || room.value.id !== route.params.roomId) {
      room.value = emptyRoom(route.params.roomId);
    }
    buyInInput.value = myRoomSeat.value?.buyInChips ?? 1000;
    await refreshRoom();
  }
}

onMounted(async () => {
  await ensureRouteRoom();
});
</script>

<template>
  <main class="page-shell room-subpage" v-if="room">
    <section class="panel mobile-panel player-panel-page stack-gap">
      <div class="room-side-header score-panel-header">
        <div>
          <div class="side-eyebrow">当前战绩</div>
          <h1>牌桌玩家</h1>
        </div>
        <button type="button" class="ghost" @click="router.push(`/room/${room.id}`)">返回牌桌</button>
      </div>

      <div class="side-meta-strip">
        <span>邀请码 {{ room.inviteCode }}</span>
        <span>人数 {{ room.members.length }}/{{ room.seatCount }}</span>
        <span>最近一手 {{ currentRoomSummary?.handNo ?? '-' }}</span>
      </div>
      <p v-if="error" class="error">{{ error }}</p>

      <table class="score-table player-board-table side-score-table live-score-table">
        <thead><tr><th>昵称</th><th>带入</th><th>手数</th><th>战绩</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="member in room.members" :key="member.userId" :class="{ 'score-row-current': member.userId === myRoomSeat?.userId }">
            <td><span v-if="member.userId === myRoomSeat?.userId" class="current-marker">▶</span>{{ member.nickname }}</td>
            <td>{{ room.seats.find(seat => seat.userId === member.userId)?.buyInChips ?? '-' }}</td>
            <td>{{ memberHands(member.userId) || '-' }}</td>
            <td :class="{ profit: memberProfit(member.userId) >= 0, loss: memberProfit(member.userId) < 0 }">{{ formatProfit(memberProfit(member.userId)) }}</td>
            <td>
              <span v-if="memberSeat(member.userId)">#{{ memberSeat(member.userId)?.seatNo }}</span>
              <button v-else-if="member.userId === room.ownerUserId && !myRoomSeat" type="button" class="inline-seat-button" @click="sitCurrentUserAtFirstOpenSeat">房主坐下</button>
              <span v-else class="muted">旁观</span>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="spectator-section side-spectator-section">
        <h2>观众（{{ spectatorMembers.length }}）</h2>
        <div class="spectator-grid spectator-grid-avatar audience-grid">
          <div v-for="member in spectatorMembers" :key="member.userId" class="spectator-card spectator-card-avatar">
            <div class="spectator-avatar large-avatar">{{ member.nickname.slice(0, 1) }}</div>
            <div class="spectator-name">{{ member.nickname }}</div>
          </div>
          <div v-if="!spectatorMembers.length" class="spectator-card spectator-card-empty">暂无观众</div>
        </div>
      </div>

      <div class="seat-control-panel side-seat-panel">
        <div class="seat-control-head side-seat-head">
          <div>
            <h2>旁观 / 坐下</h2>
            <p class="seat-help-text">{{ myRoomSeat ? `你已在 #${myRoomSeat.seatNo} 座` : '选择一个空位即可上桌' }}</p>
          </div>
          <label>买入 <input v-model.number="buyInInput" min="1" type="number" /></label>
        </div>
        <div class="seat-grid seat-grid-players seat-grid-players-panel side-seat-grid">
          <button
            v-for="seatNo in room.seatCount"
            :key="seatNo"
            type="button"
            class="seat-button seat-button-panel side-seat-button"
						:class="{ occupied: !!room.seats.find(seat => seat.seatNo === seatNo)?.userId, mine: room.seats.find(seat => seat.seatNo === seatNo)?.userId === myRoomSeat?.userId }"
						@click="toggleSeat(seatNo)"
					>
            {{ seatLabel(seatNo) }}
          </button>
        </div>
      </div>
    </section>
  </main>
</template>
