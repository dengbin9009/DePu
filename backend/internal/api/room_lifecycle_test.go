package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOwnerLeaveTransfersOwnershipAndEmptyRoomCloses(t *testing.T) {
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
	if err := json.Unmarshal(ownerLeaveRes.Body.Bytes(), &room); err != nil { t.Fatal(err) }
	ownerUserID, _ := room["ownerUserId"].(string)
	if ownerUserID == "" {
		t.Fatal("expected ownerUserId after transfer")
	}

	playerLeaveReq := httptest.NewRequest(http.MethodDelete, "/api/rooms/"+roomID+"/seats/2", nil)
	playerLeaveReq.Header.Set("Authorization", "Bearer "+playerToken)
	playerLeaveRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(playerLeaveRes, playerLeaveReq)
	if playerLeaveRes.Code != http.StatusOK {
		t.Fatalf("player leave status=%d body=%s", playerLeaveRes.Code, playerLeaveRes.Body.String())
	}
	var closed map[string]any
	if err := json.Unmarshal(playerLeaveRes.Body.Bytes(), &closed); err != nil { t.Fatal(err) }
	if closed["status"] != "closed" {
		t.Fatalf("room status = %v, want closed", closed["status"])
	}
}
