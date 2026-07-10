<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { fetchRoomHandReplay } from '../api/client';
import { actionLabel, statusLabel } from '../displayLabels';
import type { HandReplayResponse, RoomHandHistoryRecord } from '../types/game';

const props = defineProps<{
  roomId: string;
  hands: RoomHandHistoryRecord[];
  token: string;
}>();

const replay = ref<HandReplayResponse | null>(null);
const selectedIndex = ref(0);
const stepIndex = ref(0);
const loading = ref(false);
const replayError = ref('');

const replayStep = computed(() => replay.value?.steps[stepIndex.value] ?? null);

async function loadReplay(index = selectedIndex.value) {
  const hand = props.hands[index];
  if (!hand || !props.token) return;
  loading.value = true;
  replayError.value = '';
  try {
    replay.value = await fetchRoomHandReplay(props.token, props.roomId, hand.handId);
    selectedIndex.value = index;
    stepIndex.value = 0;
  } catch (err) {
    replayError.value = err instanceof Error ? err.message : '牌谱加载失败';
  } finally {
    loading.value = false;
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
  if (hands.length) void loadReplay(0);
}, { immediate: true });
</script>

<template>
  <section class="table-replay-panel">
    <header>
      <h2>牌谱回顾 - {{ hands[selectedIndex]?.handNo ?? '-' }}</h2>
      <button type="button" :disabled="!hands.length || loading" @click="loadReplay()">回放</button>
    </header>
    <p v-if="!hands.length">暂无任何数据</p>
    <p v-if="replayError" class="error">{{ replayError }}</p>
    <ol v-else class="replay-hand-list">
      <li v-for="(hand, index) in hands" :key="hand.handId">
        <button type="button" :class="{ active: selectedIndex === index }" @click="loadReplay(index)">
          第{{ hand.handNo }}手 · {{ hand.winnerSummary || '未结算' }}
        </button>
        <span>{{ hand.potSummary }} · 底池 {{ hand.totalPot }}</span>
        <span v-if="displayCards(hand.boardCards).length">公共牌 {{ displayCards(hand.boardCards).join(' ') }}</span>
        <button type="button" @click="loadReplay(index)">查看</button>
      </li>
    </ol>
    <section v-if="replay && replayStep" class="replay-step-detail">
      <div class="replay-step-head">
        <span>步骤 {{ stepIndex + 1 }}/{{ replay.steps.length }}</span>
        <strong>{{ replayStep.stage }}</strong>
        <span>底池 {{ replayStep.pot }}</span>
      </div>
      <p>上一动作：{{ stepActionText(replayStep) }}</p>
      <p>公共牌：{{ displayCards(replayStep.boardCards).join(' ') || '暂无' }}</p>
      <h3>玩家明细</h3>
      <ol>
        <li v-for="player in replayStep.players" :key="player.seatNo">
          #{{ player.seatNo }} {{ player.nickname }} · {{ statusLabel(player.status) }} · 剩余 {{ player.stack }} · 已投 {{ player.handCommitted }}
          <span v-if="displayCards(player.holeCards).length"> · 手牌 {{ displayCards(player.holeCards).join(' ') }}</span>
        </li>
      </ol>
      <div class="replay-step-actions">
        <button type="button" :disabled="stepIndex <= 0" @click="previousStep">上一动作</button>
        <button type="button" :disabled="stepIndex >= replay.steps.length - 1" @click="nextStep">下一动作</button>
      </div>
    </section>
    <footer class="replay-panel-footer">
      <button type="button" disabled>收藏</button>
      <button type="button" :disabled="selectedIndex <= 0" @click="loadReplay(selectedIndex - 1)">上一手</button>
      <span>第{{ hands[selectedIndex]?.handNo ?? 1 }}手</span>
      <button type="button" :disabled="selectedIndex >= hands.length - 1" @click="loadReplay(selectedIndex + 1)">下一手</button>
      <button type="button" disabled>投诉</button>
    </footer>
  </section>
</template>
