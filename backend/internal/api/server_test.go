package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dengbin9009/DePu/backend/internal/storage"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	store, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return NewServerWithStore(store)
}

func TestRulesetsEndpoint(t *testing.T) {
	server := testServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/rulesets", nil)

	server.Routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 2 {
		t.Fatalf("rulesets = %d, want 2", len(body))
	}
}

func TestCreateGameAndSubmitAction(t *testing.T) {
	server := testServer(t)
	createBody := []byte(`{
		"rulesetId":"long-holdem",
		"buttonSeat":1,
		"smallBlind":50,
		"bigBlind":100,
		"dealMode":"random",
		"seats":[
			{"seatNo":1,"name":"BTN","stack":1000},
			{"seatNo":2,"name":"SB","stack":1000},
			{"seatNo":3,"name":"BB","stack":1000},
			{"seatNo":4,"name":"UTG","stack":1000}
		]
	}`)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(createBody)))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRes.Code, createRes.Body.String())
	}
	var snapshot GameSnapshot
	if err := json.Unmarshal(createRes.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.CurrentSeat != 4 {
		t.Fatalf("current seat = %d, want 4", snapshot.CurrentSeat)
	}
	if len(snapshot.LegalActions) == 0 {
		t.Fatal("expected legal actions")
	}

	actionBody := []byte(`{"seatNo":4,"type":"call","version":` + itoa(snapshot.Version) + `}`)
	actionRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(actionRes, httptest.NewRequest(http.MethodPost, "/api/games/"+snapshot.ID+"/actions", bytes.NewReader(actionBody)))
	if actionRes.Code != http.StatusOK {
		t.Fatalf("action status = %d body=%s", actionRes.Code, actionRes.Body.String())
	}
	var next GameSnapshot
	if err := json.Unmarshal(actionRes.Body.Bytes(), &next); err != nil {
		t.Fatal(err)
	}
	if next.Version <= snapshot.Version {
		t.Fatalf("version did not advance: %d -> %d", snapshot.Version, next.Version)
	}
}

func TestDebugCardsRejectsDuplicate(t *testing.T) {
	server := testServer(t)
	createBody := []byte(`{
		"rulesetId":"short-deck",
		"buttonSeat":1,
		"smallBlind":50,
		"bigBlind":100,
		"dealMode":"debug",
		"seats":[
			{"seatNo":1,"name":"BTN","stack":1000},
			{"seatNo":2,"name":"SB","stack":1000}
		]
	}`)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(createBody)))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRes.Code, createRes.Body.String())
	}
	var snapshot GameSnapshot
	if err := json.Unmarshal(createRes.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}

	debugBody := []byte(`{"version":` + itoa(snapshot.Version) + `,"holeCards":{"1":["As","Ah"],"2":["As","Kh"]},"board":[]}`)
	debugRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(debugRes, httptest.NewRequest(http.MethodPost, "/api/games/"+snapshot.ID+"/debug/cards", bytes.NewReader(debugBody)))
	if debugRes.Code != http.StatusBadRequest {
		t.Fatalf("debug status = %d body=%s", debugRes.Code, debugRes.Body.String())
	}
}

func TestDebugCardsRejectsShortDeckLowCard(t *testing.T) {
	server := testServer(t)
	createBody := []byte(`{
		"rulesetId":"short-deck",
		"buttonSeat":1,
		"smallBlind":50,
		"bigBlind":100,
		"dealMode":"debug",
		"seats":[
			{"seatNo":1,"name":"BTN","stack":1000},
			{"seatNo":2,"name":"SB","stack":1000}
		]
	}`)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(createBody)))
	var snapshot GameSnapshot
	if err := json.Unmarshal(createRes.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}

	debugBody := []byte(`{"version":` + itoa(snapshot.Version) + `,"holeCards":{"1":["2s","Ah"],"2":["Ks","Kh"]},"board":[]}`)
	debugRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(debugRes, httptest.NewRequest(http.MethodPost, "/api/games/"+snapshot.ID+"/debug/cards", bytes.NewReader(debugBody)))
	if debugRes.Code != http.StatusBadRequest {
		t.Fatalf("debug status = %d body=%s", debugRes.Code, debugRes.Body.String())
	}
}
