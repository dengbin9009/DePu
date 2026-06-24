<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { createGame, fetchHistory, fetchRuleSets, replayTo, setDebugCards, submitAction } from './api/client';
import type { ActionLog, GameSnapshot, RuleSet } from './types/game';

const ruleSets = ref<RuleSet[]>([]);
const selectedRuleSet = ref('long-holdem');
const game = ref<GameSnapshot | null>(null);
const history = ref<ActionLog[]>([]);
const error = ref('');
const loading = ref(false);
const debugHoleCards = ref('1:As Ah\n2:Ks Kh');
const debugBoard = ref('Qs Js Ts 9s 8s');

onMounted(async () => {
  try {
    ruleSets.value = await fetchRuleSets();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
});

async function startGame() {
  await run(async () => {
    game.value = await createGame(selectedRuleSet.value);
    history.value = await fetchHistory(game.value.id);
  });
}

async function act(type: string) {
  if (!game.value) return;
  await run(async () => {
    game.value = await submitAction(game.value!, type);
    history.value = await fetchHistory(game.value.id);
  });
}

async function applyDebugCards() {
  if (!game.value) return;
  await run(async () => {
    const holeCards: Record<string, string[]> = {};
    for (const line of debugHoleCards.value.split('\n')) {
      const [seat, cardsText] = line.split(':');
      if (!seat || !cardsText) continue;
      holeCards[seat.trim()] = cardsText.trim().split(/\s+/).filter(Boolean);
    }
    const board = debugBoard.value.trim() ? debugBoard.value.trim().split(/\s+/) : [];
    game.value = await setDebugCards(game.value!, holeCards, board);
    history.value = await fetchHistory(game.value.id);
  });
}

async function replay(seq: number) {
  if (!game.value) return;
  await run(async () => {
    game.value = await replayTo(game.value!.id, seq);
  });
}

function selectedRuleDescription() {
  return ruleSets.value.find((rule) => rule.id === selectedRuleSet.value)?.description || 'v1 使用小盲/大盲结构。';
}

async function run(fn: () => Promise<void>) {
  loading.value = true;
  error.value = '';
  try {
    await fn();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <main class="app-shell">
    <section class="toolbar">
      <div>
        <p class="eyebrow">DePu</p>
        <h1>德州扑克牌局模拟器</h1>
      </div>
      <div class="toolbar-actions">
        <select v-model="selectedRuleSet">
          <option v-for="rule in ruleSets" :key="rule.id" :value="rule.id">
            {{ rule.name }}
          </option>
        </select>
        <button type="button" :disabled="loading" @click="startGame">新建牌局</button>
      </div>
    </section>
    <p class="rule-note">{{ selectedRuleDescription() }}</p>

    <p v-if="error" class="error">{{ error }}</p>

    <section class="layout">
      <section class="table-surface">
        <div class="board">
          <span v-for="card in game?.board || []" :key="card" class="card">{{ card }}</span>
          <span v-if="!game?.board?.length" class="muted">等待公共牌</span>
        </div>
        <div class="seats">
          <article v-for="seat in game?.seats || []" :key="seat.seatNo" class="seat" :class="{ active: seat.seatNo === game?.currentSeat }">
            <strong>{{ seat.name }}</strong>
            <span>座位 {{ seat.seatNo }}</span>
            <span>筹码 {{ seat.stack }}</span>
            <span>状态 {{ seat.status }}</span>
            <div class="cards">
              <span v-for="card in seat.holeCards || []" :key="card" class="card small">{{ card }}</span>
            </div>
          </article>
        </div>
      </section>

      <aside class="panel">
        <h2>行动</h2>
        <p v-if="game">阶段 {{ game.stage }} · 当前座位 {{ game.currentSeat || '-' }}</p>
        <div class="actions">
          <button v-for="action in game?.legalActions || []" :key="action" type="button" :disabled="loading" @click="act(action)">
            {{ action }}
          </button>
        </div>

        <h2>底池</h2>
        <ul>
          <li v-for="pot in game?.pots || []" :key="pot.id">{{ pot.id }}: {{ pot.amount }}</li>
        </ul>

        <h2>摊牌</h2>
        <ul>
          <li v-for="result in game?.showdown || []" :key="result.seatNo">
            座位 {{ result.seatNo }} · {{ result.handClass }} · {{ result.bestCards.join(' ') }}
          </li>
        </ul>

        <h2>调试发牌</h2>
        <textarea v-model="debugHoleCards" rows="3" aria-label="调试手牌"></textarea>
        <input v-model="debugBoard" aria-label="调试公共牌" />
        <button type="button" :disabled="!game || loading" @click="applyDebugCards">指定牌</button>

        <h2>历史</h2>
        <button type="button" :disabled="!game || loading" @click="replay(0)">回放当前快照</button>
        <ol class="history">
          <li v-for="item in history" :key="item.seq">
            #{{ item.seq }} {{ item.stage }} {{ item.type }} {{ item.amount || '' }}
            <button type="button" @click="replay(item.seq)">回放</button>
          </li>
        </ol>
      </aside>
    </section>
  </main>
</template>
