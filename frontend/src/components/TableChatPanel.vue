<script setup lang="ts">
import { ref } from 'vue';
import type { RoomChatMessage } from '../types/game';

defineProps<{ messages: RoomChatMessage[]; loading: boolean }>();
const emit = defineEmits<{ send: [text: string] }>();
const text = ref('');

function sendRoomChat() {
  const value = text.value.trim();
  if (!value) return;
  emit('send', value);
  text.value = '';
}
</script>

<template>
  <section class="table-chat-panel">
    <button type="button" class="quick-chat-tab">常用语</button>
    <h2>聊天记录</h2>
    <ol>
      <li v-for="message in messages" :key="message.id">{{ message.nickname }}：{{ message.kind === 'emoji' ? message.emojiCode : message.text }}</li>
      <li v-if="!messages.length" class="muted">暂无聊天记录</li>
    </ol>
    <div class="drawer-input-row">
      <input v-model="text" maxlength="40" placeholder="请输入聊天内容，上限40个汉字" @keyup.enter="sendRoomChat" />
      <button type="button" :disabled="loading || !text.trim()" @click="sendRoomChat">发送</button>
    </div>
  </section>
</template>
