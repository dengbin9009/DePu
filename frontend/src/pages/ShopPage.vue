<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchRechargeOptions } from '../api/client';
import { useAppState } from '../composables/useAppState';
import type { RechargeOption } from '../types/game';

const router = useRouter();
const { wallet, me, loading, refreshProfile, doRecharge, consumeShopReturnTo } = useAppState();
const activeTab = ref('金币');
const rechargeOptions = ref<RechargeOption[]>([]);
const message = ref('');
const returnPath = ref('');
const pendingRecharge = ref<RechargeOption | null>(null);
const tabs = ['金币', '钻石', 'VIP卡', '装扮', '道具', '课程'];

const bonusByCode: Record<string, number> = {
  small: 28,
  medium: 218,
  large: 618
};

function priceLabel(option: RechargeOption) {
  if (option.code === 'small') return '¥ 6';
  if (option.code === 'medium') return '¥ 30';
  if (option.code === 'large') return '¥ 68';
  return '模拟充值';
}

function simulateRecharge(option: RechargeOption) {
  pendingRecharge.value = option;
}

async function confirmRecharge() {
  if (!pendingRecharge.value) return;
  const option = pendingRecharge.value;
  await doRecharge(option.code);
  message.value = `充值成功：+${option.amount} 金币`;
  pendingRecharge.value = null;
}

function goBack() {
  router.push(returnPath.value || '/me');
}

onMounted(async () => {
  returnPath.value = consumeShopReturnTo();
  await refreshProfile();
  rechargeOptions.value = (await fetchRechargeOptions()).options;
});
</script>

<template>
  <main class="page-shell shop-shell">
    <section class="shop-panel">
      <header class="mobile-titlebar">
        <button type="button" class="icon-text-button" @click="goBack">返回</button>
        <h1>商城</h1>
      </header>

      <nav class="shop-tabs" aria-label="商城分类">
        <button v-for="tab in tabs" :key="tab" type="button" :class="{ active: activeTab === tab }" @click="activeTab = tab">{{ tab }}</button>
      </nav>

      <section class="asset-strip">
        <div><strong>金币</strong><span>{{ wallet?.balance ?? me?.walletBalance ?? 0 }}</span></div>
        <div><strong>钻石</strong><span>0</span></div>
        <div><strong>积分</strong><span>0</span></div>
        <div><strong>普通卡</strong><span>未激活</span></div>
      </section>

      <section v-if="activeTab === '金币'" class="shop-grid">
        <button v-for="option in rechargeOptions" :key="option.code" type="button" class="shop-product" :disabled="loading" @click="simulateRecharge(option)">
          <span class="product-icon">♠</span>
          <strong>{{ option.amount }}金币</strong>
          <span>额外赠送{{ bonusByCode[option.code] ?? 0 }}金币</span>
          <em>{{ priceLabel(option) }}</em>
        </button>
      </section>

      <section v-else class="shop-placeholder">
        <strong>{{ activeTab }}</strong>
        <span>该分类暂未开放</span>
      </section>

      <p v-if="message" class="success-text">{{ message }}</p>

      <div v-if="pendingRecharge" class="modal-backdrop">
        <section class="shop-confirm-modal" role="dialog" aria-modal="true" aria-label="确认模拟充值">
          <h2>确认模拟充值</h2>
          <p>{{ pendingRecharge.label }}</p>
          <strong>+{{ pendingRecharge.amount }} 金币</strong>
          <div class="modal-actions">
            <button type="button" class="ghost" @click="pendingRecharge = null">取消</button>
            <button type="button" :disabled="loading" @click="confirmRecharge">确认</button>
          </div>
        </section>
      </div>
    </section>
  </main>
</template>
