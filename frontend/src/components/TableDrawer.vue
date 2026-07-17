<script setup lang="ts">
defineProps<{
  open: boolean;
  title?: string;
  placement: 'left' | 'bottom' | 'right';
}>();

const emit = defineEmits<{ close: [] }>();
</script>

<template>
  <div v-if="open" class="table-drawer-backdrop" @click.self="emit('close')">
    <aside class="table-drawer" :class="`drawer-${placement}`" role="dialog" aria-modal="true" :aria-label="title || '牌桌抽屉'">
      <button type="button" class="drawer-close" aria-label="关闭抽屉" @click="emit('close')">×</button>
      <div class="table-drawer-content">
        <h2 v-if="title">{{ title }}</h2>
        <slot />
      </div>
    </aside>
  </div>
</template>
