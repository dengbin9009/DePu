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
    <aside class="table-drawer" :class="`drawer-${placement}`">
      <button type="button" class="drawer-close" @click="emit('close')">×</button>
      <h2 v-if="title">{{ title }}</h2>
      <slot />
    </aside>
  </div>
</template>
