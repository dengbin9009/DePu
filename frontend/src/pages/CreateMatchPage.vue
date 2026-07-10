<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAppState } from '../composables/useAppState';
import type { RoomVariant } from '../types/game';

const router = useRouter();
const { doCreateRoom, loading, error, clearError } = useAppState();

const matchName = ref('德扑之星');
const mode = ref<'training' | 'sng'>('training');
const variant = ref<RoomVariant>('short_holdem');
const ante = ref(20);
const minBuyIn = ref(2000);
const maxBuyIn = ref(8000);
const buyInCap = ref(60000);
const durationHours = ref(2);
const seatCount = ref(9);

const anteOptions = [10, 20, 30, 50, 100, 200, 300, 500];
const minBuyInOptions = [2000, 4000, 6000, 8000];
const buyInCapOptions = [0, 16000, 24000, 32000, 40000, 60000];
const durationOptions = [0.5, 1, 1.5, 2, 3, 4, 5, 6];
const seatOptions = [2, 3, 4, 5, 6, 7, 8, 9];

const canCreate = computed(() => mode.value === 'training' && (variant.value === 'short_holdem' || variant.value === 'holdem'));
const buyInCapLabel = computed(() => buyInCap.value === 0 ? '无限制' : `${Math.floor(buyInCap.value / 1000)}K`);

function ruleSetForVariant(next: RoomVariant) {
  if (next === 'short_holdem') return 'short-deck';
  if (next === 'holdem') return 'long-holdem';
  return 'omaha';
}

function setVariant(next: RoomVariant) {
  variant.value = next;
  if (next === 'short_holdem') ante.value = 20;
}

async function createNow() {
  clearError();
  if (!canCreate.value) return;
  const created = await doCreateRoom({
    ruleSetId: ruleSetForVariant(variant.value),
    name: matchName.value.trim() || '德扑之星',
    mode: mode.value,
    variant: variant.value,
    ante: ante.value,
    minBuyIn: minBuyIn.value,
    maxBuyIn: Math.max(maxBuyIn.value, minBuyIn.value),
    buyInCap: buyInCap.value || 60000,
    durationMinutes: Math.round(durationHours.value * 60),
    seatCount: seatCount.value,
    minPlayersToStart: 2
  });
  if (created) router.push(`/room/${created.id}`);
}
</script>

<template>
  <main class="page-shell create-match-shell">
    <section class="create-match-panel">
      <header class="mobile-titlebar">
        <button type="button" class="icon-text-button" @click="router.back()">返回</button>
        <h1>创建比赛</h1>
      </header>

      <div class="mode-tabs" aria-label="比赛类型">
        <button type="button" :class="{ active: mode === 'training' }" @click="mode = 'training'">训练赛</button>
        <button type="button" :class="{ active: mode === 'sng' }" @click="mode = 'sng'">SNG</button>
      </div>

      <div class="segmented-pill" aria-label="玩法">
        <button type="button" :class="{ active: variant === 'holdem' }" @click="setVariant('holdem')">国际扑克</button>
        <button type="button" :class="{ active: variant === 'short_holdem' }" @click="setVariant('short_holdem')">短牌</button>
        <button type="button" :class="{ active: variant === 'omaha' }" @click="setVariant('omaha')">奥马哈</button>
      </div>

      <label class="form-row">比赛牌局名字:
        <input v-model="matchName" maxlength="24" placeholder="请输入比赛名字" />
      </label>

      <section class="range-card">
        <h2>Ante设置 <strong>{{ ante }}</strong></h2>
        <div class="choice-row">
          <button v-for="item in anteOptions" :key="item" type="button" :class="{ active: ante === item }" @click="ante = item">{{ item }}</button>
        </div>
      </section>

      <section class="range-card">
        <h2>单次最小带入 <strong>{{ minBuyIn.toLocaleString('zh-CN') }}</strong></h2>
        <div class="choice-row">
          <button v-for="item in minBuyInOptions" :key="item" type="button" :class="{ active: minBuyIn === item }" @click="minBuyIn = item">{{ Math.floor(item / 1000) }}K</button>
        </div>
      </section>

      <section class="range-card">
        <h2>带入记分牌上限 <strong>{{ buyInCapLabel }}</strong></h2>
        <div class="choice-row">
          <button v-for="item in buyInCapOptions" :key="item" type="button" :class="{ active: buyInCap === item }" @click="buyInCap = item">{{ item === 0 ? '无限制' : `${Math.floor(item / 1000)}K` }}</button>
        </div>
      </section>

      <section class="range-card">
        <h2>训练时长 <strong>{{ durationHours }}h</strong></h2>
        <div class="choice-row">
          <button v-for="item in durationOptions" :key="item" type="button" :class="{ active: durationHours === item }" @click="durationHours = item">{{ item }}</button>
        </div>
      </section>

      <section class="range-card">
        <h2>单桌最大人数 <strong>{{ seatCount }}人</strong></h2>
        <div class="choice-row">
          <button v-for="item in seatOptions" :key="item" type="button" :class="{ active: seatCount === item }" @click="seatCount = item">{{ item }}</button>
        </div>
      </section>

      <p v-if="mode === 'sng' || variant === 'omaha'" class="error">该模式暂未开放</p>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="button" class="create-submit-button" :disabled="loading || !canCreate" @click="createNow">确 定 创 建</button>
    </section>
  </main>
</template>
