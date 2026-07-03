import { createRouter, createWebHistory } from 'vue-router';
import LoginPage from '../pages/LoginPage.vue';
import LobbyPage from '../pages/LobbyPage.vue';
import RoomPage from '../pages/RoomPage.vue';
import RoomInfoPage from '../pages/RoomInfoPage.vue';
import RoomPlayersPage from '../pages/RoomPlayersPage.vue';
import HistoryPage from '../pages/HistoryPage.vue';
import MePage from '../pages/MePage.vue';
import PlaceholderPage from '../pages/PlaceholderPage.vue';
import RulesTestPage from '../pages/RulesTestPage.vue';
import { useAppState } from '../composables/useAppState';

const routes = [
  { path: '/', redirect: '/lobby' },
  { path: '/login', component: LoginPage },
  { path: '/lobby', component: LobbyPage },
  { path: '/room/:roomId', component: RoomPage },
  { path: '/room/:roomId/info', component: RoomInfoPage },
  { path: '/room/:roomId/players', component: RoomPlayersPage },
  { path: '/history', component: HistoryPage },
  { path: '/me', component: MePage },
  { path: '/match', component: PlaceholderPage, props: { title: '约赛', description: '约赛功能先留空展位。' } },
  { path: '/discover', component: PlaceholderPage, props: { title: '发现', description: '发现功能先留空展位。' } },
  { path: '/team', component: PlaceholderPage, props: { title: '战队', description: '战队功能先留空展位。' } },
  { path: '/rules-test', component: RulesTestPage },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  const { token, me, refreshProfile } = useAppState();
  if (!token.value && to.path !== '/login') return '/login';
  if (token.value && !me.value) {
    try {
      await refreshProfile();
    } catch {
      if (to.path !== '/login') return '/login';
    }
  }
  if (token.value && to.path === '/login') return '/lobby';
  return true;
});
