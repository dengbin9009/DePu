<script setup lang="ts">
import { useRoute } from 'vue-router';

const route = useRoute();
const tabs = [
  { to: '/match', label: '约赛' },
  { to: '/discover', label: '发现' },
  { to: '/lobby', label: '大厅' },
  { to: '/team', label: '战队' },
  { to: '/me', label: '我的' },
];
</script>

<template>
  <nav class="bottom-tabs">
    <router-link v-for="tab in tabs" :key="tab.to" :to="tab.to" class="tab-item" :class="{ active: route.path === tab.to }">
      <span>{{ tab.label }}</span>
    </router-link>
  </nav>
</template>
