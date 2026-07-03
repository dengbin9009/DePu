<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import BottomTabBar from './components/BottomTabBar.vue';

const route = useRoute();
const showTabs = computed(() => route.path !== '/login' && route.path !== '/rules-test' && !route.path.startsWith('/room/'));
</script>

<template>
  <div class="mobile-app-shell">
    <router-view />
    <BottomTabBar v-if="showTabs" />
  </div>
</template>
