<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { createGame, fetchHistory, fetchRuleSets, replayTo, setDebugCards, submitAction } from './api/client';
import { calculateBetAmountBounds, clampBetAmount, presetBetAmount, type BetPreset } from './bettingControls';
import { cardAltText, cardBackAltText, cardBackImagePath, cardImagePath } from './cardAssets';
import {
  actionLabel,
  bettingStructureLabel,
  bettingTypeLabel,
  handClassLabel,
  potLabel,
  stageLabel,
  statusLabel
} from './displayLabels';
import { isRedCard, tableVisualState, visibleOpponentSeats } from './pokerVisuals';
import type { ActionLog, BettingStructure, GameSnapshot, RuleSet } from './types/game';

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
  <main class="app-shell" :class="{ 'has-game': Boolean(game) }">
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
    <section class="setup-strip">
      <label>
        人数
        <input v-model.number="playerCount" min="2" max="10" type="number" />
      </label>
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
    <section class="rules-strip">
      <span v-for="rule in ruleSets" :key="rule.id">
        {{ rule.name }} · {{ rule.ranks[0] }} 到 A · {{ rule.bettingStructures.map(bettingTypeLabel).join(' / ') }}
      </span>
    </section>

    <p v-if="error" class="error">{{ error }}</p>

    <section class="layout">
      <section class="table-zone" :class="{ replaying: visual().replayTransition, compact: visual().seatPositions.length >= 6 }">
        <div class="phone-stage">
          <div class="mobile-table-screen">
            <div class="table-status-bar">
              <strong>{{ game ? stageLabel(game.stage) : '准备开局' }}</strong>
              <span>{{ game ? bettingStructureLabel(game.bettingStructure) : '请选择玩法并新建牌局' }}</span>
            </div>

            <div class="table-felt">
              <article
                v-for="seat in tableSeatPositions()"
                :key="seat.seat.seatNo"
                class="seat"
                :class="{ active: seat.active, compact: seat.compact, folded: seat.seat.status === 'folded', allin: seat.seat.status === 'all_in' }"
                :style="{ left: `${seat.x}%`, top: `${seat.y}%` }"
              >
                <span v-if="seat.dealer" class="dealer-button" title="按钮位">庄</span>
                <strong>{{ seat.seat.name }}</strong>
                <span class="seat-meta">#{{ seat.seat.seatNo }} · {{ statusLabel(seat.seat.status) }}</span>
                <span class="seat-meta">筹码 {{ seat.seat.stack }}</span>
                <span v-if="seat.seat.currentHand" class="seat-hand-rank">{{ handClassLabel(seat.seat.currentHand.handClass) }}</span>
                <div class="seat-cards" aria-label="玩家手牌">
                  <span
                    v-for="card in seat.seat.holeCards || []"
                    :key="`${seat.seat.seatNo}-${card}`"
                    class="card opponent-card"
                    :class="{ red: isRedCard(card), image: hasCardImage(card) }"
                  >
                    <span class="card-fallback">{{ card }}</span>
                    <img v-if="cardImagePath(card)" :src="cardImagePath(card) || ''" :alt="cardAltText(card)" @error="hideBrokenCardImage" />
                  </span>
                </div>
              </article>

              <section class="community-core">
                <div class="board">
                  <span v-for="card in game?.board || []" :key="card" class="card board-card" :class="{ red: isRedCard(card), image: hasCardImage(card) }">
                    <span class="card-fallback">{{ card }}</span>
                    <img v-if="cardImagePath(card)" :src="cardImagePath(card) || ''" :alt="cardAltText(card)" @error="hideBrokenCardImage" />
                  </span>
                  <template v-if="shouldShowFaceDownBoard()">
                    <span v-for="slot in faceDownBoardCards" :key="`back-${slot}`" class="card board-card image face-down">
                      <span class="card-fallback">盖牌</span>
                      <img :src="cardBackImagePath()" :alt="cardBackAltText()" @error="hideBrokenCardImage" />
                    </span>
                  </template>
                  <span v-if="!game" class="muted">等待公共牌</span>
                </div>
                <div class="pot-stack">
                  <span class="chip-stack"></span>
                  <strong>底池 {{ visual().potTotal }}</strong>
                  <small>{{ bettingStructureLabel(game?.bettingStructure) }}</small>
                </div>
              </section>

              <section class="hero-hand" :class="{ active: heroSeat()?.seatNo === game?.currentSeat }">
                <div class="hero-status">
                  <strong>{{ heroSeat()?.name || '我的手牌' }}</strong>
                  <span v-if="heroSeat()">筹码 {{ heroSeat()?.stack }} · 投入 {{ heroSeat()?.streetCommitted }}</span>
                  <span v-if="heroSeat()?.currentHand" class="hero-rank">当前 {{ handClassLabel(heroSeat()?.currentHand?.handClass || '') }}</span>
                  <span v-if="!heroSeat()">新建牌局后显示手牌</span>
                </div>
                <div v-if="betAmountBounds" class="bet-amount-panel">
                  <div class="bet-amount-row">
                    <span>{{ actionLabel(betAmountBounds.action) }}金额</span>
                    <strong>{{ selectedBetAmount }}</strong>
                  </div>
                  <input
                    v-model.number="selectedBetAmount"
                    class="bet-slider"
                    type="range"
                    :min="betAmountBounds.min"
                    :max="betAmountBounds.max"
                    :step="betAmountBounds.step"
                    aria-label="下注金额滑轨"
                    @change="normalizeBetAmount"
                  />
                  <div class="bet-amount-row compact">
                    <span>最小 {{ betAmountBounds.fullMin }}</span>
                    <input
                      v-model.number="selectedBetAmount"
                      class="bet-number"
                      type="number"
                      :min="betAmountBounds.min"
                      :max="betAmountBounds.max"
                      :step="betAmountBounds.step"
                      aria-label="下注金额"
                      @blur="normalizeBetAmount"
                    />
                    <span>最多 {{ betAmountBounds.max }}</span>
                  </div>
                  <div class="bet-presets">
                    <button type="button" :disabled="loading" @click="setBetPreset('min')">最小</button>
                    <button type="button" :disabled="loading" @click="setBetPreset('half_pot')">半池</button>
                    <button type="button" :disabled="loading" @click="setBetPreset('pot')">底池</button>
                    <button type="button" :disabled="loading" @click="setBetPreset('all_in')">全下</button>
                  </div>
                  <small v-if="betAmountBounds.isShortAllIn">筹码不足完整最小加注，将按不足额全下提交。</small>
                </div>
                <div class="actions hero-actions">
                  <button v-for="action in game?.legalActions || []" :key="action" type="button" :disabled="loading" @click="act(action)">
                    {{ actionButtonLabel(action) }}
                  </button>
                </div>
                <div class="cards hero-cards">
                  <span v-for="card in heroSeat()?.holeCards || []" :key="card" class="card hero-card" :class="{ red: isRedCard(card), image: hasCardImage(card) }">
                    <span class="card-fallback">{{ card }}</span>
                    <img v-if="cardImagePath(card)" :src="cardImagePath(card) || ''" :alt="cardAltText(card)" @error="hideBrokenCardImage" />
                  </span>
                  <template v-if="!heroSeat()?.holeCards?.length">
                    <span v-for="slot in 2" :key="`hero-empty-${slot}`" class="card hero-card image face-down">
                      <span class="card-fallback">盖牌</span>
                      <img :src="cardBackImagePath()" :alt="cardBackAltText()" @error="hideBrokenCardImage" />
                    </span>
                  </template>
                </div>
              </section>
            </div>
          </div>
        </div>
      </section>

      <aside class="panel">
        <h2>牌局信息</h2>
        <p v-if="game">阶段 {{ stageLabel(game.stage) }} · 当前座位 {{ game.currentSeat || '-' }} <span v-if="game.isReplay">· 只读回放</span></p>
        <p v-else>先新建牌局，牌桌中央会优先展示公共牌和手牌。</p>

        <h2>底池</h2>
        <ul>
          <li v-for="pot in game?.pots || []" :key="pot.id">{{ potLabel(pot.id) }}: {{ pot.amount }} · 可争夺 {{ pot.eligibleSeats.join(', ') }}</li>
        </ul>

        <h2>摊牌</h2>
        <ul>
          <li v-for="result in game?.showdown || []" :key="result.seatNo">
            座位 {{ result.seatNo }} · {{ handClassLabel(result.handClass) }} · {{ result.bestCards.join(' ') }}
            <span> · 奖励 {{ formatAwards(result.potAwards) }}</span>
          </li>
        </ul>

        <h2>调试发牌</h2>
        <p class="field-hint">手牌格式示例：1:As Ah；公共牌示例：Qs Js Ts。花色暂用 s/h/d/c。</p>
        <textarea v-model="debugHoleCards" rows="3" aria-label="调试手牌"></textarea>
        <input v-model="debugBoard" aria-label="调试公共牌" />
        <button type="button" :disabled="!game || loading || game.debugLocked" @click="applyDebugCards">指定牌</button>

        <h2>历史</h2>
        <button type="button" :disabled="!game || loading" @click="replay(0)">回放初始快照</button>
        <ol class="history">
          <li v-for="item in history" :key="item.seq">
            #{{ item.seq }} {{ stageLabel(item.stage) }} {{ actionLabel(item.type) }} {{ item.amount || '' }}
            <button type="button" @click="replay(item.seq)">回放</button>
          </li>
        </ol>
      </aside>
    </section>
  </main>
</template>
