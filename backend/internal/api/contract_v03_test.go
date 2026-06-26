package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateGameAcceptsBettingStructureObject(t *testing.T) {
	server := testServer(t)
	body := []byte(`{
		"rulesetId":"short-deck",
		"buttonSeat":1,
		"bettingStructure":{"type":"ante","ante":10,"buttonBlind":50},
		"dealMode":"random",
		"seats":[
			{"seatNo":1,"name":"BTN","stack":1000},
			{"seatNo":2,"name":"A","stack":1000},
			{"seatNo":3,"name":"B","stack":1000}
		]
	}`)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body)))
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
	var snapshot struct {
		BettingStructure struct {
			Type        string `json:"type"`
			Ante        int    `json:"ante"`
			ButtonBlind int    `json:"buttonBlind"`
		} `json:"bettingStructure"`
		IsReplay    bool `json:"isReplay"`
		DebugLocked bool `json:"debugLocked"`
		CurrentSeat int  `json:"currentSeat"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.BettingStructure.Type != "ante" || snapshot.BettingStructure.Ante != 10 || snapshot.BettingStructure.ButtonBlind != 50 {
		t.Fatalf("bettingStructure = %#v", snapshot.BettingStructure)
	}
	if snapshot.IsReplay {
		t.Fatal("new authoritative snapshot should not be replay")
	}
	if snapshot.DebugLocked {
		t.Fatal("debug should not be locked before player action")
	}
	if snapshot.CurrentSeat != 2 {
		t.Fatalf("current seat = %d, want button left seat 2", snapshot.CurrentSeat)
	}
}

func TestCreateGameReturnsStableValidationCodes(t *testing.T) {
	server := testServer(t)
	body := []byte(`{
		"rulesetId":"long-holdem",
		"buttonSeat":4,
		"bettingStructure":{"type":"ante","ante":10,"buttonBlind":50},
		"dealMode":"random",
		"seats":[
			{"seatNo":1,"name":"A","stack":1000},
			{"seatNo":2,"name":"B","stack":1000}
		]
	}`)
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body)))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
	var errBody ErrorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "invalid_button" {
		t.Fatalf("code = %s, want invalid_button", errBody.Code)
	}
	if errBody.Field != "buttonSeat" {
		t.Fatalf("field = %s, want buttonSeat", errBody.Field)
	}
}
