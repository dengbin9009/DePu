import { computed, ref } from 'vue';
import { createRoom, fetchCurrentRoomHand, fetchMe, fetchRoom, fetchRoomHands, fetchUserHands, fetchWallet, joinRoom, leaveSeat, login, recharge, register, startRoomHand, submitRoomAction, takeSeat, updateProfile } from '../api/client';
import type { ProfileResponse, RechargeOption, RoomHandHistoryRecord, RoomHandState, RoomResponse, UserHandRecord, WalletResponse } from '../types/game';

const token = ref('');
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
  me.value = await fetchMe(token.value);
  wallet.value = await fetchWallet(token.value);
  roomHistory.value = (await fetchUserHands(token.value)).items;
}

async function doRegister(username: string, password: string, nickname: string) {
  await run(async () => {
    const payload = await register(username, password, nickname);
    token.value = payload.token;
    await refreshProfile();
  });
}

async function doLogin(username: string, password: string) {
  await run(async () => {
    const payload = await login(username, password);
    token.value = payload.token;
    await refreshProfile();
  });
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
  await run(async () => {
    currentRoomHand.value = await fetchCurrentRoomHand(token.value, room.value!.id);
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
    doRegister,
    doLogin,
    saveNickname,
    doRecharge,
    doCreateRoom,
    doJoinRoom,
    refreshRoom,
    doTakeSeat,
    doLeaveSeat,
    doStartRoomHand,
    refreshCurrentRoomHand,
    doRoomAction,
    startRoomPolling,
    stopRoomPolling,
    pollRoomState,
  };
}
