package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStandUpKeepsRoomMembershipAndOwnership(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	ownerLeaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/seats/1", nil)
	ownerLeaveReq.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerLeaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(ownerLeaveRes, ownerLeaveReq)
	if ownerLeaveRes.Code != http.StatusOK {
		t.Fatalf("owner leave status=%d body=%s", ownerLeaveRes.Code, ownerLeaveRes.Body.String())
	}
	var room map[string]any
	if err := json.Unmarshal(ownerLeaveRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	ownerUserID, _ := room["ownerUserId"].(string)
	if ownerUserID == "" {
		t.Fatal("expected ownerUserId to stay set after standing up")
	}
	if room["status"] != "waiting" {
		t.Fatalf("room status = %v, want waiting", room["status"])
	}
	if members, _ := room["members"].([]any); len(members) != 2 {
		t.Fatalf("members after owner stand up = %d, want 2", len(members))
	}

	playerLeaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/seats/2", nil)
	playerLeaveReq.Header.Set("Authorization", "Bearer "+playerToken)
	playerLeaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(playerLeaveRes, playerLeaveReq)
	if playerLeaveRes.Code != http.StatusOK {
		t.Fatalf("player leave status=%d body=%s", playerLeaveRes.Code, playerLeaveRes.Body.String())
	}
	var standing map[string]any
	if err := json.Unmarshal(playerLeaveRes.Body.Bytes(), &standing); err != nil {
		t.Fatal(err)
	}
	if standing["status"] != "waiting" {
		t.Fatalf("room status = %v, want waiting", standing["status"])
	}
	if members, _ := standing["members"].([]any); len(members) != 2 {
		t.Fatalf("members after both stand up = %d, want 2", len(members))
	}
}

func TestStandUpRejectsPlayingRoom(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start room status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	leaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/seats/1", nil)
	leaveReq.Header.Set("Authorization", "Bearer "+ownerToken)
	leaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaveRes, leaveReq)
	if leaveRes.Code != http.StatusConflict {
		t.Fatalf("stand during playing status=%d body=%s", leaveRes.Code, leaveRes.Body.String())
	}
	var errBody ErrorResponse
	if err := json.Unmarshal(leaveRes.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "room_not_waiting" {
		t.Fatalf("stand during playing code=%s, want room_not_waiting", errBody.Code)
	}
}

func TestRoomOwnerLeaveTransfersOwnershipToNextSeatedPlayer(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	leaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/members/me", nil)
	leaveReq.Header.Set("Authorization", "Bearer "+ownerToken)
	leaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaveRes, leaveReq)
	if leaveRes.Code != http.StatusOK {
		t.Fatalf("owner room leave status=%d body=%s", leaveRes.Code, leaveRes.Body.String())
	}

	var room map[string]any
	if err := json.Unmarshal(leaveRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if room["status"] != "waiting" {
		t.Fatalf("room status = %v, want waiting", room["status"])
	}

	playerReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	playerReq.Header.Set("Authorization", "Bearer "+playerToken)
	playerRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(playerRes, playerReq)
	if playerRes.Code != http.StatusOK {
		t.Fatalf("player profile status=%d body=%s", playerRes.Code, playerRes.Body.String())
	}
	var playerProfile struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(playerRes.Body.Bytes(), &playerProfile); err != nil {
		t.Fatal(err)
	}
	if room["ownerUserId"] != playerProfile.ID {
		t.Fatalf("ownerUserId=%v, want transferred player %s", room["ownerUserId"], playerProfile.ID)
	}

	members, _ := room["members"].([]any)
	if len(members) != 1 {
		t.Fatalf("members after owner leave = %d, want 1", len(members))
	}
	member := members[0].(map[string]any)
	if member["userId"] != playerProfile.ID || member["role"] != "owner" {
		t.Fatalf("remaining member = %#v, want transferred owner", member)
	}
}

func TestRoomPlayerLeaveDoesNotChangeOwner(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, playerToken := setupRoomWithSeats(t, server)

	ownerReq := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ownerReq.Header.Set("Authorization", "Bearer "+ownerToken)
	ownerRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(ownerRes, ownerReq)
	if ownerRes.Code != http.StatusOK {
		t.Fatalf("owner profile status=%d body=%s", ownerRes.Code, ownerRes.Body.String())
	}
	var ownerProfile struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(ownerRes.Body.Bytes(), &ownerProfile); err != nil {
		t.Fatal(err)
	}

	leaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/members/me", nil)
	leaveReq.Header.Set("Authorization", "Bearer "+playerToken)
	leaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaveRes, leaveReq)
	if leaveRes.Code != http.StatusOK {
		t.Fatalf("player room leave status=%d body=%s", leaveRes.Code, leaveRes.Body.String())
	}

	var room map[string]any
	if err := json.Unmarshal(leaveRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if room["ownerUserId"] != ownerProfile.ID {
		t.Fatalf("ownerUserId=%v, want original owner %s", room["ownerUserId"], ownerProfile.ID)
	}
	members, _ := room["members"].([]any)
	if len(members) != 1 {
		t.Fatalf("members after player leave = %d, want 1", len(members))
	}
}

func TestRoomOwnerLeaveClosesRoomWhenNoSuccessor(t *testing.T) {
	server := testServer(t)
	ownerToken := registerUser(t, server, "solo_owner", "单人房主")
	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", strings.NewReader(`{"ruleSetId":"long-holdem","seatCount":2,"minPlayersToStart":2}`))
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create room status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	roomID := created["id"].(string)

	leaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/members/me", nil)
	leaveReq.Header.Set("Authorization", "Bearer "+ownerToken)
	leaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(leaveRes, leaveReq)
	if leaveRes.Code != http.StatusOK {
		t.Fatalf("solo owner room leave status=%d body=%s", leaveRes.Code, leaveRes.Body.String())
	}

	var room map[string]any
	if err := json.Unmarshal(leaveRes.Body.Bytes(), &room); err != nil {
		t.Fatal(err)
	}
	if room["status"] != "closed" {
		t.Fatalf("room status = %v, want closed", room["status"])
	}
	if room["ownerUserId"] != "" {
		t.Fatalf("ownerUserId=%v, want empty after closed", room["ownerUserId"])
	}
}
