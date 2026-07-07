<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchRechargeOptions, fetchRuleSets } from '../api/client';
import { bettingTypeLabel } from '../displayLabels';
import { useAppState } from '../composables/useAppState';
import type { RuleSet } from '../types/game';

const router = useRouter();
const { doCreateRoom, doJoinRoom, room, loading, error, rechargeOptions, clearError } = useAppState();
const ruleSets = ref<RuleSet[]>([]);
const selectedRuleSet = ref('long-holdem');
const inviteCode = ref('');
const roomSeatCount = ref(6);
const roomMinPlayers = ref(2);
const roomBuyIn = ref(1000);

onMounted(async () => {
  clearError();
  ruleSets.value = await fetchRuleSets();
  if (!rechargeOptions.value.length) {
    rechargeOptions.value = (await fetchRechargeOptions()).options;
  }
});

async function createRoomNow() {
  const created = await doCreateRoom({ ruleSetId: selectedRuleSet.value, seatCount: roomSeatCount.value, minPlayersToStart: roomMinPlayers.value });
  if (created) router.push(`/room/${created.id}`);
}

async function joinRoomNow() {
  const joined = await doJoinRoom(inviteCode.value);
  if (joined) router.push(`/room/${joined.id}`);
}
</script>

<template>
  <main class="page-shell">
    <section class="panel mobile-panel lobby-grid">
      <article class="panel card-panel">
        <h1>房间大厅</h1>
        <p v-if="error" class="error">{{ error }}</p>
      </article>

      <article class="panel card-panel">
        <h2>创建房间</h2>
        <label>游戏模式
          <select v-model="selectedRuleSet">
            <option v-for="rule in ruleSets" :key="rule.id" :value="rule.id">{{ rule.name }} · {{ rule.bettingStructures.map(bettingTypeLabel).join(' / ') }}</option>
          </select>
        </label>
        <label>房间人数 <input v-model.number="roomSeatCount" min="2" max="10" type="number" /></label>
        <label>最少开局 <input v-model.number="roomMinPlayers" min="2" max="10" type="number" /></label>
        <label>买入 <input v-model.number="roomBuyIn" min="1" type="number" /></label>
        <button type="button" :disabled="loading" @click="createRoomNow">创建房间</button>
      </article>

      <article class="panel card-panel">
        <h2>邀请码加入</h2>
        <label>邀请码 <input v-model="inviteCode" type="text" /></label>
        <button type="button" :disabled="loading" @click="joinRoomNow">加入房间</button>
      </article>

      <article class="panel card-panel" v-if="room">
        <h2>当前房间</h2>
        <p>房间 {{ room.id }}</p>
        <p>邀请码 {{ room.inviteCode }}</p>
        <button type="button" @click="router.push(`/room/${room.id}`)">继续进入</button>
      </article>

      <article class="panel card-panel clickable" @click="router.push('/history')">
        <h2>历史战绩</h2>
        <p>查看个人战绩与房间最近牌局。</p>
      </article>

      <article class="panel card-panel clickable" @click="router.push('/rules-test')">
        <h2>规则测试页</h2>
        <p>进入独立规则引擎测试与回放页面。</p>
      </article>
    </section>
  </main>
</template>
