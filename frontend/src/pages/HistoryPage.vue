<script setup lang="ts">
import { onMounted } from 'vue';
import { useAppState } from '../composables/useAppState';

const { me, room, roomHistory, recentRoomHands, refreshProfile, refreshRoom } = useAppState();

function formatDateTime(value?: string | null) {
  if (!value) return '暂无';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN');
}

onMounted(async () => {
  await refreshProfile();
  if (room.value) await refreshRoom();
});
</script>

<template>
  <main class="page-shell">
    <section class="layout mobile-layout">
      <section class="panel mobile-panel">
        <h1>个人战绩</h1>
        <p v-if="me">总手数 {{ me.handsPlayed }} · 总收益 {{ me.totalProfit >= 0 ? '+' : '' }}{{ me.totalProfit }} · 最近对局 {{ formatDateTime(me.lastPlayedAt) }}</p>
        <ol class="history" v-if="roomHistory.length">
          <li v-for="item in roomHistory" :key="`${item.handId}-${item.nickname}`">
            <strong>#{{ item.handNo }}</strong> · 房间 {{ item.roomId }} · 昵称 {{ item.nickname }} · 收益 {{ item.profit >= 0 ? '+' : '' }}{{ item.profit }} · 赢家 {{ item.winnerSummary || '无' }}
          </li>
        </ol>
      </section>
      <section class="panel mobile-panel">
        <h1>房间最近牌局</h1>
        <p v-if="!recentRoomHands.length">当前没有房间归档信息，请先进入房间。</p>
        <ol class="history" v-else>
          <li v-for="hand in recentRoomHands" :key="hand.handId">
            <strong>#{{ hand.handNo }}</strong> · {{ formatDateTime(hand.completedAt) }} · 赢家 {{ hand.winnerSummary }} · {{ hand.potSummary }}
            <div>公共牌：{{ hand.boardCards.join(' ') }}</div>
          </li>
        </ol>
      </section>
    </section>
  </main>
</template>
