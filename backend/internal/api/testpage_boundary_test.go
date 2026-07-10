package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTestPageAndMultiplayerRoutesStaySeparated(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "boundary_owner", "边界房主")

	createRoomReq := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader([]byte(`{"ruleSetId":"holdem","seatCount":2,"minPlayersToStart":2}`)))
	createRoomReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRoomRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRoomRes, createRoomReq)
	if createRoomRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRoomRes.Code, createRoomRes.Body.String())
	}

	gameReq := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader([]byte(`{
		"rulesetId":"short-deck",
		"buttonSeat":1,
		"bettingStructure":{"type":"ante","ante":10,"buttonBlind":20},
		"dealMode":"random",
		"seats":[
			{"seatNo":1,"name":"A","stack":1000},
			{"seatNo":2,"name":"B","stack":1000}
		]
	}`)))
	gameRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(gameRes, gameReq)
	if gameRes.Code != http.StatusCreated {
		t.Fatalf("create test page game status=%d body=%s", gameRes.Code, gameRes.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/api/games/not-a-room/history", nil)
	historyRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(historyRes, historyReq)
	if historyRes.Code != http.StatusOK {
		t.Fatalf("game history route should remain handled by test-page namespace, got=%d body=%s", historyRes.Code, historyRes.Body.String())
	}

	roomReq := httptest.NewRequest(http.MethodGet, "/api/rooms/not-a-game", nil)
	roomReq.Header.Set("Authorization", "Bearer "+ownerToken)
	roomRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(roomRes, roomReq)
	if roomRes.Code != http.StatusNotFound {
		t.Fatalf("room lookup should not resolve test-page game ids, got=%d body=%s", roomRes.Code, roomRes.Body.String())
	}
}
