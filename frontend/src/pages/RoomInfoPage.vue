<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { emptyRoom, useAppState } from '../composables/useAppState';

const route = useRoute();
const router = useRouter();
const { room, myRoomSeat, loading, error, token, refreshProfile, refreshRoom, refreshCurrentRoomHand, doStartRoomHand, connectRoomSocket } = useAppState();

const ownerNickname = computed(() => {
  if (!room.value) return '';
  return room.value.members.find((member) => member.userId === room.value?.ownerUserId)?.nickname || room.value.ownerUserId;
});

async function ensureRouteRoom() {
  if (typeof route.params.roomId === 'string' && token.value) {
    if (!room.value || room.value.id !== route.params.roomId) {
      room.value = emptyRoom(route.params.roomId);
    }
    await refreshProfile();
    await refreshRoom();
    await connectRoomSocket(room.value.id);
  }
}

async function startRoomFromInfo() {
  if (!room.value) return;
  await doStartRoomHand();
  await refreshCurrentRoomHand();
  router.push(`/room/${room.value.id}`);
}

onMounted(async () => {
  await ensureRouteRoom();
});
</script>

<template>
  <main class="page-shell room-subpage" v-if="room">
    <section class="panel mobile-panel card-panel stack-gap">
      <div>
        <h1>房间信息</h1>
        <p>房间 {{ room.id }}</p>
        <p>邀请码 {{ room.inviteCode }}</p>
        <p>状态 {{ room.status }}</p>
        <p>房主 {{ ownerNickname }}</p>
        <p v-if="myRoomSeat">我的座位 #{{ myRoomSeat.seatNo }} · 买入 {{ myRoomSeat.buyInChips }}</p>
        <p v-else>当前为旁观状态</p>
      </div>

      <div class="action-grid two-col">
        <button type="button" :disabled="loading" @click="refreshRoom">刷新房间</button>
        <button type="button" :disabled="loading" @click="startRoomFromInfo">房主开局</button>
        <button type="button" class="ghost" @click="router.push(`/room/${room.id}`)">返回牌桌</button>
        <button type="button" class="ghost" @click="router.push(`/room/${room.id}/players`)">查看玩家</button>
      </div>
      <p v-if="error" class="inline-error">{{ error }}</p>
    </section>
  </main>
</template>
