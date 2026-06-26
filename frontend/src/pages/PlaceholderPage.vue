<script setup lang="ts">
defineProps<{ title: string; description: string }>();
</script>

<template>
  <main class="page-shell">
    <section class="panel mobile-panel">
      <h1>{{ title }}</h1>
      <p>{{ description }}</p>
    </section>
  </main>
</template>
