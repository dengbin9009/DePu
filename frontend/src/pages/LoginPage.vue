<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAppState } from '../composables/useAppState';

const { doLogin, doRegister, loading, error } = useAppState();
const router = useRouter();
const mode = ref<'login' | 'register'>('login');
const username = ref('owner01');
const password = ref('password1');
const nickname = ref('房主A');

async function submit() {
  try {
    if (mode.value === 'login') {
      await doLogin(username.value, password.value);
    } else {
      await doRegister(username.value, password.value, nickname.value);
    }
    router.push('/lobby');
  } catch {
  }
}
</script>

<template>
  <main class="page-shell auth-page">
    <section class="panel mobile-panel auth-card">
      <h1>{{ mode === 'login' ? '登录' : '注册' }}</h1>
      <p class="auth-subtitle">{{ mode === 'login' ? '使用账号进入大厅与多人房间。' : '创建新账号后可直接进入大厅。' }}</p>
      <label>用户名 <input v-model="username" type="text" /></label>
      <label>密码 <input v-model="password" type="password" /></label>
      <label v-if="mode === 'register'">昵称 <input v-model="nickname" type="text" /></label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="button" :disabled="loading" @click="submit">{{ mode === 'login' ? '登录' : '注册' }}</button>
      <button type="button" class="ghost auth-switch" :disabled="loading" @click="mode = mode === 'login' ? 'register' : 'login'">
        {{ mode === 'login' ? '去注册' : '去登录' }}
      </button>
    </section>
  </main>
</template>
