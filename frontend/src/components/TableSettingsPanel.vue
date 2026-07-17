<script setup lang="ts">
defineProps<{ canStand: boolean; canBuyIn: boolean; loading: boolean }>();

const emit = defineEmits<{
  stand: [];
  buyIn: [];
  leave: [];
  shortDeckRules: [];
}>();
</script>

<template>
  <section class="table-settings-grid">
    <button type="button" disabled title="桌面设置暂未开放">桌面设置（暂未开放）</button>
    <button type="button" :disabled="loading || !canStand" :title="canStand ? '站起围观' : '当前未入座'" @click="emit('stand')">站起围观</button>
    <button type="button" :disabled="loading || !canBuyIn" :title="canBuyIn ? '带入记分牌' : '已入座玩家暂不支持中途补充记分牌'" @click="emit('buyIn')">带入记分牌</button>
    <button type="button" disabled title="比赛设置暂未开放">比赛设置（暂未开放）</button>
    <button type="button" class="short-deck-rules" @click="emit('shortDeckRules')">短牌规则</button>
    <button type="button" disabled title="保位离座暂未开放">保位离座（暂未开放）</button>
    <button type="button" disabled title="降落伞功能暂未开放">降落伞（暂未开放）</button>
    <button type="button" :disabled="loading" @click="emit('leave')">退出比赛</button>
  </section>
</template>
