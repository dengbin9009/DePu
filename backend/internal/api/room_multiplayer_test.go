package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func registerUser(t *testing.T, server *Server, username, nickname string) string {
	t.Helper()
	body := []byte(`{"username":"` + username + `","password":"password1","nickname":"` + nickname + `"}`)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body)))
	if res.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", res.Code, res.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	token, _ := payload["token"].(string)
	return token
}

func TestCreateRoomJoinAndTakeSeat(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner01", "房主A")
	playerToken := registerUser(t, server, "player02", "玩家B")

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"long-holdem","seatCount":6,"minPlayersToStart":2}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var room map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	inviteCode, _ := room["inviteCode"].(string)
	roomID, _ := room["id"].(string)
	if inviteCode == "" || roomID == "" {
		t.Fatal("expected inviteCode and room id")
	}

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+inviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK {
		t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String())
	}

	joinAgainReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"  `+strings.ToLower(inviteCode)+`  "}`)))
	joinAgainReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinAgainRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinAgainRes, joinAgainReq)
	if joinAgainRes.Code != http.StatusOK {
		t.Fatalf("join room with normalized invite status=%d body=%s", joinAgainRes.Code, joinAgainRes.Body.String())
	}

	takeReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/1", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	takeReq.Header.Set("Authorization", "Bearer "+ownerToken)
	takeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeRes, takeReq)
	if takeRes.Code != http.StatusOK {
		t.Fatalf("take seat status=%d body=%s", takeRes.Code, takeRes.Body.String())
	}

	takeAgainReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/2", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	takeAgainReq.Header.Set("Authorization", "Bearer "+ownerToken)
	takeAgainRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeAgainRes, takeAgainReq)
	if takeAgainRes.Code != http.StatusConflict {
		t.Fatalf("duplicate take seat status=%d body=%s", takeAgainRes.Code, takeAgainRes.Body.String())
	}
	var errBody ErrorResponse
	if err := json.Unmarshal(takeAgainRes.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "already_seated" {
		t.Fatalf("duplicate seat code=%s, want already_seated", errBody.Code)
	}
}

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
		name  string
		body  string
		code  string
		field string
	}{
		{
			name:  "unsupported sng",
			body:  `{"ruleSetId":"short-deck","mode":"sng","variant":"short_holdem","seatCount":9,"minPlayersToStart":2}`,
			code:  "unsupported_room_mode",
			field: "mode",
		},
		{
			name:  "min buy in greater than max",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","minBuyIn":9000,"maxBuyIn":2000,"seatCount":9,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "maxBuyIn",
		},
		{
			name:  "unsupported omaha",
			body:  `{"ruleSetId":"omaha","mode":"training","variant":"omaha","seatCount":9,"minPlayersToStart":2}`,
			code:  "unsupported_variant",
			field: "variant",
		},
		{
			name:  "explicit zero duration",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","durationMinutes":0,"seatCount":9,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "durationMinutes",
		},
		{
			name:  "explicit zero ante",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","ante":0,"seatCount":9,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "ante",
		},
		{
			name:  "explicit zero min buy in",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","minBuyIn":0,"maxBuyIn":8000,"seatCount":9,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "minBuyIn",
		},
		{
			name:  "explicit zero max buy in",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","minBuyIn":2000,"maxBuyIn":0,"seatCount":9,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "maxBuyIn",
		},
		{
			name:  "explicit zero buy in cap",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","maxBuyIn":8000,"buyInCap":0,"seatCount":9,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "buyInCap",
		},
		{
			name:  "explicit zero seat count",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","seatCount":0,"minPlayersToStart":2}`,
			code:  "invalid_room_config",
			field: "seatCount",
		},
		{
			name:  "explicit zero min players",
			body:  `{"ruleSetId":"short-deck","mode":"training","variant":"short_holdem","seatCount":9,"minPlayersToStart":0}`,
			code:  "invalid_room_config",
			field: "minPlayersToStart",
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
			if errBody.Code != tc.code || errBody.Field != tc.field {
				t.Fatalf("error=%#v, want code=%s field=%s", errBody, tc.code, tc.field)
			}
		})
	}
}

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
		field  string
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
		if errBody.Code != "invalid_buy_in" || errBody.Field != tc.field {
			t.Fatalf("amount=%d error=%#v", tc.amount, errBody)
		}
	}
}

func TestTakeSeatRequiresRoomMembership(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "seat_member_owner", "成员房主")
	outsiderToken := registerUser(t, server, "seat_member_outsider", "非成员玩家")

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"long-holdem","seatCount":6,"minPlayersToStart":2}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var room map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	roomID := room["id"].(string)

	takeReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/2", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	takeReq.Header.Set("Authorization", "Bearer "+outsiderToken)
	takeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeRes, takeReq)
	if takeRes.Code != http.StatusForbidden {
		t.Fatalf("outsider take seat status=%d body=%s", takeRes.Code, takeRes.Body.String())
	}
	var errBody ErrorResponse
	if err := json.Unmarshal(takeRes.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "forbidden" {
		t.Fatalf("outsider take seat code=%s, want forbidden", errBody.Code)
	}
}

func TestJoinRejectsInvalidInviteCodeAndSeatTaken(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner01", "房主A")
	playerToken := registerUser(t, server, "player02", "玩家B")

	badJoinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"BADCODE"}`)))
	badJoinReq.Header.Set("Authorization", "Bearer "+playerToken)
	badJoinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(badJoinRes, badJoinReq)
	if badJoinRes.Code != http.StatusBadRequest {
		t.Fatalf("invalid invite status=%d body=%s", badJoinRes.Code, badJoinRes.Body.String())
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"long-holdem","seatCount":6,"minPlayersToStart":2}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	var room map[string]any
	_ = json.Unmarshal(createRes.Body.Bytes(), &room)
	inviteCode, _ := room["inviteCode"].(string)
	roomID, _ := room["id"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+inviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK {
		t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String())
	}

	take1 := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/1", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	take1.Header.Set("Authorization", "Bearer "+ownerToken)
	res1 := httptest.NewRecorder()
	server.Routes().ServeHTTP(res1, take1)
	if res1.Code != http.StatusOK {
		t.Fatalf("take1 status=%d body=%s", res1.Code, res1.Body.String())
	}

	take2 := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/1", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	take2.Header.Set("Authorization", "Bearer "+playerToken)
	res2 := httptest.NewRecorder()
	server.Routes().ServeHTTP(res2, take2)
	if res2.Code != http.StatusConflict {
		t.Fatalf("take2 status=%d body=%s", res2.Code, res2.Body.String())
	}
}

func TestTakeSeatRejectsInsufficientCoins(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner03", "房主C")
	playerToken := registerUser(t, server, "player03", "玩家C")

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"long-holdem","seatCount":6,"minPlayersToStart":2}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	var room map[string]any
	_ = json.Unmarshal(createRes.Body.Bytes(), &room)
	inviteCode, _ := room["inviteCode"].(string)
	roomID, _ := room["id"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+inviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK {
		t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String())
	}

	takeReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/2", bytes.NewReader([]byte(`{"buyInChips":999999}`)))
	takeReq.Header.Set("Authorization", "Bearer "+playerToken)
	takeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeRes, takeReq)
	if takeRes.Code != http.StatusConflict {
		t.Fatalf("take seat status=%d body=%s", takeRes.Code, takeRes.Body.String())
	}

	roomReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID, nil)
	roomReq.Header.Set("Authorization", "Bearer "+playerToken)
	roomRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(roomRes, roomReq)
	if roomRes.Code != http.StatusOK {
		t.Fatalf("room status=%d body=%s", roomRes.Code, roomRes.Body.String())
	}
	var roomState struct {
		Seats []struct {
			SeatNo int     `json:"seatNo"`
			UserID *string `json:"userId"`
		} `json:"seats"`
	}
	if err := json.Unmarshal(roomRes.Body.Bytes(), &roomState); err != nil {
		t.Fatal(err)
	}
	for _, seat := range roomState.Seats {
		if seat.SeatNo == 2 && seat.UserID != nil {
			t.Fatal("seat 2 should remain empty after insufficient coins failure")
		}
	}
}

func TestStartFailsWithoutEnoughSeatedPlayersAndDoesNotCreateCurrentHand(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner04", "房主D")

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"long-holdem","seatCount":6,"minPlayersToStart":2}`)))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var room map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	roomID, _ := room["id"].(string)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusForbidden {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
	currentRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(currentRes, currentReq)
	if currentRes.Code != http.StatusNotFound {
		t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
	}
}
