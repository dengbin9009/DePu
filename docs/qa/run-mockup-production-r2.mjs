const API = 'http://127.0.0.1:5190';
const RUN = `r2_${Date.now().toString(36)}`;

const results = [];
const state = {};

function record(name, ok, detail = {}) {
  results.push({ name, ok, detail });
  const mark = ok ? 'PASS' : 'FAIL';
  console.log(`${mark} ${name}`);
  if (!ok) {
    console.log(JSON.stringify(detail, null, 2));
  }
}

function assert(name, condition, detail = {}) {
  record(name, Boolean(condition), detail);
  if (!condition) {
    throw new Error(`failed: ${name}`);
  }
}

function payloadOf(message) {
  if (!message || message.payload == null) return {};
  if (typeof message.payload === 'string') return JSON.parse(message.payload);
  return message.payload;
}

async function request(method, path, { token, body, raw = false } = {}) {
  const headers = {};
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const res = await fetch(`${API}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  let json = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }
  if (raw) return { status: res.status, json, text };
  return { status: res.status, json };
}

async function register(username, nickname) {
  const res = await request('POST', '/api/auth/register', {
    body: { username, password: 'password123', nickname },
  });
  assert(`注册 ${username}`, res.status === 201 && res.json.token, res);
  return res.json;
}

async function login(username) {
  const res = await request('POST', '/api/auth/login', {
    body: { username, password: 'password123' },
  });
  assert(`登录 ${username}`, res.status === 200 && res.json.token, res);
  return res.json;
}

function token(user) {
  return user.token;
}

async function httpSuite() {
  const health = await request('GET', '/health');
  assert('健康检查', health.status === 200 && health.json.status === 'ok', health);

  const noAuth = await request('GET', '/api/me');
  assert('未登录访问 /api/me 被拒绝', noAuth.status === 401 && noAuth.json.code === 'unauthorized', noAuth);

  const shortPass = await request('POST', '/api/auth/register', {
    body: { username: 'short_r2', password: 'short', nickname: '短密码R2' },
  });
  assert('短密码注册被拒绝', shortPass.status === 400 && shortPass.json.code === 'invalid_password', shortPass);

  state.owner = await register(`owner_${RUN}`, `房主${RUN}`);
  state.player = await register(`player_${RUN}`, `玩家${RUN}`);
  state.outsider = await register(`outsider_${RUN}`, `路人${RUN}`);

  const dupUser = await request('POST', '/api/auth/register', {
    body: { username: `owner_${RUN}`, password: 'password123', nickname: `房主${RUN}重复` },
  });
  assert('重复用户名被拒绝', dupUser.status === 409 && dupUser.json.code === 'duplicate_username', dupUser);

  const dupNick = await request('POST', '/api/auth/register', {
    body: { username: `dup_nick_${RUN}`, password: 'password123', nickname: `房主${RUN}` },
  });
  assert('重复昵称被拒绝', dupNick.status === 409 && dupNick.json.code === 'duplicate_nickname', dupNick);

  const badLogin = await request('POST', '/api/auth/login', {
    body: { username: `owner_${RUN}`, password: 'wrong-password' },
  });
  assert('错误密码登录被拒绝', badLogin.status === 401 && badLogin.json.code === 'unauthorized', badLogin);

  await login(`owner_${RUN}`);

  const rules = await request('GET', '/api/rulesets');
  assert('规则集接口返回 holdem/short-deck', rules.status === 200 && Array.isArray(rules.json) && rules.json.some((r) => r.id === 'short-deck') && rules.json.some((r) => r.id === 'long-holdem'), rules);

  const invalidRoom = await request('POST', '/api/rooms', {
    token: token(state.owner),
    body: { ruleSetId: 'short-deck', mode: 'sng', variant: 'short_holdem', seatCount: 9, minPlayersToStart: 2 },
  });
  assert('暂不支持 SNG 建房被明确拒绝', invalidRoom.status === 400 && invalidRoom.json.field === 'mode', invalidRoom);

  const invalidSeatCount = await request('POST', '/api/rooms', {
    token: token(state.owner),
    body: { ruleSetId: 'short-deck', mode: 'training', variant: 'short_holdem', seatCount: 10, minPlayersToStart: 2 },
  });
  assert('超过 9 人建房被拒绝', invalidSeatCount.status === 400 && invalidSeatCount.json.field === 'seatCount', invalidSeatCount);

  const invalidBuyInRange = await request('POST', '/api/rooms', {
    token: token(state.owner),
    body: { ruleSetId: 'short-deck', mode: 'training', variant: 'short_holdem', minBuyIn: 8000, maxBuyIn: 2000, seatCount: 9, minPlayersToStart: 2 },
  });
  assert('最小带入大于最大带入被拒绝', invalidBuyInRange.status === 400 && invalidBuyInRange.json.field === 'maxBuyIn', invalidBuyInRange);

  const roomRes = await request('POST', '/api/rooms', {
    token: token(state.owner),
    body: {
      ruleSetId: 'short-deck',
      name: 'R2工业验收桌',
      mode: 'training',
      variant: 'short_holdem',
      ante: 20,
      minBuyIn: 2000,
      maxBuyIn: 8000,
      buyInCap: 60000,
      durationMinutes: 120,
      seatCount: 9,
      minPlayersToStart: 2,
    },
  });
  assert('创建短牌训练赛房间成功', roomRes.status === 201 && roomRes.json.id && roomRes.json.inviteCode && roomRes.json.seatCount === 9, roomRes);
  state.room = roomRes.json;

  const getRoom = await request('GET', `/api/rooms/${state.room.id}`, { token: token(state.owner) });
  assert('房间详情返回展示配置', getRoom.status === 200 && getRoom.json.name === 'R2工业验收桌' && getRoom.json.ante === 20 && getRoom.json.minBuyIn === 2000 && getRoom.json.maxBuyIn === 8000, getRoom);

  const nonMemberSeat = await request('POST', `/api/rooms/${state.room.id}/seats/2`, {
    token: token(state.outsider),
    body: { buyInChips: 2000 },
  });
  assert('非成员不能直接入座', nonMemberSeat.status === 403 && nonMemberSeat.json.code === 'forbidden', nonMemberSeat);

  const nonMemberLeaderboard = await request('GET', `/api/rooms/${state.room.id}/leaderboard`, { token: token(state.outsider) });
  assert('非成员不能查看战绩榜', nonMemberLeaderboard.status === 403 && nonMemberLeaderboard.json.code === 'forbidden', nonMemberLeaderboard);

  const ownerLowBuyIn = await request('POST', `/api/rooms/${state.room.id}/seats/1`, {
    token: token(state.owner),
    body: { buyInChips: 1999 },
  });
  assert('低于最小带入被拒绝', ownerLowBuyIn.status === 400 && ownerLowBuyIn.json.field === 'buyInChips', ownerLowBuyIn);

  const ownerOverBuyIn = await request('POST', `/api/rooms/${state.room.id}/seats/1`, {
    token: token(state.owner),
    body: { buyInChips: 8001 },
  });
  assert('高于带入上限被拒绝', ownerOverBuyIn.status === 400 && ownerOverBuyIn.json.field === 'buyInChips', ownerOverBuyIn);

  const ownerSeat = await request('POST', `/api/rooms/${state.room.id}/seats/1`, {
    token: token(state.owner),
    body: { buyInChips: 2000 },
  });
  assert('房主最小带入入座成功', ownerSeat.status === 200 && ownerSeat.json.seats.some((s) => s.seatNo === 1 && s.userId === state.owner.user.id && s.buyInChips === 2000), ownerSeat);

  const ownerSecondSeat = await request('POST', `/api/rooms/${state.room.id}/seats/3`, {
    token: token(state.owner),
    body: { buyInChips: 2000 },
  });
  assert('同一用户重复坐其他座位被拒绝', ownerSecondSeat.status === 409 && ownerSecondSeat.json.code === 'already_seated', ownerSecondSeat);

  const leaveOwner = await request('DELETE', `/api/rooms/${state.room.id}/seats/1`, { token: token(state.owner) });
  assert('等待态站起释放座位且保留成员资格', leaveOwner.status === 200 && leaveOwner.json.ownerUserId === state.owner.user.id && leaveOwner.json.members.some((m) => m.userId === state.owner.user.id) && leaveOwner.json.seats.every((s) => s.userId !== state.owner.user.id), leaveOwner);

  const badJoin = await request('POST', '/api/rooms/join', {
    token: token(state.player),
    body: { inviteCode: 'BADR2' },
  });
  assert('无效邀请码加入被拒绝', badJoin.status === 400 && badJoin.json.code === 'invalid_invite_code', badJoin);

  const join = await request('POST', '/api/rooms/join', {
    token: token(state.player),
    body: { inviteCode: `  ${String(state.room.inviteCode).toLowerCase()}  ` },
  });
  assert('邀请码大小写和空格归一化加入成功', join.status === 200 && join.json.members.some((m) => m.userId === state.player.user.id), join);

  const joinAgain = await request('POST', '/api/rooms/join', {
    token: token(state.player),
    body: { inviteCode: state.room.inviteCode },
  });
  assert('重复加入保持幂等', joinAgain.status === 200 && joinAgain.json.members.filter((m) => m.userId === state.player.user.id).length === 1, joinAgain);

  const playerInsufficient = await request('POST', `/api/rooms/${state.room.id}/seats/2`, {
    token: token(state.player),
    body: { buyInChips: 8000 },
  });
  assert('余额不足大额带入被拒绝', playerInsufficient.status === 409 && playerInsufficient.json.code === 'insufficient_coins', playerInsufficient);

  const options = await request('GET', '/api/recharge/options');
  assert('充值档位返回 small/medium/large', options.status === 200 && options.json.options.length === 3 && options.json.options.some((o) => o.code === 'large'), options);

  const invalidRecharge = await request('POST', '/api/recharge', {
    token: token(state.player),
    body: { optionCode: 'large', confirm: false },
  });
  assert('未确认充值被拒绝', invalidRecharge.status === 400 && invalidRecharge.json.code === 'invalid_recharge', invalidRecharge);

  const recharge = await request('POST', '/api/recharge', {
    token: token(state.player),
    body: { optionCode: 'large', confirm: true },
  });
  assert('模拟充值成功并写入交易', recharge.status === 200 && recharge.json.walletBalance >= 15000 && recharge.json.transaction.type === 'recharge_simulated', recharge);

  const ownerSeatAgain = await request('POST', `/api/rooms/${state.room.id}/seats/1`, {
    token: token(state.owner),
    body: { buyInChips: 2000 },
  });
  assert('房主重新入座成功', ownerSeatAgain.status === 200 && ownerSeatAgain.json.seats.some((s) => s.seatNo === 1 && s.userId === state.owner.user.id), ownerSeatAgain);

  const playerSeat = await request('POST', `/api/rooms/${state.room.id}/seats/2`, {
    token: token(state.player),
    body: { buyInChips: 8000 },
  });
  assert('充值后玩家 8K 带入成功', playerSeat.status === 200 && playerSeat.json.seats.some((s) => s.seatNo === 2 && s.userId === state.player.user.id && s.buyInChips === 8000), playerSeat);

  const seatTaken = await request('POST', `/api/rooms/${state.room.id}/seats/2`, {
    token: token(state.outsider),
    body: { buyInChips: 2000 },
  });
  assert('未入会路人抢已占座位仍按成员权限拒绝', seatTaken.status === 403 && seatTaken.json.code === 'forbidden', seatTaken);

  const nonOwnerStart = await request('POST', `/api/rooms/${state.room.id}/start`, { token: token(state.player) });
  assert('非房主不能开局', nonOwnerStart.status === 403 && nonOwnerStart.json.code === 'not_room_owner', nonOwnerStart);

  const start = await request('POST', `/api/rooms/${state.room.id}/start`, { token: token(state.owner) });
  assert('房主开局成功', start.status === 200 && start.json.handId && start.json.status, start);
  state.hand = start.json;

  const current = await request('GET', `/api/rooms/${state.room.id}/current-hand`, { token: token(state.owner) });
  assert('当前手牌接口返回开局状态', current.status === 200 && current.json.handId === state.hand.handId && Array.isArray(current.json.availableActions), current);
  state.hand = current.json;

  const leaveDuringPlaying = await request('DELETE', `/api/rooms/${state.room.id}/seats/1`, { token: token(state.owner) });
  assert('游戏中站起被拒绝', leaveDuringPlaying.status === 409 && leaveDuringPlaying.json.code === 'room_not_waiting', leaveDuringPlaying);

  const wrongActor = state.hand.currentSeat === 1 ? state.player : state.owner;
  const nonActorAction = await request('POST', `/api/rooms/${state.room.id}/actions`, {
    token: token(wrongActor),
    body: { action: 'fold', amount: 0 },
  });
  assert('非当前行动玩家不能操作', nonActorAction.status === 403 && nonActorAction.json.code === 'not_your_turn', nonActorAction);

  const actor = state.hand.currentSeat === 1 ? state.owner : state.player;
  const action = state.hand.availableActions.includes('check') ? 'check' : state.hand.availableActions.includes('call') ? 'call' : 'fold';
  const act = await request('POST', `/api/rooms/${state.room.id}/actions`, {
    token: token(actor),
    body: { action, amount: 0 },
  });
  assert('当前行动玩家可提交合法动作', act.status === 200 && act.json.handId === state.hand.handId, act);

  const leaderboard = await request('GET', `/api/rooms/${state.room.id}/leaderboard`, { token: token(state.owner) });
  assert('成员可查看当前战绩榜', leaderboard.status === 200 && Array.isArray(leaderboard.json.items), leaderboard);

  const recentHands = await request('GET', `/api/rooms/${state.room.id}/hands/recent`, { token: token(state.owner) });
  assert('牌谱最近记录接口可访问', recentHands.status === 200 && Array.isArray(recentHands.json.items), recentHands);

  const missingReplay = await request('GET', `/api/rooms/${state.room.id}/hands/not_exist/replay`, { token: token(state.owner) });
  assert('不存在牌谱回放返回 replay_not_found', missingReplay.status === 404 && missingReplay.json.code === 'replay_not_found', missingReplay);

  const myHands = await request('GET', '/api/me/hands', { token: token(state.owner) });
  assert('我的手牌接口可访问', myHands.status === 200 && Array.isArray(myHands.json.items), myHands);

  const wallet = await request('GET', '/api/me/wallet', { token: token(state.player) });
  assert('钱包接口包含充值和买入交易', wallet.status === 200 && wallet.json.transactions.some((t) => t.type === 'recharge_simulated') && wallet.json.transactions.some((t) => t.type === 'buy_in'), wallet);

  const game = await request('POST', '/api/games', {
    body: {
      rulesetId: 'short-deck',
      buttonSeat: 1,
      bettingStructure: { type: 'ante', ante: 10, buttonBlind: 20 },
      dealMode: 'random',
      seats: [
        { seatNo: 1, name: 'A', stack: 1000 },
        { seatNo: 2, name: 'B', stack: 1000 },
      ],
    },
  });
  assert('规则测试页 /api/games 创建兼容', game.status === 201 && game.json.id && game.json.legalActions.length > 0, game);

  const badGameReplay = await request('POST', `/api/games/${game.json.id}/replay`, {
    body: { toSeq: 9999 },
  });
  assert('规则测试页回放越界返回 replay_out_of_range', badGameReplay.status === 400 && badGameReplay.json.code === 'replay_out_of_range', badGameReplay);
}

function b64url(input) {
  return Buffer.from(input).toString('base64url');
}

async function socketSuite() {
  const wsBase = 'ws://127.0.0.1:5190';
  const mk = (name, user) => new Promise((resolve, reject) => {
    const ws = new WebSocket(`${wsBase}/api/socket?token=${user.token}`);
    const messages = [];
    const waiters = [];
    const timer = setTimeout(() => reject(new Error(`${name} websocket timeout`)), 3000);
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      messages.push(msg);
      for (let i = 0; i < waiters.length; i += 1) {
        const waiter = waiters[i];
        if (waiter.predicate(msg)) {
          waiters.splice(i, 1);
          waiter.resolve(msg);
          break;
        }
      }
    };
    ws.onerror = reject;
    ws.onopen = () => {
      const waitFor = (predicate, timeoutMs = 3000) => new Promise((res, rej) => {
        const existing = messages.find(predicate);
        if (existing) {
          res(existing);
          return;
        }
        const waiter = { predicate, resolve: res };
        waiters.push(waiter);
        setTimeout(() => {
          const idx = waiters.indexOf(waiter);
          if (idx >= 0) waiters.splice(idx, 1);
          rej(new Error(`${name} wait timeout`));
        }, timeoutMs);
      });
      clearTimeout(timer);
      resolve({
        ws,
        messages,
        send: (payload) => ws.send(JSON.stringify(payload)),
        waitFor,
        close: () => ws.close(),
      });
    };
  });

  const socketOwner = await register(`socket_owner_${RUN}`, `Socket房主${RUN}`);
  const socketPlayer = await register(`socket_player_${RUN}`, `Socket玩家${RUN}`);
  const socketOutsider = await register(`socket_outsider_${RUN}`, `Socket路人${RUN}`);
  const room = await request('POST', '/api/rooms', {
    token: socketOwner.token,
    body: { ruleSetId: 'long-holdem', name: 'Socket R2', mode: 'training', variant: 'holdem', seatCount: 6, minPlayersToStart: 2, minBuyIn: 1000, maxBuyIn: 5000 },
  });
  await request('POST', '/api/rooms/join', { token: socketPlayer.token, body: { inviteCode: room.json.inviteCode } });
  await request('POST', `/api/rooms/${room.json.id}/seats/1`, { token: socketOwner.token, body: { buyInChips: 1000 } });
  await request('POST', `/api/rooms/${room.json.id}/seats/2`, { token: socketPlayer.token, body: { buyInChips: 1000 } });

  const owner = await mk('owner', socketOwner);
  await owner.waitFor((m) => m.type === 'connection.ready');
  owner.send({ type: 'room.subscribe', requestId: 'sub-owner-r2', roomId: room.json.id });
  await owner.waitFor((m) => m.type === 'ack' && m.requestId === 'sub-owner-r2');
  const snapshot = await owner.waitFor((m) => m.type === 'room.snapshot' && m.requestId === 'sub-owner-r2');
  const snapshotPayload = payloadOf(snapshot);
  assert('Socket 订阅返回房间快照/战绩/聊天/在线状态', snapshotPayload.room.id === room.json.id && Array.isArray(snapshotPayload.leaderboard) && Array.isArray(snapshotPayload.recentChatMessages) && Array.isArray(snapshotPayload.presence), snapshotPayload);

  const outsider = await mk('outsider', socketOutsider);
  await outsider.waitFor((m) => m.type === 'connection.ready');
  outsider.send({ type: 'room.subscribe', requestId: 'sub-outsider-r2', roomId: room.json.id });
  const outsiderError = await outsider.waitFor((m) => m.type === 'error' && m.requestId === 'sub-outsider-r2');
  assert('Socket 非成员订阅被拒绝', payloadOf(outsiderError).code === 'forbidden', outsiderError);

  owner.send({ type: 'chat.send', requestId: 'chat-empty-r2', roomId: room.json.id, payload: { kind: 'text', text: '   ' } });
  const emptyChat = await owner.waitFor((m) => m.type === 'error' && m.requestId === 'chat-empty-r2');
  assert('Socket 空聊天被拒绝', payloadOf(emptyChat).code === 'chat_message_empty', emptyChat);

  owner.send({ type: 'chat.send', requestId: 'chat-good-r2', roomId: room.json.id, payload: { kind: 'text', text: '工业验收消息' } });
  await owner.waitFor((m) => m.type === 'ack' && m.requestId === 'chat-good-r2');
  const chatMsg = await owner.waitFor((m) => m.type === 'chat.message' && m.requestId === 'chat-good-r2');
  assert('Socket 文本聊天广播成功', payloadOf(chatMsg).message.text === '工业验收消息', chatMsg);

  owner.send({ type: 'chat.send', requestId: 'chat-rate-r2', roomId: room.json.id, payload: { kind: 'text', text: '太快' } });
  const rate = await owner.waitFor((m) => m.type === 'error' && m.requestId === 'chat-rate-r2');
  assert('Socket 聊天频率限制生效', payloadOf(rate).code === 'chat_rate_limited', rate);

  await new Promise((resolve) => setTimeout(resolve, 350));
  owner.send({ type: 'chat.send', requestId: 'chat-emoji-r2', roomId: room.json.id, payload: { kind: 'emoji', emojiCode: 'nice_hand' } });
  await owner.waitFor((m) => m.type === 'ack' && m.requestId === 'chat-emoji-r2');
  const emoji = await owner.waitFor((m) => m.type === 'chat.message' && m.requestId === 'chat-emoji-r2');
  assert('Socket 表情聊天广播成功', payloadOf(emoji).message.emojiCode === 'nice_hand', emoji);

  owner.send({ type: 'room.start_hand', requestId: 'start-socket-r2', roomId: room.json.id });
  await owner.waitFor((m) => m.type === 'ack' && m.requestId === 'start-socket-r2');
  const started = await owner.waitFor((m) => m.type === 'hand.started' && m.requestId === 'start-socket-r2');
  const startedPayload = payloadOf(started);
  assert('Socket 房主开局广播 hand.started', startedPayload.hand.handId && startedPayload.hand.roomId === room.json.id, startedPayload);

  owner.send({ type: 'room.action', requestId: 'bad-action-r2', roomId: room.json.id, payload: { action: 'raise', amount: -1 } });
  const badAction = await owner.waitFor((m) => m.type === 'error' && m.requestId === 'bad-action-r2');
  assert('Socket 非法动作不广播成功', payloadOf(badAction).code === 'invalid_action', badAction);

  owner.send({ type: 'room.unsubscribe', requestId: 'unsub-r2', roomId: room.json.id });
  const unsub = await owner.waitFor((m) => m.type === 'ack' && m.requestId === 'unsub-r2');
  assert('Socket 取消订阅确认', unsub.type === 'ack', unsub);

  owner.close();
  outsider.close();
}

await httpSuite();
await socketSuite();

const summary = {
  generatedAt: new Date().toISOString(),
  total: results.length,
  passed: results.filter((r) => r.ok).length,
  failed: results.filter((r) => !r.ok).length,
  results,
  roomId: state.room?.id,
  inviteCode: state.room?.inviteCode,
};

await import('node:fs/promises').then((fs) => fs.writeFile('/Users/dengbin/Code/github/DePu/docs/qa/mockup-production-r2-api-socket-20260707.json', JSON.stringify(summary, null, 2)));
console.log(JSON.stringify({ total: summary.total, passed: summary.passed, failed: summary.failed, roomId: summary.roomId, inviteCode: summary.inviteCode }, null, 2));
