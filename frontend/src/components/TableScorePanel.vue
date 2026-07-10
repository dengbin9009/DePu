<script setup lang="ts">
import type { RoomLeaderboardItem, RoomMember, RoomSeat } from '../types/game';

defineProps<{
  leaderboard: RoomLeaderboardItem[];
  members: RoomMember[];
  seats: RoomSeat[];
  durationMinutes?: number;
}>();

function seatBuyIn(userId: string, seats: RoomSeat[]) {
  return seats.find((seat) => seat.userId === userId)?.buyInChips ?? 0;
}
</script>

<template>
  <section class="table-score-panel">
    <header><strong>当前战绩</strong><span>剩余时间 {{ Math.floor((durationMinutes ?? 120) / 60).toString().padStart(2, '0') }}:00:00</span></header>
    <table>
      <thead><tr><th>昵称</th><th>带入</th><th>手数</th><th>战绩</th></tr></thead>
      <tbody>
        <tr v-for="item in leaderboard" :key="item.userId">
          <td>{{ item.nickname }}</td>
          <td>{{ seatBuyIn(item.userId, seats) || '-' }}</td>
          <td>{{ item.handsPlayed }}</td>
          <td>{{ item.netProfit >= 0 ? '+' : '' }}{{ item.netProfit }}</td>
        </tr>
      </tbody>
    </table>
    <p v-if="!leaderboard.length">观众({{ members.filter(member => !seats.some(seat => seat.userId === member.userId)).length }}/{{ members.length }})</p>
  </section>
</template>
