<script setup lang="ts">
const emit = defineEmits<{
  stand: [];
  buyIn: [];
  leave: [];
  shortDeckRules: [];
}>();
</script>

<template>
  <section class="table-settings-grid">
    <button type="button">桌面设置</button>
    <button type="button" @click="emit('stand')">站起围观</button>
    <button type="button" @click="emit('buyIn')">带入记分牌</button>
    <button type="button">比赛设置</button>
    <button type="button" class="short-deck-rules" @click="emit('shortDeckRules')">短牌规则</button>
    <button type="button" disabled>保位离座</button>
    <button type="button" disabled>降落伞说明</button>
    <button type="button" @click="emit('leave')">退出比赛</button>
  </section>
</template>
