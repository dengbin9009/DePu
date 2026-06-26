<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAppState } from '../composables/useAppState';
import { walletTransactionLabel } from '../displayLabels';

const { me, wallet, saveNickname, refreshProfile, loading } = useAppState();
const router = useRouter();
const nickname = ref('');

function formatDateTime(value?: string | null) {
  if (!value) return '暂无';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN');
}

onMounted(async () => {
  await refreshProfile();
  nickname.value = me.value?.nickname ?? '';
});

async function submitNickname() {
  await saveNickname(nickname.value);
}
</script>

<template>
  <main class="page-shell">
    <section class="panel mobile-panel profile-card">
      <h1>个人中心</h1>
      <div class="profile-head">
        <div class="avatar-placeholder">我</div>
        <div>
          <strong>{{ me?.nickname || '未登录用户' }}</strong>
          <p>ID：{{ me?.id || '-' }}</p>
          <p>金币 {{ wallet?.balance ?? me?.walletBalance ?? 0 }}</p>
          <p>总手数 {{ me?.handsPlayed ?? 0 }} · 总收益 {{ (me?.totalProfit ?? 0) >= 0 ? '+' : '' }}{{ me?.totalProfit ?? 0 }}</p>
          <p>最近对局 {{ formatDateTime(me?.lastPlayedAt) }}</p>
        </div>
      </div>
      <label>修改昵称 <input v-model="nickname" type="text" /></label>
      <button type="button" :disabled="loading" @click="submitNickname">保存昵称</button>
    </section>

    <section class="panel mobile-panel">
      <h2>功能入口</h2>
      <div class="menu-list">
        <button type="button" @click="router.push('/history')">历史战绩</button>
        <button type="button" @click="router.push('/rules-test')">规则测试页</button>
        <button type="button" disabled>商城</button>
        <button type="button" disabled>背包</button>
        <button type="button" disabled>客服</button>
        <button type="button" disabled>生涯</button>
        <button type="button" disabled>牌谱收藏</button>
      </div>
    </section>

    <section v-if="wallet?.transactions?.length" class="panel mobile-panel">
      <h2>钱包流水</h2>
      <ol class="history">
        <li v-for="txn in wallet.transactions" :key="txn.id">
          {{ walletTransactionLabel(txn.type) }} · {{ txn.amount >= 0 ? '+' : '' }}{{ txn.amount }} · 余额 {{ txn.balanceAfter }} · {{ formatDateTime(txn.createdAt) }}
        </li>
      </ol>
    </section>
  </main>
</template>
