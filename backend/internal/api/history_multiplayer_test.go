package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoomRecentHandsAndUserHands(t *testing.T) {
	server := testServer(t)
	roomID, ownerToken, _ := setupRoomWithSeats(t, server)

	startReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/start", nil)
	startReq.Header.Set("Authorization", "Bearer "+ownerToken)
	startRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusOK {
		t.Fatalf("start status=%d body=%s", startRes.Code, startRes.Body.String())
	}

	handsReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+roomID+"/hands/recent", nil)
	handsReq.Header.Set("Authorization", "Bearer "+ownerToken)
	handsRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(handsRes, handsReq)
	if handsRes.Code != http.StatusOK {
		t.Fatalf("recent hands status=%d body=%s", handsRes.Code, handsRes.Body.String())
	}

	myReq := httptest.NewRequest(http.MethodGet, "/api/me/hands", nil)
	myReq.Header.Set("Authorization", "Bearer "+ownerToken)
	myRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(myRes, myReq)
	if myRes.Code != http.StatusOK {
		t.Fatalf("my hands status=%d body=%s", myRes.Code, myRes.Body.String())
	}
}
