import { computed, ref } from 'vue';
import { createRoom, fetchCurrentRoomHand, fetchMe, fetchRoom, fetchRoomHands, fetchUserHands, fetchWallet, joinRoom, leaveSeat, login, recharge, register, startRoomHand, submitRoomAction, takeSeat, updateProfile } from '../api/client';
import type { ProfileResponse, RechargeOption, RoomHandHistoryRecord, RoomHandState, RoomResponse, UserHandRecord, WalletResponse } from '../types/game';

const TOKEN_STORAGE_KEY = 'depu.auth.token';

function storedToken() {
  if (typeof window === 'undefined') return '';
  try {
    return window.sessionStorage.getItem(TOKEN_STORAGE_KEY) ?? '';
  } catch {
    return '';
  }
}

function setToken(nextToken: string) {
  token.value = nextToken;
  if (typeof window === 'undefined') return;
  try {
    if (nextToken) {
      window.sessionStorage.setItem(TOKEN_STORAGE_KEY, nextToken);
    } else {
      window.sessionStorage.removeItem(TOKEN_STORAGE_KEY);
    }
  } catch {
  }
}

export function emptyRoom(roomId: string): RoomResponse {
  return {
    id: roomId,
    inviteCode: '',
    ownerUserId: '',
    status: 'waiting',
    members: [],
    seats: [],
    seatCount: 0,
  };
}

const token = ref(storedToken());
const me = ref<ProfileResponse | null>(null);
const wallet = ref<WalletResponse | null>(null);
const room = ref<RoomResponse | null>(null);
const currentRoomHand = ref<RoomHandState | null>(null);
const recentRoomHands = ref<RoomHandHistoryRecord[]>([]);
const roomHistory = ref<UserHandRecord[]>([]);
const rechargeOptions = ref<RechargeOption[]>([]);
const loading = ref(false);
const error = ref('');
let roomPollTimer: number | null = null;

const myRoomSeat = computed(() => room.value?.seats.find((seat) => seat.userId && seat.userId === me.value?.id) ?? null);
const isMyTurn = computed(() => !!currentRoomHand.value && !!myRoomSeat.value && currentRoomHand.value.currentSeat === myRoomSeat.value.seatNo);
const myRoomHandPlayer = computed(() => !!currentRoomHand.value && !!myRoomSeat.value ? currentRoomHand.value.players.find((player) => player.seatNo === myRoomSeat.value?.seatNo) ?? null : null);

async function run<T>(fn: () => Promise<T>) {
  loading.value = true;
  error.value = '';
  try {
    return await fn();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
    throw err;
  } finally {
    loading.value = false;
  }
}

async function refreshProfile() {
  if (!token.value) return;
  try {
    me.value = await fetchMe(token.value);
    wallet.value = await fetchWallet(token.value);
    roomHistory.value = (await fetchUserHands(token.value)).items;
  } catch (err) {
    if (err instanceof Error && err.message.includes('authentication required')) {
      doLogout();
    }
    throw err;
  }
}

async function refreshHistoryDetails() {
  if (!token.value) return;
  await refreshProfile();
  const historyRoomId = room.value?.id || roomHistory.value[0]?.roomId;
  if (!historyRoomId) {
    recentRoomHands.value = [];
    return;
  }
  recentRoomHands.value = (await fetchRoomHands(token.value, historyRoomId)).items;
}

async function doRegister(username: string, password: string, nickname: string) {
  await run(async () => {
    const payload = await register(username, password, nickname);
    setToken(payload.token);
    await refreshProfile();
  });
}

async function doLogin(username: string, password: string) {
  await run(async () => {
    const payload = await login(username, password);
    setToken(payload.token);
    await refreshProfile();
  });
}

function doLogout() {
  stopRoomPolling();
  setToken('');
  me.value = null;
  wallet.value = null;
  room.value = null;
  currentRoomHand.value = null;
  recentRoomHands.value = [];
  roomHistory.value = [];
  error.value = '';
}

async function saveNickname(nickname: string) {
  if (!token.value) return;
  await run(async () => {
    me.value = await updateProfile(token.value, nickname);
  });
}

async function doRecharge(code: string) {
  if (!token.value) return;
  await run(async () => {
    await recharge(token.value, code);
    wallet.value = await fetchWallet(token.value);
  });
}

async function pollRoomState() {
  if (!token.value || !room.value) return;
  try {
    const latestRoom = await fetchRoom(token.value, room.value.id);
    room.value = latestRoom;
    recentRoomHands.value = (await fetchRoomHands(token.value, latestRoom.id)).items;
    if (latestRoom.status === 'playing') {
      currentRoomHand.value = await fetchCurrentRoomHand(token.value, latestRoom.id);
    }
  } catch {
  }
}

function stopRoomPolling() {
  if (roomPollTimer !== null) {
    window.clearInterval(roomPollTimer);
    roomPollTimer = null;
  }
}

function startRoomPolling() {
  stopRoomPolling();
  if (!token.value || !room.value) return;
  roomPollTimer = window.setInterval(() => {
    void pollRoomState();
  }, 2000);
}

async function doCreateRoom(payload: { ruleSetId: string; seatCount: number; minPlayersToStart: number; }) {
  if (!token.value) return null;
  return run(async () => {
    room.value = await createRoom(token.value, payload);
    recentRoomHands.value = [];
    startRoomPolling();
    return room.value;
  });
}

async function doJoinRoom(inviteCode: string) {
  if (!token.value) return null;
  return run(async () => {
    room.value = await joinRoom(token.value, inviteCode.trim());
    recentRoomHands.value = (await fetchRoomHands(token.value, room.value.id)).items;
    startRoomPolling();
    return room.value;
  });
}

async function refreshRoom() {
  if (!token.value || !room.value) return;
  await run(async () => {
    room.value = await fetchRoom(token.value, room.value!.id);
    recentRoomHands.value = (await fetchRoomHands(token.value, room.value!.id)).items;
  });
}

async function doTakeSeat(seatNo: number, buyInChips: number) {
  if (!token.value || !room.value) return;
  await run(async () => {
    room.value = await takeSeat(token.value, room.value!.id, seatNo, buyInChips);
    wallet.value = await fetchWallet(token.value);
  });
}

async function doLeaveSeat(seatNo: number) {
  if (!token.value || !room.value) return;
  await run(async () => {
    room.value = await leaveSeat(token.value, room.value!.id, seatNo);
    wallet.value = await fetchWallet(token.value);
  });
}

async function doStartRoomHand() {
  if (!token.value || !room.value) return;
  await run(async () => {
    currentRoomHand.value = await startRoomHand(token.value, room.value!.id);
    recentRoomHands.value = (await fetchRoomHands(token.value, room.value!.id)).items;
  });
}

async function refreshCurrentRoomHand() {
  if (!token.value || !room.value) return;
  await safeRefreshCurrentRoomHand();
}

async function safeRefreshCurrentRoomHand() {
  await run(async () => {
    try {
      currentRoomHand.value = await fetchCurrentRoomHand(token.value, room.value!.id);
    } catch (err) {
      if (err instanceof Error && err.message.includes('current hand not found')) {
        currentRoomHand.value = null;
        return;
      }
      throw err;
    }
    recentRoomHands.value = (await fetchRoomHands(token.value, room.value!.id)).items;
  });
}

async function doRoomAction(action: string) {
  if (!token.value || !room.value) return;
  await run(async () => {
    currentRoomHand.value = await submitRoomAction(token.value, room.value!.id, action, 0);
    recentRoomHands.value = (await fetchRoomHands(token.value, room.value!.id)).items;
    roomHistory.value = (await fetchUserHands(token.value)).items;
    wallet.value = await fetchWallet(token.value);
  });
}

export function useAppState() {
  return {
    token,
    me,
    wallet,
    room,
    currentRoomHand,
    recentRoomHands,
    roomHistory,
    rechargeOptions,
    loading,
    error,
    myRoomSeat,
    isMyTurn,
    myRoomHandPlayer,
    run,
    refreshProfile,
    refreshHistoryDetails,
    doRegister,
    doLogin,
    doLogout,
    saveNickname,
    doRecharge,
    doCreateRoom,
    doJoinRoom,
    refreshRoom,
    doTakeSeat,
    doLeaveSeat,
    doStartRoomHand,
    refreshCurrentRoomHand: safeRefreshCurrentRoomHand,
    doRoomAction,
    startRoomPolling,
    stopRoomPolling,
    pollRoomState,
  };
}
