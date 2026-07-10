package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupRoomWithSeats(t *testing.T, server *Server) (roomID string, ownerToken string, playerToken string) {
	t.Helper()
	ownerToken = registerUser(t, server, "owner01", "房主A")
	playerToken = registerUser(t, server, "player02", "玩家B")

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
	roomID, _ = room["id"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewReader([]byte(`{"inviteCode":"`+inviteCode+`"}`)))
	joinReq.Header.Set("Authorization", "Bearer "+playerToken)
	joinRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(joinRes, joinReq)
	if joinRes.Code != http.StatusOK {
		t.Fatalf("join room status=%d body=%s", joinRes.Code, joinRes.Body.String())
	}

	for _, seat := range []struct{ token string; seat int }{{ownerToken, 1}, {playerToken, 2}} {
		req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/seats/"+itoa(seat.seat), bytes.NewReader([]byte(`{"buyInChips":1000}`)))
		req.Header.Set("Authorization", "Bearer "+seat.token)
		res := httptest.NewRecorder()
		server.Routes().ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("take seat status=%d body=%s", res.Code, res.Body.String())
		}
	}
	return roomID, ownerToken, playerToken
}

func TestOwnerCanStartRoomHandAndFetchCurrentHand(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/current-hand", nil)
	currentReq.Header.Set("Authorization", "Bearer "+ownerToken)
	currentRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(currentRes, currentReq)
	if currentRes.Code != http.StatusOK {
		t.Fatalf("current hand status=%d body=%s", currentRes.Code, currentRes.Body.String())
	}
}

func TestNonOwnerCannotStartAndNonCurrentActorCannotAct(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	badStartReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	badStartReq.Header.Set("Authorization", "Bearer "+playerToken)
	badStartRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(badStartRes, badStartReq)
	if badStartRes.Code != http.StatusForbidden {
		t.Fatalf("non-owner start status=%d body=%s", badStartRes.Code, badStartRes.Body.String())
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	actReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/actions", bytes.NewReader([]byte(`{"action":"call","amount":0}`)))
	actReq.Header.Set("Authorization", "Bearer "+playerToken)
	actRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(actRes, actReq)
	if actRes.Code != http.StatusForbidden {
		t.Fatalf("non-current actor status=%d body=%s", actRes.Code, actRes.Body.String())
	}
}
