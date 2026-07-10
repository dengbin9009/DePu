<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { fetchRoomHandReplay } from '../api/client';
import { cardAltText, cardImagePath } from '../cardAssets';
import { useAppState } from '../composables/useAppState';
import type { HandReplayResponse } from '../types/game';

const route = useRoute();
const router = useRouter();
const { token, loading, error, run } = useAppState();
const replay = ref<HandReplayResponse | null>(null);
const stepIndex = ref(0);
const playing = ref(false);
let playTimer: number | null = null;

const replayStep = computed(() => replay.value?.steps[stepIndex.value] ?? null);

function displayCards(cards?: string[] | null) {
  return cards?.filter(Boolean) ?? [];
}

function hideBrokenCardImage(event: Event) {
  if (event.target instanceof HTMLImageElement) {
    event.target.style.visibility = 'hidden';
  }
}

function previousStep() {
  stepIndex.value = Math.max(0, stepIndex.value - 1);
}

function nextStep() {
  const maxIndex = Math.max(0, (replay.value?.steps.length ?? 1) - 1);
  stepIndex.value = Math.min(maxIndex, stepIndex.value + 1);
}

function stopPlayback() {
  playing.value = false;
  if (playTimer !== null) {
    window.clearInterval(playTimer);
    playTimer = null;
  }
}

function togglePlayback() {
  if (playing.value) {
    stopPlayback();
    return;
  }
  playing.value = true;
  playTimer = window.setInterval(() => {
    if (!replay.value || stepIndex.value >= replay.value.steps.length - 1) {
      stopPlayback();
      return;
    }
    nextStep();
  }, 900);
}

onMounted(async () => {
  if (!token.value || typeof route.params.roomId !== 'string' || typeof route.params.handId !== 'string') return;
  await run(async () => {
    replay.value = await fetchRoomHandReplay(token.value, route.params.roomId as string, route.params.handId as string);
    stepIndex.value = 0;
  });
});

onBeforeUnmount(stopPlayback);
</script>

<template>
  <main class="page-shell">
    <section class="panel mobile-panel replay-panel">
      <button type="button" class="ghost" @click="router.back()">返回</button>
      <h1>手牌回放</h1>
      <p v-if="error">{{ error }}</p>
      <p v-if="loading">加载中</p>
      <template v-if="replay && replayStep">
        <div class="minimal-strip subtle-strip">
          <span>#{{ replay.handId }}</span>
          <span>步骤 {{ stepIndex + 1 }}/{{ replay.steps.length }}</span>
          <span>{{ replayStep.stage }}</span>
          <span>底池 {{ replayStep.pot }}</span>
          <span v-if="replayStep.currentSeat">行动位 #{{ replayStep.currentSeat }}</span>
        </div>
        <div class="replay-controls">
          <button type="button" :disabled="stepIndex === 0" @click="previousStep">上一步</button>
          <button type="button" :disabled="stepIndex >= replay.steps.length - 1" @click="nextStep">下一步</button>
          <button type="button" @click="togglePlayback">{{ playing ? '暂停' : '播放' }}</button>
        </div>
        <section class="history-card-block">
          <span class="history-card-label">公共牌</span>
          <span class="history-card-row" aria-label="公共牌">
            <span v-for="card in displayCards(replayStep.boardCards)" :key="card" class="history-card-frame">
              <img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="cardAltText(card)" class="history-playing-card" @error="hideBrokenCardImage" />
            </span>
          </span>
        </section>
        <ol class="history">
          <li v-for="player in replayStep.players" :key="player.seatNo">
            #{{ player.seatNo }} {{ player.nickname }} · {{ player.status }} · 剩余 {{ player.stack }} · 投入 {{ player.handCommitted }}
            <span v-if="displayCards(player.holeCards).length" class="history-card-row" aria-label="手牌">
              <span v-for="card in displayCards(player.holeCards)" :key="card" class="history-card-frame">
                <img v-if="cardImagePath(card)" :src="cardImagePath(card) || undefined" :alt="cardAltText(card)" class="history-playing-card" @error="hideBrokenCardImage" />
              </span>
            </span>
          </li>
        </ol>
      </template>
    </section>
  </main>
</template>
