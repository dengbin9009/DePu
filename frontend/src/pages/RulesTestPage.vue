<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { createGame, fetchHistory, fetchRuleSets, replayTo, setDebugCards, submitAction } from '../api/client';
import { calculateBetAmountBounds, clampBetAmount, presetBetAmount, type BetPreset } from '../bettingControls';
import { cardImagePath, cardBackImagePath } from '../cardAssets';
import { actionLabel, bettingTypeLabel, handClassLabel, potLabel } from '../displayLabels';
import { isRedCard, tableVisualState, visibleOpponentSeats } from '../pokerVisuals';
import type { ActionLog, BettingStructure, GameSnapshot, RuleSet } from '../types/game';

const ruleSets = ref<RuleSet[]>([]);
const selectedRuleSet = ref('long-holdem');
const selectedBetting = ref<'blinds' | 'ante'>('blinds');
const dealMode = ref<'random' | 'debug'>('random');
const smallBlind = ref(50);
const bigBlind = ref(100);
const ante = ref(10);
const buttonBlind = ref(50);
const playerCount = ref(4);
const game = ref<GameSnapshot | null>(null);
const history = ref<ActionLog[]>([]);
const error = ref('');
const loading = ref(false);
const debugHoleCards = ref('1:As Ah\n2:Ks Kh');
const debugBoard = ref('Qs Js Ts 9s 8s');
const replayTransition = ref(false);
const selectedBetAmount = ref(0);
const faceDownBoardCards = [0, 1, 2];
const betAmountBounds = computed(() => calculateBetAmountBounds(game.value));

onMounted(async () => {
  try {
    ruleSets.value = await fetchRuleSets();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
});

async function startGame() {
  await run(async () => {
    const seats = Array.from({ length: playerCount.value }, (_, i) => ({
      seatNo: i + 1,
      name: defaultName(i + 1),
      stack: 1000
    }));
    game.value = await createGame({
      rulesetId: selectedRuleSet.value,
      buttonSeat: 1,
      bettingStructure: currentBettingStructure(),
      dealMode: dealMode.value,
      seats
    });
    history.value = await fetchHistory(game.value.id);
  });
}

async function act(type: string) {
  if (!game.value) return;
  await run(async () => {
    const amount = isAmountAction(type) ? clampBetAmount(game.value, selectedBetAmount.value) : 0;
    game.value = await submitAction(game.value!, type, amount);
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
    replayTransition.value = true;
    game.value = await replayTo(game.value!.id, seq);
    window.setTimeout(() => {
      replayTransition.value = false;
    }, 420);
  });
}

function selectedRuleDescription() {
  return ruleSets.value.find((rule) => rule.id === selectedRuleSet.value)?.description || 'v1 使用小盲/大盲结构。';
}

function currentBettingStructure(): BettingStructure {
  if (selectedBetting.value === 'ante' && selectedRuleSet.value === 'short-deck') {
    return { type: 'ante', ante: ante.value, buttonBlind: buttonBlind.value };
  }
  return { type: 'blinds', smallBlind: smallBlind.value, bigBlind: bigBlind.value };
}

function defaultName(seatNo: number) {
  if (seatNo === 1) return '按钮';
  if (seatNo === 2) return '小盲';
  if (seatNo === 3) return '大盲';
  return `玩家${seatNo}`;
}

function visual() {
  return tableVisualState(game.value, { replayTransition: replayTransition.value });
}

function heroSeat() {
  return game.value?.seats[0] ?? null;
}

function seatHandClass(handClass?: string | null) {
  return handClassLabel(handClass ?? '');
}

function showdownSeatName(seatNo: number) {
  return game.value?.seats.find((seat) => seat.seatNo === seatNo)?.name ?? `座位 ${seatNo}`;
}

function tableSeatPositions() {
  const hero = heroSeat();
  return visibleOpponentSeats(visual().seatPositions, hero?.seatNo);
}

function formatAwards(awards: Record<string, number> | null) {
  if (!awards) return '无';
  return Object.entries(awards)
    .map(([potId, amount]) => `${potLabel(potId)} +${amount}`)
    .join(' · ');
}

function isAmountAction(action: string) {
  return action === 'bet' || action === 'raise';
}

function actionButtonLabel(action: string) {
  if (!isAmountAction(action)) return actionLabel(action);
  return `${actionLabel(action)} ${selectedBetAmount.value}`;
}

function setBetPreset(preset: BetPreset) {
  selectedBetAmount.value = presetBetAmount(game.value, preset);
}

function normalizeBetAmount() {
  selectedBetAmount.value = clampBetAmount(game.value, selectedBetAmount.value);
}

function hasCardImage(card: string) {
  return Boolean(cardImagePath(card));
}

function shouldShowFaceDownBoard() {
  return Boolean(game.value && !game.value.board?.length);
}

function hideBrokenCardImage(event: Event) {
  if (event.target instanceof HTMLImageElement) {
    event.target.style.display = 'none';
  }
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

watch(
  betAmountBounds,
  (bounds) => {
    selectedBetAmount.value = bounds?.defaultAmount ?? 0;
  },
  { immediate: true }
);
</script>

<template>
  <main class="page-shell rules-test-shell" :class="{ 'has-game': Boolean(game) }">
    <section class="toolbar panel mobile-panel">
      <div>
        <p class="eyebrow">规则测试页</p>
        <h1>德州规则引擎测试 / 回放</h1>
      </div>
      <div class="toolbar-actions">
        <select v-model="selectedRuleSet">
          <option v-for="rule in ruleSets" :key="rule.id" :value="rule.id">{{ rule.name }}</option>
        </select>
        <button type="button" :disabled="loading" @click="startGame">新建测试牌局</button>
      </div>
    </section>

    <section class="setup-strip panel mobile-panel">
      <label>人数 <input v-model.number="playerCount" min="2" max="10" type="number" /></label>
      <label>
        发牌
        <select v-model="dealMode">
          <option value="random">随机</option>
          <option value="debug">调试</option>
        </select>
      </label>
      <label>
        下注结构
        <select v-model="selectedBetting">
          <option value="blinds">{{ bettingTypeLabel('blinds') }}</option>
          <option value="ante" :disabled="selectedRuleSet !== 'short-deck'">{{ bettingTypeLabel('ante') }}</option>
        </select>
      </label>
      <template v-if="selectedBetting === 'ante' && selectedRuleSet === 'short-deck'">
        <label>前注 <input v-model.number="ante" min="1" type="number" /></label>
        <label>按钮盲注 <input v-model.number="buttonBlind" min="1" type="number" /></label>
      </template>
      <template v-else>
        <label>小盲 <input v-model.number="smallBlind" min="1" type="number" /></label>
        <label>大盲 <input v-model.number="bigBlind" min="1" type="number" /></label>
      </template>
    </section>

    <p class="rule-note">{{ selectedRuleDescription() }}</p>
    <p v-if="error" class="error-banner">{{ error }}</p>

    <section v-if="game" class="table-zone panel mobile-panel">
      <section class="phone-stage">
        <div class="mobile-table-screen">
          <div class="table-status-bar">
            <span>阶段 {{ game.stage }}</span>
            <span>当前座位 #{{ game.currentSeat ?? '-' }}</span>
            <span>最低加注 {{ game.minRaise }}</span>
          </div>
          <div class="table-felt">
            <section class="community-core">
              <div class="board">
                <div v-if="shouldShowFaceDownBoard()" class="board-cards">
                  <span v-for="idx in faceDownBoardCards" :key="idx" class="card opponent-card">
                    <img :src="cardBackImagePath()" :alt="`公共牌背面 ${idx + 1}`" @error="hideBrokenCardImage" />
                  </span>
                </div>
                <div v-else class="board-cards">
                  <span v-for="card in game.board" :key="card" class="card" :class="{ red: isRedCard(card) }">
                    <img v-if="cardImagePath(card)" :src="cardImagePath(card)!" :alt="card" @error="hideBrokenCardImage" />
                    <template v-else>{{ card }}</template>
                  </span>
                </div>
              </div>
              <div class="pot-stack">底池 {{ visual().potTotal }}</div>
            </section>

            <div v-for="seat in tableSeatPositions()" :key="seat.seat.seatNo" class="seat" :style="{ left: `${seat.x}%`, top: `${seat.y}%` }">
              <div class="seat-cards">
                <span v-for="card in seat.seat.holeCards || []" :key="card" class="card opponent-card" :class="{ red: isRedCard(card) }">
                  <img v-if="hasCardImage(card)" :src="cardImagePath(card)!" :alt="card" @error="hideBrokenCardImage" />
                  <template v-else>{{ card }}</template>
                </span>
              </div>
              <div class="hero-status">{{ seat.seat.name }} · {{ seat.seat.stack }}</div>
              <div class="seat-hand-rank">{{ seatHandClass((seat.seat.currentHand as any)?.handClass) }}</div>
            </div>

            <div class="hero-hand" v-if="heroSeat()">
              <div class="hero-cards">
                <span v-for="card in heroSeat()?.holeCards || []" :key="card" class="card" :class="{ red: isRedCard(card) }">
                  <img v-if="hasCardImage(card)" :src="cardImagePath(card)!" :alt="card" @error="hideBrokenCardImage" />
                  <template v-else>{{ card }}</template>
                </span>
              </div>
              <div class="hero-status">{{ heroSeat()?.name }} · 记分 {{ heroSeat()?.stack }}</div>
              <div class="hero-rank">{{ seatHandClass(heroSeat()?.currentHand?.handClass) }}</div>
            </div>
          </div>
        </div>
      </section>

      <section class="actions hero-actions">
        <button v-for="action in game.legalActions" :key="action" type="button" :disabled="loading" @click="act(action)">{{ actionButtonLabel(action) }}</button>
      </section>

      <section v-if="betAmountBounds" class="bet-amount-panel">
        <input class="bet-slider" v-model.number="selectedBetAmount" type="range" :min="betAmountBounds.min" :max="betAmountBounds.max" :step="betAmountBounds.step" @change="normalizeBetAmount" />
        <div class="bet-presets">
          <button type="button" @click="setBetPreset('min')">最小</button>
          <button type="button" @click="setBetPreset('half_pot')">1/2 池</button>
          <button type="button" @click="setBetPreset('pot')">底池</button>
          <button type="button" @click="setBetPreset('all_in')">All-in</button>
        </div>
      </section>
    </section>

    <section class="panel mobile-panel" v-if="dealMode === 'debug'">
      <h2>调试发牌</h2>
      <textarea v-model="debugHoleCards" rows="4"></textarea>
      <textarea v-model="debugBoard" rows="2"></textarea>
      <button type="button" :disabled="!game || loading" @click="applyDebugCards">应用调试牌</button>
    </section>

    <section class="layout mobile-layout">
      <section class="panel mobile-panel">
        <h2>历史</h2>
        <ol class="history">
          <li v-for="item in history" :key="item.seq">
            #{{ item.seq }} · {{ actionLabel(item.type) }}
            <span v-if="item.amount"> {{ item.amount }}</span>
            <button type="button" :disabled="loading" @click="replay(item.seq)">回放到这里</button>
          </li>
        </ol>
      </section>

      <section class="panel mobile-panel" v-if="game?.showdown?.length">
        <h2>摊牌</h2>
        <ol class="history">
          <li v-for="seat in game.showdown" :key="seat.seatNo">
            {{ showdownSeatName(seat.seatNo) }} · {{ seatHandClass(seat.handClass) }} · {{ formatAwards(seat.potAwards || null) }}
          </li>
        </ol>
      </section>
    </section>
  </main>
</template>
