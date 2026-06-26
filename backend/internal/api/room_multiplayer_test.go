package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func registerUser(t *testing.T, server *Server, username, nickname string) string {
	t.Helper()
	body := []byte(`{"username":"` + username + `","password":"password1","nickname":"` + nickname + `"}`)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body)))
	if res.Code != http.StatusCreated { t.Fatalf("register status=%d body=%s", res.Code, res.Body.String()) }
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil { t.Fatal(err) }
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
	if err := json.Unmarshal(createRes.Body.Bytes(), &room); err != nil { t.Fatal(err) }
	inviteCode, _ := room["inviteCode"].(string)
	roomID, _ := room["id"].(string)
	if inviteCode == "" || roomID == "" { t.Fatal("expected inviteCode and room id") }

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+inviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK { t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String()) }

	takeReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/1", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	takeReq.Header.Set("Authorization", "Bearer "+ownerToken)
	takeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeRes, takeReq)
	if takeRes.Code != http.StatusOK { t.Fatalf("take seat status=%d body=%s", takeRes.Code, takeRes.Body.String()) }
}

func TestJoinRejectsInvalidInviteCodeAndSeatTaken(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "owner01", "房主A")
	playerToken := registerUser(t, server, "player02", "玩家B")

	badJoinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"BADCODE"}`)))
	badJoinReq.Header.Set("Authorization", "Bearer "+playerToken)
	badJoinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(badJoinRes, badJoinReq)
	if badJoinRes.Code != http.StatusBadRequest { t.Fatalf("invalid invite status=%d body=%s", badJoinRes.Code, badJoinRes.Body.String()) }

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
	if joinRes.Code != http.StatusOK { t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String()) }

	take1 := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/1", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	take1.Header.Set("Authorization", "Bearer "+ownerToken)
	res1 := httptest.NewRecorder()
	server.Routes().ServeHTTP(res1, take1)
	if res1.Code != http.StatusOK { t.Fatalf("take1 status=%d body=%s", res1.Code, res1.Body.String()) }

	take2 := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/1", bytes.NewReader([]byte(`{"buyInChips":1000}`)))
	take2.Header.Set("Authorization", "Bearer "+playerToken)
	res2 := httptest.NewRecorder()
	server.Routes().ServeHTTP(res2, take2)
	if res2.Code != http.StatusConflict { t.Fatalf("take2 status=%d body=%s", res2.Code, res2.Body.String()) }
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
	if joinRes.Code != http.StatusOK { t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String()) }

	takeReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/2", bytes.NewReader([]byte(`{"buyInChips":999999}`)))
	takeReq.Header.Set("Authorization", "Bearer "+playerToken)
	takeRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(takeRes, takeReq)
	if takeRes.Code != http.StatusConflict { t.Fatalf("take seat status=%d body=%s", takeRes.Code, takeRes.Body.String()) }

	roomReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID, nil)
	roomReq.Header.Set("Authorization", "Bearer "+playerToken)
	roomRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(roomRes, roomReq)
	if roomRes.Code != http.StatusOK { t.Fatalf("room status=%d body=%s", roomRes.Code, roomRes.Body.String()) }
	var roomState struct {
		Seats []struct {
			SeatNo int `json:"seatNo"`
			UserID *string `json:"userId"`
		} `json:"seats"`
	}
	if err := json.Unmarshal(roomRes.Body.Bytes(), &roomState); err != nil { t.Fatal(err) }
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
	if createRes.Code != http.StatusCreated { t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String()) }
	var room map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &room); err != nil { t.Fatal(err) }
	roomID, _ := room["id"].(string)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusForbidden { t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String()) }

	currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
	currentRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(currentRes, currentReq)
	if currentRes.Code != http.StatusNotFound {
		t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
	}
}
