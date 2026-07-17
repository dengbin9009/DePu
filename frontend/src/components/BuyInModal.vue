<script setup lang="ts">
import { computed, ref, watch } from 'vue';

const props = defineProps<{
  open: boolean;
  min: number;
  max: number;
  walletBalance: number;
}>();

const emit = defineEmits<{
  close: [];
  confirm: [amount: number];
  shop: [];
}>();

const amount = ref(props.min);
const options = computed(() => {
  const min = props.min || 2000;
  const max = props.max || 6000;
  const values = [min, min + 1000, min + 2000, min + 3000, max]
    .filter((value, index, array) => value <= max && array.indexOf(value) === index);
  return values.length ? values : [2000, 3000, 4000, 5000, 6000];
});
const insufficient = computed(() => amount.value > props.walletBalance);

watch(() => props.open, (open) => {
  if (open) amount.value = props.min || 2000;
});
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
    <section class="buy-in-modal" role="dialog" aria-modal="true" aria-label="补充记分牌">
      <h2>补充记分牌</h2>
      <p>在下一手开始前，为您补充记分牌</p>
      <strong class="buy-in-amount">{{ amount.toLocaleString('zh-CN') }}</strong>
      <span class="buy-in-label">带入记分牌</span>

      <div class="buy-in-options">
        <button v-for="option in options" :key="option" type="button" :class="{ active: amount === option }" @click="amount = option">
          {{ Math.floor(option / 1000) }}K
        </button>
      </div>

      <div class="buy-in-balance">
        <span>消耗 <strong>{{ amount }}</strong></span>
        <span>可用 <strong>{{ walletBalance }}</strong></span>
      </div>

      <button type="button" class="shop-link-button" @click="emit('shop')">前往商城购买金币 &gt;</button>

      <p v-if="insufficient" class="error">金币不足，请先购买金币</p>

      <div class="modal-actions">
        <button type="button" class="ghost" @click="emit('close')">取消</button>
        <button type="button" :disabled="insufficient" @click="emit('confirm', amount)">确定</button>
      </div>
    </section>
  </div>
</template>
