<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { fetchRoomHandReplay, fetchRoomHands } from '../api/client';
import { actionLabel, stageLabel, statusLabel } from '../displayLabels';
import { cardFaceVisual } from '../pokerVisuals';
import type { HandReplayResponse, RoomHandHistoryRecord } from '../types/game';

const props = defineProps<{
  roomId: string;
  hands: RoomHandHistoryRecord[];
  token: string;
}>();

const replay = ref<HandReplayResponse | null>(null);
const historyHands = ref<RoomHandHistoryRecord[]>([...props.hands]);
const selectedIndex = ref(0);
const stepIndex = ref(0);
const historyLoading = ref(false);
const replayLoading = ref(false);
const historyError = ref('');
const replayError = ref('');
let replayRequestId = 0;

const replayStep = computed(() => replay.value?.steps[stepIndex.value] ?? null);

async function loadReplay(index = selectedIndex.value, hands = historyHands.value) {
  const hand = hands[index];
  if (!hand || !props.token) return;
  const requestId = ++replayRequestId;
  selectedIndex.value = index;
  stepIndex.value = 0;
  replay.value = null;
  replayLoading.value = true;
  replayError.value = '';
  try {
    const response = await fetchRoomHandReplay(props.token, props.roomId, hand.handId);
    if (requestId !== replayRequestId) return;
    replay.value = response;
  } catch (err) {
    if (requestId !== replayRequestId) return;
    replayError.value = err instanceof Error ? `牌谱回放加载失败：${err.message}` : '牌谱回放加载失败';
  } finally {
    if (requestId === replayRequestId) replayLoading.value = false;
  }
}

async function refreshHistory() {
  if (!props.token) return;
  historyLoading.value = true;
  historyError.value = '';
  try {
    const items = (await fetchRoomHands(props.token, props.roomId)).items;
    historyHands.value = items;
    selectedIndex.value = 0;
    stepIndex.value = 0;
    replay.value = null;
    replayError.value = '';
    if (items.length) await loadReplay(0, items);
  } catch (err) {
    historyError.value = err instanceof Error ? `牌谱历史加载失败：${err.message}` : '牌谱历史加载失败';
  } finally {
    historyLoading.value = false;
  }
}

function displayCards(cards?: string[] | null) {
  return cards?.filter(Boolean) ?? [];
}

function stepActionText(step = replayStep.value) {
  if (!step?.action) return step?.seq === 0 ? '手牌开始' : '等待动作';
  return `#${step.action.seatNo} ${actionLabel(step.action.type)}${step.action.amount ? ` ${step.action.amount}` : ''}`;
}

function previousStep() {
  stepIndex.value = Math.max(0, stepIndex.value - 1);
}

function nextStep() {
  stepIndex.value = Math.min(Math.max(0, (replay.value?.steps.length ?? 1) - 1), stepIndex.value + 1);
}

watch(() => props.hands, (hands) => {
  historyHands.value = [...hands];
});

onMounted(() => {
  void refreshHistory();
});
</script>

<template>
  <section class="table-replay-panel">
    <header>
      <h2>牌谱回顾 - {{ historyHands[selectedIndex]?.handNo ?? '-' }}</h2>
      <button type="button" :disabled="!historyHands.length || replayLoading" @click="loadReplay()">回放</button>
    </header>
    <p v-if="historyLoading && !historyHands.length">正在加载牌谱历史...</p>
    <div v-else-if="historyError" class="replay-load-state">
      <p class="error">{{ historyError }}</p>
      <button type="button" :disabled="historyLoading" @click="refreshHistory">重试加载历史</button>
    </div>
    <p v-else-if="!historyHands.length">暂无已结算牌谱</p>
    <div v-if="replayError" class="replay-load-state">
      <p class="error">{{ replayError }}</p>
      <button type="button" :disabled="replayLoading" @click="loadReplay()">重试回放</button>
    </div>
    <ol v-if="!historyError && historyHands.length" class="replay-hand-list">
      <li v-for="(hand, index) in historyHands" :key="hand.handId">
        <button type="button" :class="{ active: selectedIndex === index }" @click="loadReplay(index)">
          第{{ hand.handNo }}手 · {{ hand.winnerSummary || '未结算' }}
        </button>
        <span>{{ hand.potSummary }} · 底池 {{ hand.totalPot }}</span>
        <div v-if="displayCards(hand.boardCards).length" class="replay-card-row replay-hand-board" aria-label="公共牌">
          <span class="replay-card-row-label">公共牌</span>
          <span
            v-for="card in displayCards(hand.boardCards)"
            :key="card"
            class="table-card-face replay-card-face"
            :class="cardFaceVisual(card).colorClass"
            :data-card="card"
            :aria-label="cardFaceVisual(card).ariaLabel"
          >
            <span class="card-rank">{{ cardFaceVisual(card).rankLabel }}</span>
            <span class="card-suit">{{ cardFaceVisual(card).suitSymbol }}</span>
          </span>
        </div>
        <button type="button" @click="loadReplay(index)">查看</button>
      </li>
    </ol>
    <section v-if="replay && replayStep" class="replay-step-detail">
      <div class="replay-step-head">
        <span>步骤 #{{ replayStep.seq }} · {{ stepIndex + 1 }}/{{ replay.steps.length }}</span>
        <strong>阶段 {{ stageLabel(replayStep.stage) }}</strong>
        <span>底池 {{ replayStep.pot }}</span>
      </div>
      <p>上一动作：{{ stepActionText(replayStep) }}</p>
      <div class="replay-card-row replay-step-board" aria-label="公共牌">
        <span class="replay-card-row-label">公共牌：</span>
        <template v-if="displayCards(replayStep.boardCards).length">
          <span
            v-for="card in displayCards(replayStep.boardCards)"
            :key="card"
            class="table-card-face replay-card-face"
            :class="cardFaceVisual(card).colorClass"
            :data-card="card"
            :aria-label="cardFaceVisual(card).ariaLabel"
          >
            <span class="card-rank">{{ cardFaceVisual(card).rankLabel }}</span>
            <span class="card-suit">{{ cardFaceVisual(card).suitSymbol }}</span>
          </span>
        </template>
        <span v-else class="muted">暂无</span>
      </div>
      <h3>玩家明细</h3>
      <ol>
        <li v-for="player in replayStep.players" :key="player.seatNo">
          <span>#{{ player.seatNo }} {{ player.nickname }} · {{ statusLabel(player.status) }} · 剩余 {{ player.stack }} · 已投 {{ player.handCommitted }}</span>
          <div v-if="displayCards(player.holeCards).length" class="replay-card-row replay-player-hole-cards" aria-label="手牌">
            <span class="replay-card-row-label">手牌</span>
            <span
              v-for="card in displayCards(player.holeCards)"
              :key="card"
              class="table-card-face replay-card-face"
              :class="cardFaceVisual(card).colorClass"
              :data-card="card"
              :aria-label="cardFaceVisual(card).ariaLabel"
            >
              <span class="card-rank">{{ cardFaceVisual(card).rankLabel }}</span>
              <span class="card-suit">{{ cardFaceVisual(card).suitSymbol }}</span>
            </span>
          </div>
        </li>
      </ol>
      <div class="replay-step-actions">
        <button type="button" :disabled="stepIndex <= 0" @click="previousStep">上一动作</button>
        <button type="button" :disabled="stepIndex >= replay.steps.length - 1" @click="nextStep">下一动作</button>
      </div>
    </section>
    <footer class="replay-panel-footer">
      <button type="button" disabled title="收藏功能暂未开放">收藏（暂未开放）</button>
      <button type="button" :disabled="selectedIndex <= 0" @click="loadReplay(selectedIndex - 1)">上一手</button>
      <span>第{{ historyHands[selectedIndex]?.handNo ?? 1 }}手</span>
      <button type="button" :disabled="selectedIndex >= historyHands.length - 1" @click="loadReplay(selectedIndex + 1)">下一手</button>
      <button type="button" disabled title="投诉功能暂未开放">投诉（暂未开放）</button>
    </footer>
  </section>
</template>
