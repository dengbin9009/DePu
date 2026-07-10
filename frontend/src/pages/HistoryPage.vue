<script setup lang="ts">
import { onMounted } from 'vue';
import { cardAltText, cardImagePath } from '../cardAssets';
import { useAppState } from '../composables/useAppState';
import { useRouter } from 'vue-router';

const { me, roomHistory, recentRoomHands, refreshHistoryDetails } = useAppState();
const router = useRouter();

function formatDateTime(value?: string | null) {
  if (!value) return '暂无';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN');
}

function displayCards(cards?: string[] | null) {
  return cards?.filter(Boolean) ?? [];
}

function hideBrokenCardImage(event: Event) {
  if (event.target instanceof HTMLImageElement) {
    event.target.style.visibility = 'hidden';
  }
}

onMounted(async () => {
  await refreshHistoryDetails();
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
            <button type="button" class="ghost" @click="router.push(`/room/${hand.roomId}/hands/${hand.handId}/replay`)">查看回放</button>
            <div class="history-card-block">
              <span class="history-card-label">公共牌</span>
              <span class="history-card-row" aria-label="公共牌">
                <span v-for="card in displayCards(hand.boardCards)" :key="card" class="history-card-frame">
                  <img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="cardAltText(card)" class="history-playing-card" @error="hideBrokenCardImage" />
                </span>
              </span>
            </div>
            <div v-for="participant in hand.participants" :key="`${hand.handId}-${participant.seatNo}`" class="history-participant">
              <span class="history-participant-title">#{{ participant.seatNo }} {{ participant.nickname }} · 投入 {{ participant.handCommitted }} · 返奖 {{ participant.awardAmount }} · 净 {{ participant.profit >= 0 ? '+' : '' }}{{ participant.profit }}</span>
              <span class="history-card-block">
                <span class="history-card-label">手牌</span>
                <span class="history-card-row" aria-label="手牌">
                  <span v-for="card in displayCards(participant.holeCards)" :key="card" class="history-card-frame">
                    <img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="cardAltText(card)" class="history-playing-card" @error="hideBrokenCardImage" />
                  </span>
                </span>
              </span>
              <span class="history-card-block">
                <span class="history-card-label">最佳牌</span>
                <span class="history-card-row" aria-label="最佳牌">
                  <span v-for="card in displayCards(participant.bestCards)" :key="card" class="history-card-frame">
                    <img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="cardAltText(card)" class="history-playing-card" @error="hideBrokenCardImage" />
                  </span>
                </span>
              </span>
            </div>
          </li>
        </ol>
      </section>
    </section>
  </main>
</template>
