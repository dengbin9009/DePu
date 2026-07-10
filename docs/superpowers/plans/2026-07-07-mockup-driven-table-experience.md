# Mockup-Driven Table Experience Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the OpenSpec `mockup-driven-table-experience` change: configurable training-room creation, gold shop, table buy-in modal, mobile poker-table layout, and in-table chat/score/replay/settings panels.

**Architecture:** Keep the Go backend as the authority for room configuration, buy-in validation, wallet changes, socket snapshots, hand history, and replay privacy. Extend room metadata first, then update the Vue state/API layer, then replace the current engineering-style room UI with reusable mobile table components and panels that consume the same authoritative state.

**Tech Stack:** Go `net/http` + MySQL storage, Vue 3 + TypeScript + Vue Router + Vite, Vitest, OpenSpec CLI.

---

## File Map

Backend:

- Modify `backend/internal/storage/storage.go`: extend `RoomRecord`, migrate room metadata columns, add `CreateRoomOptions`, validate room config, validate buy-in range.
- Modify `backend/internal/api/server.go`: accept extended create-room payload, map validation errors to field-level HTTP errors, keep existing route compatibility.
- Modify `backend/internal/api/socket.go`: ensure `room.snapshot` and `room.updated` naturally include extended `RoomRecord` fields.
- Modify `backend/internal/api/room_multiplayer_test.go`: add config and buy-in validation tests.
- Modify `backend/internal/storage/storage_test.go`: add schema/default/validation tests when direct store coverage is useful.

Frontend state/API/types:

- Modify `frontend/src/types/game.ts`: add room metadata fields and create-room payload type.
- Modify `frontend/src/api/client.ts`: send extended create-room payload without breaking old callers.
- Modify `frontend/src/api/client.test.ts`: assert extended payload serialization.
- Modify `frontend/src/composables/useAppState.ts`: add shop return context and extended `doCreateRoom` payload.
- Modify `frontend/src/router/index.ts`: add `/create-match` and `/shop` routes.

Frontend pages/components:

- Create `frontend/src/pages/CreateMatchPage.vue`: effect-image-style create competition page.
- Create `frontend/src/pages/ShopPage.vue`: gold shop with simulated recharge.
- Create `frontend/src/components/BuyInModal.vue`: table buy-in modal.
- Create `frontend/src/components/TableDrawer.vue`: shared drawer/sheet wrapper.
- Create `frontend/src/components/TableChatPanel.vue`: chat bottom sheet.
- Create `frontend/src/components/TableScorePanel.vue`: current score side drawer.
- Create `frontend/src/components/TableReplayPanel.vue`: in-table hand replay panel.
- Create `frontend/src/components/TableSettingsPanel.vue`: settings bottom menu.
- Modify `frontend/src/pages/LobbyPage.vue`: route create-room flow to create match.
- Modify `frontend/src/pages/RoomPage.vue`: mobile table stage, modal, drawers, owner buttons, seat actions.
- Modify `frontend/src/pages/MePage.vue`: enable shop entry instead of disabled shop button.
- Modify `frontend/src/style.css`: mobile table, shop, create-match, modal, drawer, and panel styles.

Frontend tests:

- Modify `frontend/src/multiplayerTable.test.ts`: update source-contract tests to new table surface.
- Modify `frontend/src/App.visual.test.ts`: update visual contract tokens.
- Add `frontend/src/mockupDrivenExperience.test.ts`: focused contract tests for create match, shop, buy-in modal, and panels.

Validation:

- `npx --yes @fission-ai/openspec validate mockup-driven-table-experience --strict --no-interactive`
- `cd backend && go test ./... -count=1`
- `cd frontend && npm test -- --run`
- `cd frontend && npm run typecheck`

---

## Task 1: Backend Room Metadata And Buy-In Rules

**Files:**
- Modify: `backend/internal/storage/storage.go`
- Modify: `backend/internal/api/server.go`
- Test: `backend/internal/api/room_multiplayer_test.go`
- Test: `backend/internal/storage/storage_test.go`

- [ ] **Step 1: Write failing API tests for extended room creation**

Append tests to `backend/internal/api/room_multiplayer_test.go`:

```go
func TestCreateRoomWithMockupDrivenConfig(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner_cfg", "配置房主")

	body := []byte(`{
		"ruleSetId":"short-deck",
		"name":"周末短牌局",
		"mode":"training",
		"variant":"short_holdem",
		"ante":20,
		"minBuyIn":2000,
		"maxBuyIn":8000,
		"buyInCap":60000,
		"durationMinutes":120,
		"seatCount":9,
		"minPlayersToStart":2
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", res.Code, res.Body.String())
	}

	var room struct {
		Name            string `json:"name"`
		Mode            string `json:"mode"`
		Variant         string `json:"variant"`
		Ante            int    `json:"ante"`
		MinBuyIn        int    `json:"minBuyIn"`
		MaxBuyIn        int    `json:"maxBuyIn"`
		BuyInCap        int    `json:"buyInCap"`
		DurationMinutes int    `json:"durationMinutes"`
		SeatCount       int    `json:"seatCount"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if room.Name != "周末短牌局" || room.Mode != "training" || room.Variant != "short_holdem" {
		t.Fatalf("unexpected room metadata: %#v", room)
	}
	if room.Ante != 20 || room.MinBuyIn != 2000 || room.MaxBuyIn != 8000 || room.BuyInCap != 60000 || room.DurationMinutes != 120 || room.SeatCount != 9 {
		t.Fatalf("unexpected room config: %#v", room)
	}
}

func TestCreateRoomRejectsUnsupportedModeAndInvalidBuyInRange(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner_bad_cfg", "配置错误房主")

	cases := []struct {
		name string
		body string
		code string
		field string
	}{
		{
			name: "unsupported sng",
			body: `{"ruleSetId":"short-deck","mode":"sng","variant":"short_holdem","seatCount":9,"minPlayersToStart":2}`,
			code: "unsupported_room_mode",
			field: "mode",
		},
		{
			name: "min buy in greater than max",
			body: `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","minBuyIn":9000,"maxBuyIn":2000,"seatCount":9,"minPlayersToStart":2}`,
			code: "invalid_room_config",
			field: "minBuyIn",
		},
		{
			name: "unsupported omaha",
			body: `{"ruleSetId":"omaha","mode":"training","variant":"omaha","seatCount":9,"minPlayersToStart":2}`,
			code: "unsupported_variant",
			field: "variant",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Authorization", "Bearer "+ownerToken)
			res := httptest.NewRecorder()
			server.Routes().ServeHTTP(res, req)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
			var errBody ErrorResponse
			if err := json.Unmarshal(res.Body.Bytes(), &errBody); err != nil {
				t.Fatal(err)
			}
			if errBody.Code != tc.code || errBody.Error.Field != tc.field {
				t.Fatalf("error=%#v, want code=%s field=%s", errBody, tc.code, tc.field)
			}
		})
	}
}
```

- [ ] **Step 2: Write failing buy-in range API test**

Append to `backend/internal/api/room_multiplayer_test.go`:

```go
func TestTakeSeatRejectsBuyInOutsideRoomRange(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner_range", "买入房主")
	playerToken := registerUser(t, server, "player_range", "买入玩家")

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{
		"ruleSetId":"short-deck",
		"mode":"training",
		"variant":"short_holdem",
		"minBuyIn":2000,
		"maxBuyIn":6000,
		"seatCount":9,
		"minPlayersToStart":2
	}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var room map[string]any
	_ = json.Unmarshal(createRes.Body.Bytes(), &room)
	roomID := room["id"].(string)
	inviteCode := room["inviteCode"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+inviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK {
		t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String())
	}

	for _, tc := range []struct {
		amount int
		field string
	}{
		{amount: 1000, field: "buyInChips"},
		{amount: 7000, field: "buyInChips"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/2", bytes.NewReader([]byte(fmt.Sprintf(`{"buyInChips":%d}`, tc.amount))))
		req.Header.Set("Authorization", "Bearer "+playerToken)
		res := httptest.NewRecorder()
		server.Routes().ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("amount=%d status=%d body=%s", tc.amount, res.Code, res.Body.String())
		}
		var errBody ErrorResponse
		if err := json.Unmarshal(res.Body.Bytes(), &errBody); err != nil {
			t.Fatal(err)
		}
		if errBody.Code != "invalid_buy_in" || errBody.Error.Field != tc.field {
			t.Fatalf("amount=%d error=%#v", tc.amount, errBody)
		}
	}
}
```

This test needs `fmt`; add it to the import block.

- [ ] **Step 3: Run backend tests and confirm they fail**

Run:

```bash
cd backend && go test ./internal/api -run 'TestCreateRoomWithMockupDrivenConfig|TestCreateRoomRejectsUnsupportedModeAndInvalidBuyInRange|TestTakeSeatRejectsBuyInOutsideRoomRange' -count=1
```

Expected: FAIL because room metadata is not accepted/returned and buy-in range is not validated.

- [ ] **Step 4: Implement storage room config types and migration**

In `backend/internal/storage/storage.go`, add fields:

```go
type RoomRecord struct {
	ID                string             `json:"id"`
	InviteCode        string             `json:"inviteCode"`
	OwnerUserID       string             `json:"ownerUserId"`
	Status            string             `json:"status"`
	RuleSetID         string             `json:"ruleSetId,omitempty"`
	Name              string             `json:"name,omitempty"`
	Mode              string             `json:"mode,omitempty"`
	Variant           string             `json:"variant,omitempty"`
	Ante              int                `json:"ante,omitempty"`
	MinBuyIn          int                `json:"minBuyIn,omitempty"`
	MaxBuyIn          int                `json:"maxBuyIn,omitempty"`
	BuyInCap          int                `json:"buyInCap,omitempty"`
	DurationMinutes   int                `json:"durationMinutes,omitempty"`
	Level             int                `json:"level,omitempty"`
	SeatCount         int                `json:"seatCount,omitempty"`
	MinPlayersToStart int                `json:"minPlayersToStart,omitempty"`
	Members           []RoomMemberRecord `json:"members"`
	Seats             []RoomSeatRecord   `json:"seats"`
	CurrentGameID     string             `json:"-"`
}

type CreateRoomOptions struct {
	RuleSetID         string
	Name              string
	Mode              string
	Variant           string
	Ante              int
	MinBuyIn          int
	MaxBuyIn          int
	BuyInCap          int
	DurationMinutes   int
	SeatCount         int
	MinPlayersToStart int
}
```

Add normalization helpers:

```go
func normalizeCreateRoomOptions(opt CreateRoomOptions) (CreateRoomOptions, error) {
	opt.RuleSetID = strings.TrimSpace(opt.RuleSetID)
	opt.Name = strings.TrimSpace(opt.Name)
	opt.Mode = strings.TrimSpace(opt.Mode)
	opt.Variant = strings.TrimSpace(opt.Variant)
	if opt.RuleSetID == "" {
		opt.RuleSetID = "short-deck"
	}
	if opt.Name == "" {
		opt.Name = "德扑之星"
	}
	if opt.Mode == "" {
		opt.Mode = "training"
	}
	if opt.Variant == "" {
		if opt.RuleSetID == "short-deck" {
			opt.Variant = "short_holdem"
		} else {
			opt.Variant = "holdem"
		}
	}
	if opt.SeatCount == 0 {
		opt.SeatCount = 6
	}
	if opt.MinPlayersToStart == 0 {
		opt.MinPlayersToStart = 2
	}
	if opt.Ante == 0 {
		opt.Ante = 20
	}
	if opt.MinBuyIn == 0 {
		opt.MinBuyIn = 2000
	}
	if opt.MaxBuyIn == 0 {
		opt.MaxBuyIn = 8000
	}
	if opt.BuyInCap == 0 {
		opt.BuyInCap = 60000
	}
	if opt.DurationMinutes == 0 {
		opt.DurationMinutes = 120
	}
	if opt.Mode != "training" {
		return opt, fieldError("unsupported_room_mode", "mode")
	}
	if opt.Variant != "short_holdem" && opt.Variant != "holdem" {
		return opt, fieldError("unsupported_variant", "variant")
	}
	if opt.SeatCount < 2 || opt.SeatCount > 9 {
		return opt, fieldError("invalid_room_config", "seatCount")
	}
	if opt.MinPlayersToStart < 2 || opt.MinPlayersToStart > opt.SeatCount {
		return opt, fieldError("invalid_room_config", "minPlayersToStart")
	}
	if opt.MinBuyIn <= 0 || opt.MaxBuyIn < opt.MinBuyIn {
		return opt, fieldError("invalid_room_config", "minBuyIn")
	}
	if opt.BuyInCap < opt.MaxBuyIn {
		return opt, fieldError("invalid_room_config", "buyInCap")
	}
	if opt.DurationMinutes <= 0 {
		return opt, fieldError("invalid_room_config", "durationMinutes")
	}
	return opt, nil
}
```

Add a small typed error near storage helpers:

```go
type FieldError struct {
	Code  string
	Field string
}

func (e FieldError) Error() string { return e.Code }

func fieldError(code, field string) FieldError {
	return FieldError{Code: code, Field: field}
}
```

Update `migrate()`:

```go
alterStmts := []string{
	`alter table rooms add column name varchar(128) not null default '德扑之星'`,
	`alter table rooms add column mode varchar(32) not null default 'training'`,
	`alter table rooms add column variant varchar(32) not null default 'short_holdem'`,
	`alter table rooms add column ante integer not null default 20`,
	`alter table rooms add column min_buy_in integer not null default 2000`,
	`alter table rooms add column max_buy_in integer not null default 8000`,
	`alter table rooms add column buy_in_cap integer not null default 60000`,
	`alter table rooms add column duration_minutes integer not null default 120`,
	`alter table rooms add column level integer not null default 1`,
}
for _, stmt := range alterStmts {
	if _, err := s.db.Exec(stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
}
```

Keep the existing `create table if not exists rooms` statement compatible by adding these columns to it as well.

- [ ] **Step 5: Replace `CreateRoom` with option-based implementation while preserving callers**

In `backend/internal/storage/storage.go`, replace the old `CreateRoom` body with:

```go
func (s *Store) CreateRoom(ownerUserID, ruleSetID string, seatCount, minPlayersToStart int) (*RoomRecord, error) {
	return s.CreateRoomWithOptions(ownerUserID, CreateRoomOptions{
		RuleSetID:         ruleSetID,
		SeatCount:         seatCount,
		MinPlayersToStart: minPlayersToStart,
	})
}

func (s *Store) CreateRoomWithOptions(ownerUserID string, opt CreateRoomOptions) (*RoomRecord, error) {
	opt, err := normalizeCreateRoomOptions(opt)
	if err != nil {
		return nil, err
	}
	roomID := fmt.Sprintf("room_%d", time.Now().UTC().UnixNano())
	inviteCode := fmt.Sprintf("R%06d", time.Now().UTC().UnixNano()%1000000)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	owner, err := s.FindUserByID(ownerUserID)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(
		`insert into rooms(id, invite_code, owner_user_id, status, rule_set_id, name, mode, variant, ante, min_buy_in, max_buy_in, buy_in_cap, duration_minutes, level, seat_count, min_players_to_start, created_at, updated_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roomID, inviteCode, ownerUserID, "waiting", opt.RuleSetID, opt.Name, opt.Mode, opt.Variant, opt.Ante, opt.MinBuyIn, opt.MaxBuyIn, opt.BuyInCap, opt.DurationMinutes, 1, opt.SeatCount, opt.MinPlayersToStart, now, now,
	)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`insert into room_members(room_id, user_id, role, joined_at) values(?, ?, ?, ?)`, roomID, ownerUserID, "owner", now); err != nil {
		return nil, err
	}
	for seatNo := 1; seatNo <= opt.SeatCount; seatNo++ {
		if _, err = tx.Exec(`insert into room_seats(room_id, seat_no, seat_status, updated_at) values(?, ?, ?, ?)`, roomID, seatNo, "empty", now); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	room, err := s.RoomByID(roomID)
	if err != nil {
		return nil, err
	}
	if len(room.Members) == 0 {
		room.Members = []RoomMemberRecord{{UserID: ownerUserID, Nickname: owner.Nickname, Role: "owner", JoinedAt: now}}
	}
	return room, nil
}
```

Update `RoomByID` query and scan to include the new columns.

- [ ] **Step 6: Enforce buy-in range in `TakeSeat`**

In `TakeSeat`, after loading wallet balance, load room min/max:

```go
var minBuyIn, maxBuyIn int
if err = tx.QueryRow(`select min_buy_in, max_buy_in from rooms where id = ?`, roomID).Scan(&minBuyIn, &maxBuyIn); err != nil {
	return nil, err
}
if buyInChips < minBuyIn || buyInChips > maxBuyIn {
	return nil, fieldError("invalid_buy_in", "buyInChips")
}
```

- [ ] **Step 7: Map storage field errors in HTTP API**

In `backend/internal/api/server.go`, extend create request:

```go
var req struct {
	RuleSetID         string `json:"ruleSetId"`
	Name              string `json:"name"`
	Mode              string `json:"mode"`
	Variant           string `json:"variant"`
	Ante              int    `json:"ante"`
	MinBuyIn          int    `json:"minBuyIn"`
	MaxBuyIn          int    `json:"maxBuyIn"`
	BuyInCap          int    `json:"buyInCap"`
	DurationMinutes   int    `json:"durationMinutes"`
	SeatCount         int    `json:"seatCount"`
	MinPlayersToStart int    `json:"minPlayersToStart"`
}
```

Call:

```go
room, err := s.store.CreateRoomWithOptions(user.ID, storage.CreateRoomOptions{
	RuleSetID:         req.RuleSetID,
	Name:              req.Name,
	Mode:              req.Mode,
	Variant:           req.Variant,
	Ante:              req.Ante,
	MinBuyIn:          req.MinBuyIn,
	MaxBuyIn:          req.MaxBuyIn,
	BuyInCap:          req.BuyInCap,
	DurationMinutes:   req.DurationMinutes,
	SeatCount:         req.SeatCount,
	MinPlayersToStart: req.MinPlayersToStart,
})
if err != nil {
	if fieldErr, ok := err.(storage.FieldError); ok {
		writeError(w, http.StatusBadRequest, fieldErr.Code, fieldErr.Code, fieldErr.Field)
		return
	}
	writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
	return
}
```

In seat handling, before generic storage error:

```go
if fieldErr, ok := err.(storage.FieldError); ok {
	writeError(w, http.StatusBadRequest, fieldErr.Code, fieldErr.Code, fieldErr.Field)
	return
}
```

- [ ] **Step 8: Run backend tests**

Run:

```bash
cd backend && go test ./internal/api ./internal/storage -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit backend room config**

```bash
git add backend/internal/storage/storage.go backend/internal/api/server.go backend/internal/api/room_multiplayer_test.go backend/internal/storage/storage_test.go
git commit -m "feat: add configurable room metadata"
```

---

## Task 2: Frontend Types, API, Routes, And Shared State

**Files:**
- Modify: `frontend/src/types/game.ts`
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/api/client.test.ts`
- Modify: `frontend/src/composables/useAppState.ts`
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: Write failing API client test for extended room creation**

Add to `frontend/src/api/client.test.ts` imports:

```ts
import { createRoom } from './client';
```

Add test:

```ts
it('sends mockup-driven room configuration when creating a room', async () => {
  const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => {
    expect(_url).toBe('/api/rooms');
    expect(init?.method).toBe('POST');
    expect((init?.headers as Record<string, string>).Authorization).toBe('Bearer tok_room');
    const body = JSON.parse(String(init?.body));
    expect(body).toEqual({
      ruleSetId: 'short-deck',
      name: '周末短牌局',
      mode: 'training',
      variant: 'short_holdem',
      ante: 20,
      minBuyIn: 2000,
      maxBuyIn: 8000,
      buyInCap: 60000,
      durationMinutes: 120,
      seatCount: 9,
      minPlayersToStart: 2
    });
    return new Response(JSON.stringify({
      id: 'room_1',
      inviteCode: 'R123456',
      ownerUserId: 'user_1',
      status: 'waiting',
      ruleSetId: body.ruleSetId,
      name: body.name,
      mode: body.mode,
      variant: body.variant,
      ante: body.ante,
      minBuyIn: body.minBuyIn,
      maxBuyIn: body.maxBuyIn,
      buyInCap: body.buyInCap,
      durationMinutes: body.durationMinutes,
      level: 1,
      seatCount: body.seatCount,
      minPlayersToStart: body.minPlayersToStart,
      members: [],
      seats: []
    }), { status: 201, headers: { 'Content-Type': 'application/json' } });
  });
  vi.stubGlobal('fetch', fetchMock);

  const room = await createRoom('tok_room', {
    ruleSetId: 'short-deck',
    name: '周末短牌局',
    mode: 'training',
    variant: 'short_holdem',
    ante: 20,
    minBuyIn: 2000,
    maxBuyIn: 8000,
    buyInCap: 60000,
    durationMinutes: 120,
    seatCount: 9,
    minPlayersToStart: 2
  });

  expect(room.name).toBe('周末短牌局');
  expect(room.minBuyIn).toBe(2000);
});
```

- [ ] **Step 2: Run API client test and confirm it fails**

Run:

```bash
cd frontend && npm test -- --run src/api/client.test.ts
```

Expected: FAIL because `createRoom` payload type does not include the new fields.

- [ ] **Step 3: Extend room types**

In `frontend/src/types/game.ts`, add:

```ts
export type RoomMode = 'training' | 'sng';
export type RoomVariant = 'short_holdem' | 'holdem' | 'omaha';

export interface CreateRoomPayload {
  ruleSetId: string;
  name?: string;
  mode?: RoomMode;
  variant?: RoomVariant;
  ante?: number;
  minBuyIn?: number;
  maxBuyIn?: number;
  buyInCap?: number;
  durationMinutes?: number;
  seatCount: number;
  minPlayersToStart: number;
}
```

Extend `RoomResponse`:

```ts
name?: string;
mode?: RoomMode;
variant?: RoomVariant;
ante?: number;
minBuyIn?: number;
maxBuyIn?: number;
buyInCap?: number;
durationMinutes?: number;
level?: number;
```

- [ ] **Step 4: Update API client and app state payload types**

In `frontend/src/api/client.ts`, import `CreateRoomPayload` and change:

```ts
export async function createRoom(token: string, payload: CreateRoomPayload): Promise<RoomResponse> {
  const res = await fetch(apiUrl('/api/rooms'), { method: 'POST', headers: authHeaders(token), body: JSON.stringify(payload) });
  return readJSON(res);
}
```

In `frontend/src/composables/useAppState.ts`, import `CreateRoomPayload` and change:

```ts
async function doCreateRoom(payload: CreateRoomPayload) {
```

Add shop return context:

```ts
const shopReturnTo = ref('');

function setShopReturnTo(path: string) {
  shopReturnTo.value = path;
}

function consumeShopReturnTo() {
  const path = shopReturnTo.value;
  shopReturnTo.value = '';
  return path;
}
```

Return `shopReturnTo`, `setShopReturnTo`, and `consumeShopReturnTo` from `useAppState()`.

- [ ] **Step 5: Add routes**

In `frontend/src/router/index.ts`, import:

```ts
import CreateMatchPage from '../pages/CreateMatchPage.vue';
import ShopPage from '../pages/ShopPage.vue';
```

Add routes:

```ts
{ path: '/create-match', component: CreateMatchPage },
{ path: '/shop', component: ShopPage },
```

- [ ] **Step 6: Run frontend targeted tests**

Run:

```bash
cd frontend && npm test -- --run src/api/client.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit frontend API foundation**

```bash
git add frontend/src/types/game.ts frontend/src/api/client.ts frontend/src/api/client.test.ts frontend/src/composables/useAppState.ts frontend/src/router/index.ts
git commit -m "feat: add room config frontend contracts"
```

---

## Task 3: Create Match Page

**Files:**
- Create: `frontend/src/pages/CreateMatchPage.vue`
- Modify: `frontend/src/pages/LobbyPage.vue`
- Modify: `frontend/src/multiplayerTable.test.ts`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Write failing source-contract test**

Add to `frontend/src/multiplayerTable.test.ts`:

```ts
import createMatchSource from './pages/CreateMatchPage.vue?raw';
```

Add test:

```ts
it('provides a mockup-driven create match flow', () => {
  [
    '创建比赛',
    '训练赛',
    'SNG',
    '国际扑克',
    '短牌',
    '奥马哈',
    '比赛牌局名字',
    'Ante设置',
    '单次最小带入',
    '带入记分牌上限',
    '训练时长',
    '单桌最大人数',
    '确 定 创 建',
    'doCreateRoom',
    "variant: 'short_holdem'",
    "mode: 'training'",
    "router.push(`/room/${created.id}`)"
  ].forEach((token) => expect(createMatchSource).toContain(token));

  expect(lobbySource).toContain("router.push('/create-match')");
});
```

- [ ] **Step 2: Run test and confirm it fails**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: FAIL because `CreateMatchPage.vue` does not exist.

- [ ] **Step 3: Create `CreateMatchPage.vue`**

Create `frontend/src/pages/CreateMatchPage.vue`:

```vue
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAppState } from '../composables/useAppState';
import type { RoomVariant } from '../types/game';

const router = useRouter();
const { doCreateRoom, loading, error, clearError } = useAppState();

const matchName = ref('德扑之星');
const mode = ref<'training' | 'sng'>('training');
const variant = ref<RoomVariant>('short_holdem');
const ante = ref(20);
const minBuyIn = ref(2000);
const maxBuyIn = ref(8000);
const buyInCap = ref(60000);
const durationHours = ref(2);
const seatCount = ref(9);

const anteOptions = [10, 20, 30, 50, 100, 200, 300, 500];
const minBuyInOptions = [2000, 4000, 6000, 8000];
const buyInCapOptions = [0, 16000, 24000, 32000, 40000, 60000];
const durationOptions = [0.5, 1, 1.5, 2, 3, 4, 5, 6];
const seatOptions = [2, 3, 4, 5, 6, 7, 8, 9];

const canCreate = computed(() => mode.value === 'training' && (variant.value === 'short_holdem' || variant.value === 'holdem'));
const buyInCapLabel = computed(() => buyInCap.value === 0 ? '无限制' : `${Math.floor(buyInCap.value / 1000)}K`);

function setVariant(next: RoomVariant) {
  variant.value = next;
  if (next === 'short_holdem') {
    ante.value = 20;
  }
}

async function createNow() {
  clearError();
  if (!canCreate.value) return;
  const created = await doCreateRoom({
    ruleSetId: variant.value === 'short_holdem' ? 'short-deck' : 'long-holdem',
    name: matchName.value.trim() || '德扑之星',
    mode: mode.value,
    variant: variant.value,
    ante: ante.value,
    minBuyIn: minBuyIn.value,
    maxBuyIn: maxBuyIn.value,
    buyInCap: buyInCap.value,
    durationMinutes: Math.round(durationHours.value * 60),
    seatCount: seatCount.value,
    minPlayersToStart: 2
  });
  if (created) router.push(`/room/${created.id}`);
}
</script>

<template>
  <main class="page-shell create-match-shell">
    <section class="create-match-panel">
      <header class="mobile-titlebar">
        <button type="button" class="icon-text-button" @click="router.back()">返回</button>
        <h1>创建比赛</h1>
      </header>

      <div class="mode-tabs" aria-label="比赛类型">
        <button type="button" :class="{ active: mode === 'training' }" @click="mode = 'training'">训练赛</button>
        <button type="button" :class="{ active: mode === 'sng' }" @click="mode = 'sng'">SNG</button>
      </div>

      <div class="segmented-pill" aria-label="玩法">
        <button type="button" :class="{ active: variant === 'holdem' }" @click="setVariant('holdem')">国际扑克</button>
        <button type="button" :class="{ active: variant === 'short_holdem' }" @click="setVariant('short_holdem')">短牌</button>
        <button type="button" :class="{ active: variant === 'omaha' }" @click="setVariant('omaha')">奥马哈</button>
      </div>

      <label class="form-row">比赛牌局名字:
        <input v-model="matchName" maxlength="24" placeholder="请输入比赛名字" />
      </label>

      <section class="range-card">
        <h2>Ante设置 <strong>{{ ante }}</strong></h2>
        <div class="choice-row">
          <button v-for="item in anteOptions" :key="item" type="button" :class="{ active: ante === item }" @click="ante = item">{{ item }}</button>
        </div>
      </section>

      <section class="range-card">
        <h2>单次最小带入 <strong>{{ minBuyIn.toLocaleString('zh-CN') }}</strong></h2>
        <div class="choice-row">
          <button v-for="item in minBuyInOptions" :key="item" type="button" :class="{ active: minBuyIn === item }" @click="minBuyIn = item">{{ Math.floor(item / 1000) }}K</button>
        </div>
      </section>

      <section class="range-card">
        <h2>带入记分牌上限 <strong>{{ buyInCapLabel }}</strong></h2>
        <div class="choice-row">
          <button v-for="item in buyInCapOptions" :key="item" type="button" :class="{ active: buyInCap === item }" @click="buyInCap = item">{{ item === 0 ? '无限制' : `${Math.floor(item / 1000)}K` }}</button>
        </div>
      </section>

      <section class="range-card">
        <h2>训练时长 <strong>{{ durationHours }}h</strong></h2>
        <div class="choice-row">
          <button v-for="item in durationOptions" :key="item" type="button" :class="{ active: durationHours === item }" @click="durationHours = item">{{ item }}</button>
        </div>
      </section>

      <section class="range-card">
        <h2>单桌最大人数 <strong>{{ seatCount }}人</strong></h2>
        <div class="choice-row">
          <button v-for="item in seatOptions" :key="item" type="button" :class="{ active: seatCount === item }" @click="seatCount = item">{{ item }}</button>
        </div>
      </section>

      <p v-if="mode === 'sng' || variant === 'omaha'" class="error">该模式暂未开放</p>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="button" class="create-submit-button" :disabled="loading || !canCreate" @click="createNow">确 定 创 建</button>
    </section>
  </main>
</template>
```

- [ ] **Step 4: Update lobby entry**

In `frontend/src/pages/LobbyPage.vue`, change the create-room article button to route:

```vue
<button type="button" @click="router.push('/create-match')">创建比赛</button>
```

Keep the existing join-room form.

- [ ] **Step 5: Add styles**

Append to `frontend/src/style.css`:

```css
.create-match-shell {
  min-height: 100dvh;
  background: #101c26;
  color: #cfe8ff;
}

.create-match-panel {
  width: min(100%, 430px);
  min-height: 100dvh;
  margin: 0 auto;
  padding: 18px 10px 96px;
}

.mobile-titlebar {
  display: grid;
  grid-template-columns: 72px 1fr 72px;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
}

.mobile-titlebar h1 {
  margin: 0;
  text-align: center;
  font-size: 30px;
  letter-spacing: 0;
}

.mode-tabs,
.segmented-pill,
.choice-row {
  display: grid;
  gap: 8px;
}

.mode-tabs {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 16px;
}

.segmented-pill {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  padding: 6px;
  border-radius: 28px;
  background: #1c3a5c;
  margin-bottom: 14px;
}

.mode-tabs button,
.segmented-pill button,
.choice-row button {
  min-height: 42px;
  border: 0;
  border-radius: 8px;
  color: #90c9ff;
  background: transparent;
  font-weight: 700;
}

.mode-tabs button.active,
.segmented-pill button.active,
.choice-row button.active,
.create-submit-button {
  color: #081824;
  background: linear-gradient(90deg, #edb64f, #ffe0a4);
}

.form-row,
.range-card {
  display: block;
  margin: 12px 0;
  padding: 16px;
  border-radius: 8px;
  background: #1a3554;
}

.form-row input {
  width: 100%;
  margin-top: 10px;
  border: 0;
  border-radius: 24px;
  padding: 12px 16px;
  color: #cfe8ff;
  background: #122740;
}

.range-card h2 {
  margin: 0 0 12px;
  font-size: 18px;
}

.range-card h2 strong {
  margin-left: 8px;
  color: #67bfff;
}

.choice-row {
  grid-template-columns: repeat(auto-fit, minmax(56px, 1fr));
}

.create-submit-button {
  position: fixed;
  left: 50%;
  bottom: 16px;
  width: min(390px, calc(100vw - 28px));
  transform: translateX(-50%);
  min-height: 56px;
  border: 0;
  border-radius: 28px;
  font-size: 22px;
  font-weight: 800;
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit create match page**

```bash
git add frontend/src/pages/CreateMatchPage.vue frontend/src/pages/LobbyPage.vue frontend/src/multiplayerTable.test.ts frontend/src/style.css
git commit -m "feat: add create match page"
```

---

## Task 4: Gold Shop Page

**Files:**
- Create: `frontend/src/pages/ShopPage.vue`
- Modify: `frontend/src/pages/MePage.vue`
- Modify: `frontend/src/multiplayerTable.test.ts`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Write failing shop contract test**

In `frontend/src/multiplayerTable.test.ts`, import:

```ts
import shopSource from './pages/ShopPage.vue?raw';
```

Add test:

```ts
it('provides a gold shop backed by simulated recharge', () => {
  [
    '商城',
    '金币',
    '钻石',
    'VIP卡',
    '装扮',
    '道具',
    '课程',
    '普通卡',
    'fetchRechargeOptions',
    'doRecharge',
    'simulateRecharge',
    'consumeShopReturnTo',
    'router.push(returnPath || \\'/me\\')'
  ].forEach((token) => expect(shopSource).toContain(token));

  expect(meSource).toContain("router.push('/shop')");
});
```

- [ ] **Step 2: Run test and confirm it fails**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: FAIL because `ShopPage.vue` does not exist.

- [ ] **Step 3: Create `ShopPage.vue`**

Create `frontend/src/pages/ShopPage.vue`:

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { fetchRechargeOptions } from '../api/client';
import { useAppState } from '../composables/useAppState';
import type { RechargeOption } from '../types/game';

const router = useRouter();
const { wallet, me, loading, refreshProfile, doRecharge, consumeShopReturnTo } = useAppState();
const activeTab = ref('金币');
const rechargeOptions = ref<RechargeOption[]>([]);
const message = ref('');
const returnPath = ref('');
const tabs = ['金币', '钻石', 'VIP卡', '装扮', '道具', '课程'];

const bonusByCode: Record<string, number> = {
  small: 28,
  medium: 218,
  large: 618
};

function priceLabel(option: RechargeOption) {
  if (option.code === 'small') return '¥ 6';
  if (option.code === 'medium') return '¥ 30';
  if (option.code === 'large') return '¥ 68';
  return '模拟充值';
}

async function simulateRecharge(option: RechargeOption) {
  const confirmed = window.confirm(`确认模拟充值 ${option.label}？`);
  if (!confirmed) return;
  await doRecharge(option.code);
  message.value = `充值成功：+${option.amount} 金币`;
}

function goBack() {
  router.push(returnPath.value || '/me');
}

onMounted(async () => {
  returnPath.value = consumeShopReturnTo();
  await refreshProfile();
  rechargeOptions.value = (await fetchRechargeOptions()).options;
});
</script>

<template>
  <main class="page-shell shop-shell">
    <section class="shop-panel">
      <header class="mobile-titlebar">
        <button type="button" class="icon-text-button" @click="goBack">返回</button>
        <h1>商城</h1>
      </header>

      <nav class="shop-tabs" aria-label="商城分类">
        <button v-for="tab in tabs" :key="tab" type="button" :class="{ active: activeTab === tab }" @click="activeTab = tab">{{ tab }}</button>
      </nav>

      <section class="asset-strip">
        <div><strong>金币</strong><span>{{ wallet?.balance ?? me?.walletBalance ?? 0 }}</span></div>
        <div><strong>钻石</strong><span>0</span></div>
        <div><strong>积分</strong><span>0</span></div>
        <div><strong>普通卡</strong><span>未激活</span></div>
      </section>

      <section v-if="activeTab === '金币'" class="shop-grid">
        <button v-for="option in rechargeOptions" :key="option.code" type="button" class="shop-product" :disabled="loading" @click="simulateRecharge(option)">
          <span class="product-icon">♠</span>
          <strong>{{ option.amount }}金币</strong>
          <span>额外赠送{{ bonusByCode[option.code] ?? 0 }}金币</span>
          <em>{{ priceLabel(option) }}</em>
        </button>
      </section>

      <section v-else class="shop-placeholder">
        <strong>{{ activeTab }}</strong>
        <span>该分类暂未开放</span>
      </section>

      <p v-if="message" class="success-text">{{ message }}</p>
    </section>
  </main>
</template>
```

- [ ] **Step 4: Enable shop entry from profile**

In `frontend/src/pages/MePage.vue`, change disabled shop button:

```vue
<button type="button" @click="router.push('/shop')">商城</button>
```

- [ ] **Step 5: Add shop styles**

Append to `frontend/src/style.css`:

```css
.shop-shell {
  min-height: 100dvh;
  background: #101c26;
  color: #d7e9ff;
}

.shop-panel {
  width: min(100%, 430px);
  margin: 0 auto;
  padding: 18px 10px 28px;
}

.shop-tabs {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 4px;
  margin-bottom: 18px;
}

.shop-tabs button {
  min-height: 42px;
  border: 0;
  border-bottom: 3px solid transparent;
  color: #d7e9ff;
  background: transparent;
  font-weight: 800;
}

.shop-tabs button.active {
  color: #ffd179;
  border-bottom-color: #ffd179;
}

.asset-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  padding: 18px 12px;
  border-radius: 8px;
  background: #31ad73;
  color: white;
}

.asset-strip div,
.shop-product {
  display: grid;
  place-items: center;
  gap: 6px;
  text-align: center;
}

.shop-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 18px;
}

.shop-product {
  min-height: 146px;
  border: 0;
  border-radius: 8px;
  overflow: hidden;
  color: #cfe8ff;
  background: #1a3554;
}

.shop-product em {
  width: 100%;
  padding: 10px;
  color: #55ff65;
  background: #38669b;
  font-style: normal;
  font-weight: 800;
}

.product-icon {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 8px;
  color: #1a3554;
  background: white;
}

.shop-placeholder {
  display: grid;
  min-height: 240px;
  place-items: center;
  margin-top: 18px;
  border-radius: 8px;
  background: #1a3554;
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit shop page**

```bash
git add frontend/src/pages/ShopPage.vue frontend/src/pages/MePage.vue frontend/src/multiplayerTable.test.ts frontend/src/style.css
git commit -m "feat: add gold shop page"
```

---

## Task 5: Buy-In Modal And Table Seat Flow

**Files:**
- Create: `frontend/src/components/BuyInModal.vue`
- Modify: `frontend/src/pages/RoomPage.vue`
- Modify: `frontend/src/multiplayerTable.test.ts`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Write failing modal contract test**

In `frontend/src/multiplayerTable.test.ts`, import:

```ts
import buyInModalSource from './components/BuyInModal.vue?raw';
```

Add test:

```ts
it('opens a table buy-in modal before taking a seat', () => {
  [
    '补充记分牌',
    '在下一手开始前，为您补充记分牌',
    '带入记分牌',
    '消耗',
    '可用',
    '前往商城购买金币',
    'emit(\\'confirm\\'',
    'emit(\\'shop\\')'
  ].forEach((token) => expect(buyInModalSource).toContain(token));

  [
    'BuyInModal',
    'pendingSeatNo',
    'openBuyInModal',
    'confirmBuyIn',
    'setShopReturnTo',
    "router.push('/shop')"
  ].forEach((token) => expect(roomSource).toContain(token));
});
```

- [ ] **Step 2: Run test and confirm it fails**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: FAIL because component and room integration do not exist.

- [ ] **Step 3: Create `BuyInModal.vue`**

Create `frontend/src/components/BuyInModal.vue`:

```vue
<script setup lang="ts">
import { computed, ref, watch } from 'vue';

const props = defineProps<{
  open: boolean;
  min: number;
  max: number;
  walletBalance: number;
}>();

const emit = defineEmits<{
  close: [];
  confirm: [amount: number];
  shop: [];
}>();

const amount = ref(props.min);
const options = computed(() => {
  const min = props.min || 2000;
  const max = props.max || 6000;
  const values = [min, min + 1000, min + 2000, min + 3000, max]
    .filter((value, index, array) => value <= max && array.indexOf(value) === index);
  return values.length ? values : [2000, 3000, 4000, 5000, 6000];
});
const insufficient = computed(() => amount.value > props.walletBalance);

watch(() => props.open, (open) => {
  if (open) amount.value = props.min || 2000;
});
</script>

<template>
  <div v-if="open" class="modal-backdrop">
    <section class="buy-in-modal" role="dialog" aria-modal="true" aria-label="补充记分牌">
      <h2>补充记分牌</h2>
      <p>在下一手开始前，为您补充记分牌</p>
      <strong class="buy-in-amount">{{ amount.toLocaleString('zh-CN') }}</strong>
      <span class="buy-in-label">带入记分牌</span>

      <div class="buy-in-options">
        <button v-for="option in options" :key="option" type="button" :class="{ active: amount === option }" @click="amount = option">
          {{ Math.floor(option / 1000) }}K
        </button>
      </div>

      <div class="buy-in-balance">
        <span>消耗 <strong>{{ amount }}</strong></span>
        <span>可用 <strong>{{ walletBalance }}</strong></span>
      </div>

      <button type="button" class="shop-link-button" @click="emit('shop')">前往商城购买金币 &gt;</button>

      <p v-if="insufficient" class="error">金币不足，请先购买金币</p>

      <div class="modal-actions">
        <button type="button" class="ghost" @click="emit('close')">取消</button>
        <button type="button" :disabled="insufficient" @click="emit('confirm', amount)">确定</button>
      </div>
    </section>
  </div>
</template>
```

- [ ] **Step 4: Integrate modal in `RoomPage.vue`**

Import:

```ts
import BuyInModal from '../components/BuyInModal.vue';
```

Destructure `wallet` and `setShopReturnTo` from `useAppState()`.

Add state and helpers:

```ts
const pendingSeatNo = ref<number | null>(null);
const buyInModalOpen = computed(() => pendingSeatNo.value !== null);
const roomMinBuyIn = computed(() => room.value?.minBuyIn ?? defaultBuyIn.value ?? 2000);
const roomMaxBuyIn = computed(() => room.value?.maxBuyIn ?? Math.max(roomMinBuyIn.value, 6000));
const walletBalance = computed(() => wallet.value?.balance ?? me.value?.walletBalance ?? 0);

function openBuyInModal(seatNo: number) {
  if (roomSeat(seatNo)?.userId) return;
  pendingSeatNo.value = seatNo;
}

function closeBuyInModal() {
  pendingSeatNo.value = null;
}

async function confirmBuyIn(amount: number) {
  if (!pendingSeatNo.value) return;
  await doTakeSeat(pendingSeatNo.value, amount);
  closeBuyInModal();
  await refreshRoom();
  await refreshCurrentRoomHand();
}

function goToShopFromBuyIn() {
  setShopReturnTo(router.currentRoute.value.fullPath);
  router.push('/shop');
}
```

Change empty seat click from players route to modal:

```vue
@click="roomSeat(seatNo)?.userId ? router.push(`/room/${room.id}/players`) : openBuyInModal(seatNo)"
```

Add near the end of the room section:

```vue
<BuyInModal
  :open="buyInModalOpen"
  :min="roomMinBuyIn"
  :max="roomMaxBuyIn"
  :wallet-balance="walletBalance"
  @close="closeBuyInModal"
  @confirm="confirmBuyIn"
  @shop="goToShopFromBuyIn"
/>
```

- [ ] **Step 5: Add modal styles**

Append to `frontend/src/style.css`:

```css
.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 18px;
  background: rgba(0, 0, 0, 0.46);
}

.buy-in-modal {
  width: min(360px, 100%);
  padding: 24px;
  border-radius: 8px;
  color: #34475d;
  background: #dde7f2;
  text-align: center;
}

.buy-in-modal h2 {
  margin: 0 0 12px;
  font-size: 26px;
}

.buy-in-amount {
  display: block;
  margin-top: 18px;
  color: #ff594b;
  font-size: 42px;
}

.buy-in-options {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 4px;
  margin: 18px 0;
  padding: 10px;
  border-radius: 8px;
  background: #c9d6e4;
}

.buy-in-options button {
  min-height: 42px;
  border: 0;
  color: #34475d;
  background: transparent;
  font-weight: 800;
}

.buy-in-options button.active {
  color: white;
  border-radius: 20px;
  background: #2a9fea;
}

.buy-in-balance,
.modal-actions {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.shop-link-button {
  margin: 18px 0;
  border: 0;
  color: #48ad61;
  background: transparent;
  font-size: 18px;
}

.modal-actions button {
  flex: 1;
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit buy-in modal**

```bash
git add frontend/src/components/BuyInModal.vue frontend/src/pages/RoomPage.vue frontend/src/multiplayerTable.test.ts frontend/src/style.css
git commit -m "feat: add table buy-in modal"
```

---

## Task 6: Mobile Table Stage

**Files:**
- Modify: `frontend/src/pages/RoomPage.vue`
- Modify: `frontend/src/multiplayerTable.test.ts`
- Modify: `frontend/src/App.visual.test.ts`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Update failing table-surface tests**

In `frontend/src/multiplayerTable.test.ts`, replace old expectations for side-panel table tools with tokens:

```ts
it('renders the mockup-driven mobile table shell', () => {
  [
    '绿色竞技 远离赌博',
    '网络延时',
    'mock-table-felt',
    'table-room-center',
    '解散比赛',
    '邀请好友',
    '开 始',
    '底部工具栏',
    'tool-settings',
    'tool-chat',
    'tool-mic',
    'openPanel(\\'settings\\')',
    'openPanel(\\'chat\\')',
    'openPanel(\\'score\\')',
    'openPanel(\\'replay\\')'
  ].forEach((token) => expect(roomSource).toContain(token));

  expect(roomSource).not.toContain('v11-table-tools');
});
```

- [ ] **Step 2: Run tests and confirm they fail**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts src/App.visual.test.ts
```

Expected: FAIL because the room page still uses the old side-panel layout.

- [ ] **Step 3: Refactor `RoomPage.vue` state for table panels and owner controls**

Add:

```ts
const activePanel = ref<'chat' | 'score' | 'replay' | 'settings' | null>(null);
const isRoomOwner = computed(() => !!room.value && room.value.ownerUserId === me.value?.id);
const tableLatencyMs = ref(43);

function openPanel(panel: 'chat' | 'score' | 'replay' | 'settings') {
  activePanel.value = panel;
}

function closePanel() {
  activePanel.value = null;
}

async function startHandFromTable() {
  await doStartRoomHand();
}

function roomVariantLabel() {
  if (room.value?.variant === 'short_holdem' || room.value?.ruleSetId === 'short-deck') return '短牌';
  if (room.value?.variant === 'omaha') return '奥马哈';
  return '国际扑克';
}
```

Destructure `doStartRoomHand` from app state if not already present.

- [ ] **Step 4: Replace old template shell**

In `RoomPage.vue`, replace the topbar/side-panel structure with this shape while preserving existing card/action rendering inside the table:

```vue
<main class="room-mobile-screen" v-if="room">
  <section class="mock-table-felt" aria-label="正式多人牌桌">
    <div class="responsible-gaming">绿色竞技 远离赌博 · 谨防诈骗 健康生活</div>
    <div class="latency-badge">网络延时<br><strong>{{ tableLatencyMs }}ms</strong></div>

    <div class="seat-ring seat-ring-casino" v-if="room.seatCount">
      <button
        v-for="seatNo in room.seatCount"
        :key="seatNo"
        type="button"
        class="mock-seat"
        :class="{ mine: myRoomSeat?.seatNo === seatNo, acting: currentRoomHand?.currentSeat === seatNo, empty: !roomSeat(seatNo)?.userId }"
        @click="roomSeat(seatNo)?.userId ? router.push(`/room/${room.id}/players`) : openBuyInModal(seatNo)"
      >
        <span>{{ roomSeat(seatNo)?.nickname || (myRoomSeat ? '坐下' : '空座') }}</span>
        <small v-if="seatPlayer(seatNo)">{{ seatPlayer(seatNo)?.stack }}</small>
      </button>
    </div>

    <div class="owner-action-row" v-if="!currentRoomHand">
      <button type="button" :disabled="!isRoomOwner">解散比赛</button>
      <button type="button">邀请好友</button>
      <button type="button" :disabled="!isRoomOwner || loading" @click="startHandFromTable">开 始</button>
    </div>

    <section class="table-room-center">
      <span>训练赛◆{{ roomVariantLabel() }} 第 {{ recentRoomHands[0]?.handNo ? recentRoomHands[0].handNo + 1 : 1 }} 手</span>
      <strong>{{ room.name || '德扑之星' }}</strong>
      <span>◎ {{ room.ante ?? 20 }}　级别:{{ room.level ?? 1 }}</span>
      <span>&lt; {{ room.id.replace('room_', '').slice(-3) || '121' }} &gt;</span>
      <span>邀请码:{{ room.inviteCode || '加载中' }}</span>
    </section>

    <div class="board-zone clean-board-zone table-center-stack" v-if="currentRoomHand">
      <!-- keep existing community cards and pot markup here -->
    </div>

    <div class="hero-panel hero-panel-clean mock-hero-panel" v-if="myRoomSeat">
      <!-- keep existing hero cards and hero meta markup here -->
    </div>

    <div class="mock-bottom-toolbar" aria-label="底部工具栏">
      <button type="button" class="tool-settings" @click="openPanel('settings')">▦</button>
      <button type="button" @click="leaveTable">↩</button>
      <button type="button" class="tool-score" @click="openPanel('score')">战绩</button>
      <button type="button" class="tool-replay" @click="openPanel('replay')">牌谱</button>
      <button type="button" class="tool-chat" @click="openPanel('chat')">▤</button>
      <button type="button" class="tool-mic" disabled>麦克风关</button>
    </div>

    <!-- keep existing action buttons dock when currentRoomHand exists -->
    <BuyInModal ... />
  </section>
</main>
```

Do not remove `remainingActionSeconds`, `hero-actions-dock`, `doRoomAction`, or card display logic.

- [ ] **Step 5: Add mobile table styles**

Append to `frontend/src/style.css`:

```css
.room-mobile-screen {
  min-height: 100dvh;
  margin: 0;
  padding: 0;
  overflow: hidden;
  color: #cfe8ff;
  background: #082016;
}

.mock-table-felt {
  position: relative;
  width: min(100vw, 430px);
  min-height: 100dvh;
  margin: 0 auto;
  overflow: hidden;
  background:
    radial-gradient(circle at 50% 42%, rgba(72, 152, 121, 0.44), transparent 34%),
    linear-gradient(160deg, #0d3c2b, #0b241d 72%);
}

.responsible-gaming {
  position: absolute;
  top: 8px;
  left: 50%;
  z-index: 2;
  width: max-content;
  max-width: calc(100% - 80px);
  transform: translateX(-50%);
  color: rgba(10, 22, 17, 0.8);
  font-weight: 800;
}

.latency-badge {
  position: absolute;
  top: 58px;
  right: 0;
  z-index: 2;
  padding: 8px 12px;
  border-radius: 6px 0 0 6px;
  color: #89a39c;
  background: rgba(0, 0, 0, 0.74);
  text-align: center;
  font-size: 12px;
}

.latency-badge strong {
  color: #4bff76;
}

.mock-seat {
  position: absolute;
  display: grid;
  width: 62px;
  height: 62px;
  place-items: center;
  border: 2px solid rgba(201, 232, 228, 0.36);
  border-radius: 8px;
  color: rgba(211, 238, 245, 0.72);
  background: rgba(7, 40, 31, 0.38);
  font-weight: 800;
}

.mock-seat:nth-child(1) { left: 44%; top: 8%; }
.mock-seat:nth-child(2) { right: 27%; top: 8%; }
.mock-seat:nth-child(3) { right: 4%; top: 22%; }
.mock-seat:nth-child(4) { right: 4%; top: 41%; }
.mock-seat:nth-child(5) { right: 4%; top: 60%; }
.mock-seat:nth-child(6) { left: 44%; bottom: 9%; }
.mock-seat:nth-child(7) { left: 4%; top: 60%; }
.mock-seat:nth-child(8) { left: 4%; top: 41%; }
.mock-seat:nth-child(9) { left: 4%; top: 22%; }

.owner-action-row {
  position: absolute;
  top: 34%;
  left: 50%;
  z-index: 3;
  display: flex;
  gap: 8px;
  transform: translateX(-50%);
}

.owner-action-row button {
  min-width: 88px;
  min-height: 42px;
  border: 2px solid #f8da91;
  border-radius: 6px;
  color: white;
  background: #c79c49;
  font-weight: 900;
}

.table-room-center {
  position: absolute;
  top: 46%;
  left: 50%;
  z-index: 1;
  display: grid;
  width: 260px;
  gap: 6px;
  transform: translate(-50%, -50%);
  color: rgba(2, 35, 26, 0.72);
  text-align: center;
  font-weight: 800;
}

.table-room-center strong {
  font-size: 34px;
}

.mock-bottom-toolbar {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 5;
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  align-items: end;
  gap: 8px;
  padding: 14px 12px 20px;
  background: linear-gradient(0deg, rgba(5, 22, 19, 0.86), rgba(5, 22, 19, 0.12));
}

.mock-bottom-toolbar button {
  min-height: 48px;
  border: 0;
  color: #d6b067;
  background: transparent;
  font-weight: 900;
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts src/App.visual.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit mobile table stage**

```bash
git add frontend/src/pages/RoomPage.vue frontend/src/multiplayerTable.test.ts frontend/src/App.visual.test.ts frontend/src/style.css
git commit -m "feat: add mobile table stage"
```

---

## Task 7: Table Drawers And Panels

**Files:**
- Create: `frontend/src/components/TableDrawer.vue`
- Create: `frontend/src/components/TableChatPanel.vue`
- Create: `frontend/src/components/TableScorePanel.vue`
- Create: `frontend/src/components/TableReplayPanel.vue`
- Create: `frontend/src/components/TableSettingsPanel.vue`
- Modify: `frontend/src/pages/RoomPage.vue`
- Modify: `frontend/src/multiplayerTable.test.ts`
- Modify: `frontend/src/style.css`

- [ ] **Step 1: Write failing panel contract tests**

In `frontend/src/multiplayerTable.test.ts`, import panel sources and add:

```ts
import tableChatPanelSource from './components/TableChatPanel.vue?raw';
import tableScorePanelSource from './components/TableScorePanel.vue?raw';
import tableReplayPanelSource from './components/TableReplayPanel.vue?raw';
import tableSettingsPanelSource from './components/TableSettingsPanel.vue?raw';

it('provides in-table chat score replay and settings panels', () => {
  ['常用语', '聊天记录', '请输入聊天内容，上限40个汉字', '发送', 'sendRoomChat'].forEach((token) => expect(tableChatPanelSource).toContain(token));
  ['当前战绩', '剩余时间', '昵称', '带入', '手数', '战绩', '观众'].forEach((token) => expect(tableScorePanelSource).toContain(token));
  ['牌谱回顾', '回放', '暂无任何数据', '收藏', '投诉', 'fetchRoomHandReplay'].forEach((token) => expect(tableReplayPanelSource).toContain(token));
  ['桌面设置', '站起围观', '带入记分牌', '比赛设置', '短牌规则', '保位离座', '降落伞说明', '退出比赛'].forEach((token) => expect(tableSettingsPanelSource).toContain(token));
});
```

- [ ] **Step 2: Run tests and confirm they fail**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: FAIL because components do not exist.

- [ ] **Step 3: Create `TableDrawer.vue`**

Create `frontend/src/components/TableDrawer.vue`:

```vue
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
```

- [ ] **Step 4: Create `TableChatPanel.vue`**

Create component with props for `messages`, `loading`, emits `send`, and local 40-char input. Include exact placeholder `请输入聊天内容，上限40个汉字`.

```vue
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
```

- [ ] **Step 5: Create score, replay, and settings panels**

Create `TableScorePanel.vue`:

```vue
<script setup lang="ts">
import type { RoomLeaderboardItem, RoomMember, RoomSeat } from '../types/game';

defineProps<{
  leaderboard: RoomLeaderboardItem[];
  members: RoomMember[];
  seats: RoomSeat[];
  durationMinutes?: number;
}>();

function seatBuyIn(userId: string, seats: RoomSeat[]) {
  return seats.find((seat) => seat.userId === userId)?.buyInChips ?? 0;
}
</script>

<template>
  <section class="table-score-panel">
    <header><strong>当前战绩</strong><span>剩余时间 {{ Math.floor((durationMinutes ?? 120) / 60).toString().padStart(2, '0') }}:00:00</span></header>
    <table>
      <thead><tr><th>昵称</th><th>带入</th><th>手数</th><th>战绩</th></tr></thead>
      <tbody>
        <tr v-for="item in leaderboard" :key="item.userId">
          <td>{{ item.nickname }}</td>
          <td>{{ seatBuyIn(item.userId, seats) || '-' }}</td>
          <td>{{ item.handsPlayed }}</td>
          <td>{{ item.netProfit >= 0 ? '+' : '' }}{{ item.netProfit }}</td>
        </tr>
      </tbody>
    </table>
    <p v-if="!leaderboard.length">观众({{ members.filter(member => !seats.some(seat => seat.userId === member.userId)).length }}/{{ members.length }})</p>
  </section>
</template>
```

Create `TableReplayPanel.vue` with `fetchRoomHandReplay`, `暂无任何数据`, `收藏`, and `投诉`; use the existing `recentRoomHands` list and emit route fallback for full replay if needed.

Create `TableSettingsPanel.vue`:

```vue
<script setup lang="ts">
const emit = defineEmits<{
  stand: [];
  buyIn: [];
  leave: [];
}>();
</script>

<template>
  <section class="table-settings-grid">
    <button type="button">桌面设置</button>
    <button type="button" @click="emit('stand')">站起围观</button>
    <button type="button" @click="emit('buyIn')">带入记分牌</button>
    <button type="button">比赛设置</button>
    <button type="button">短牌规则</button>
    <button type="button" disabled>保位离座</button>
    <button type="button" disabled>降落伞说明</button>
    <button type="button" @click="emit('leave')">退出比赛</button>
  </section>
</template>
```

- [ ] **Step 6: Wire panels into `RoomPage.vue`**

Import all components. Add after toolbar:

```vue
<TableDrawer :open="activePanel === 'chat'" placement="bottom" @close="closePanel">
  <TableChatPanel :messages="chatMessages" :loading="loading" @send="sendRoomChat" />
</TableDrawer>

<TableDrawer :open="activePanel === 'score'" placement="left" @close="closePanel">
  <TableScorePanel :leaderboard="roomLeaderboard" :members="room.members" :seats="room.seats" :duration-minutes="room.durationMinutes" />
</TableDrawer>

<TableDrawer :open="activePanel === 'replay'" placement="right" @close="closePanel">
  <TableReplayPanel :room-id="room.id" :hands="recentRoomHands" :token="token" />
</TableDrawer>

<TableDrawer :open="activePanel === 'settings'" placement="bottom" @close="closePanel">
  <TableSettingsPanel @stand="myRoomSeat && doLeaveSeat(myRoomSeat.seatNo)" @buy-in="myRoomSeat ? openBuyInModal(myRoomSeat.seatNo) : firstOpenSeatNo && openBuyInModal(firstOpenSeatNo)" @leave="leaveTable" />
</TableDrawer>
```

- [ ] **Step 7: Add drawer styles**

Append to `frontend/src/style.css`:

```css
.table-drawer-backdrop {
  position: fixed;
  inset: 0;
  z-index: 35;
  background: rgba(0, 0, 0, 0.34);
}

.table-drawer {
  position: absolute;
  color: #cfe8ff;
  background: #1d2632;
  box-shadow: 0 0 28px rgba(0, 0, 0, 0.28);
}

.drawer-left {
  top: 0;
  bottom: 0;
  left: 0;
  width: min(360px, 82vw);
  padding: 18px;
}

.drawer-right {
  top: 0;
  right: 0;
  bottom: 0;
  width: min(378px, 88vw);
  padding: 18px;
}

.drawer-bottom {
  left: 0;
  right: 0;
  bottom: 0;
  min-height: 34vh;
  max-height: 72vh;
  padding: 18px;
  border-radius: 8px 8px 0 0;
}

.drawer-close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 17px;
  color: white;
  background: #196da6;
}

.drawer-input-row {
  display: grid;
  grid-template-columns: 1fr 88px;
  gap: 10px;
  margin-top: 18px;
}

.drawer-input-row input {
  min-height: 48px;
  border: 0;
  border-radius: 6px;
  padding: 0 12px;
}

.table-settings-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.table-settings-grid button {
  min-height: 44px;
  border: 2px solid #f8da91;
  border-radius: 6px;
  color: white;
  background: #c79c49;
  font-weight: 900;
}
```

- [ ] **Step 8: Run tests**

Run:

```bash
cd frontend && npm test -- --run src/multiplayerTable.test.ts
```

Expected: PASS.

- [ ] **Step 9: Commit table panels**

```bash
git add frontend/src/components/TableDrawer.vue frontend/src/components/TableChatPanel.vue frontend/src/components/TableScorePanel.vue frontend/src/components/TableReplayPanel.vue frontend/src/components/TableSettingsPanel.vue frontend/src/pages/RoomPage.vue frontend/src/multiplayerTable.test.ts frontend/src/style.css
git commit -m "feat: add table drawer panels"
```

---

## Task 8: Final Compatibility, Visual Polish, And Verification

**Files:**
- Modify: `openspec/changes/mockup-driven-table-experience/tasks.md`
- Modify as needed: `frontend/src/style.css`
- Modify as needed: tests touched by this work

- [ ] **Step 1: Run OpenSpec validation**

Run:

```bash
npx --yes @fission-ai/openspec validate mockup-driven-table-experience --strict --no-interactive
```

Expected:

```text
Change 'mockup-driven-table-experience' is valid
```

- [ ] **Step 2: Run backend full test suite**

Run:

```bash
cd backend && go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run frontend tests**

Run:

```bash
cd frontend && npm test -- --run
```

Expected: PASS.

- [ ] **Step 4: Run frontend typecheck**

Run:

```bash
cd frontend && npm run typecheck
```

Expected: PASS.

- [ ] **Step 5: Manual smoke with dev server**

Run backend and frontend using the project's existing MySQL-backed setup. Then verify:

```text
1. Register/login two accounts.
2. Open /create-match, create a training short-deck room with 9 seats and 20 ante.
3. Join from the second account by invite code.
4. Click an empty seat, open buy-in modal, choose a legal buy-in, and sit down.
5. If wallet is insufficient, open /shop, simulate recharge, return to the room, and sit down.
6. Owner starts the hand.
7. Open chat, current score, replay, and settings panels; close each and confirm the table state remains current.
8. Visit /rules-test and confirm the independent rules page still works.
```

- [ ] **Step 6: Mark completed OpenSpec tasks**

In `openspec/changes/mockup-driven-table-experience/tasks.md`, mark completed items `[x]` only after the corresponding implementation and verification passed. Leave any unimplemented non-goal placeholders as completed only if they have a visible disabled/placeholder state and tests cover that state.

- [ ] **Step 7: Final status check**

Run:

```bash
git status --short
```

Expected: only intentional changed files remain.

- [ ] **Step 8: Commit final verification updates**

```bash
git add openspec/changes/mockup-driven-table-experience/tasks.md frontend/src/style.css frontend/src/multiplayerTable.test.ts frontend/src/App.visual.test.ts
git commit -m "test: verify mockup-driven table experience"
```

---

## Self-Review

Spec coverage:

- 效果图式创建比赛配置: Task 1, Task 2, Task 3.
- 商城金币页与模拟充值入口: Task 4.
- 牌桌内补充记分牌买入: Task 1, Task 5.
- 移动端牌桌主舞台: Task 6.
- 牌桌内聊天记录面板: Task 7.
- 当前战绩牌桌抽屉: Task 7.
- 牌谱回顾面板: Task 7.
- 牌桌设置菜单: Task 7.
- 现有多人能力与规则测试页保持兼容: Task 8.

Type consistency:

- Backend room metadata uses camelCase JSON fields matching frontend `RoomResponse`.
- Frontend create payload uses `CreateRoomPayload`, consumed by `createRoom()` and `doCreateRoom()`.
- Buy-in validation error code is `invalid_buy_in`, field is `buyInChips`.
- Table panels consume existing `chatMessages`, `roomLeaderboard`, `recentRoomHands`, `room.seats`, and `room.members`.

Known execution note:

- This plan intentionally preserves compatibility through `Store.CreateRoom(...)` while adding `Store.CreateRoomWithOptions(...)`, because existing tests and call sites already use the old method signature.
