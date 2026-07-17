package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dengbin9009/DePu/backend/internal/storage"
	"github.com/dengbin9009/DePu/backend/internal/testmysql"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	store, err := openTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return NewServerWithStore(store)
}

func openTestStore(t *testing.T) (*storage.Store, error) {
	t.Helper()
	database, err := testmysql.CreateDatabase(testmysql.AdminDSN(), "api")
	if err != nil {
		t.Skipf("mysql test database unavailable: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Cleanup(); err != nil {
			t.Errorf("cleanup mysql test database %s: %v", database.Name, err)
		}
	})
	store, err := storage.OpenWithConfig(storage.Config{Driver: storage.DriverMySQL, DSN: database.DSN})
	if err != nil {
		t.Skipf("mysql test store unavailable: %v", err)
	}
	return store, nil
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
	var rawSnapshot map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &rawSnapshot); err != nil {
		t.Fatal(err)
	}
	if rawSnapshot["currentBet"] != float64(100) {
		t.Fatalf("currentBet = %v, want 100", rawSnapshot["currentBet"])
	}
	if rawSnapshot["minRaise"] != float64(100) {
		t.Fatalf("minRaise = %v, want 100", rawSnapshot["minRaise"])
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

func TestSnapshotCurrentHandAppearsAfterFlopOnly(t *testing.T) {
	server := testServer(t)
	createBody := []byte(`{
		"rulesetId":"long-holdem",
		"buttonSeat":1,
		"smallBlind":50,
		"bigBlind":100,
		"dealMode":"debug",
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
	var preflop GameSnapshot
	if err := json.Unmarshal(createRes.Body.Bytes(), &preflop); err != nil {
		t.Fatal(err)
	}
	if len(preflop.Seats) == 0 || preflop.Seats[0].CurrentHand != nil {
		t.Fatalf("preflop current hand = %#v, want nil", preflop.Seats[0].CurrentHand)
	}

	debugBody := []byte(`{
		"version":` + itoa(preflop.Version) + `,
		"holeCards":{
			"1":["As","Ad"],
			"2":["Ks","Qh"],
			"3":["9c","8c"],
			"4":["2h","3d"]
		},
		"board":["Ah","7s","2c"]
	}`)
	debugRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(debugRes, httptest.NewRequest(http.MethodPost, "/api/games/"+preflop.ID+"/debug/cards", bytes.NewReader(debugBody)))
	if debugRes.Code != http.StatusOK {
		t.Fatalf("debug status = %d body=%s", debugRes.Code, debugRes.Body.String())
	}
	var flop GameSnapshot
	if err := json.Unmarshal(debugRes.Body.Bytes(), &flop); err != nil {
		t.Fatal(err)
	}
	if len(flop.Seats) == 0 || flop.Seats[0].CurrentHand == nil {
		t.Fatalf("flop current hand missing: %#v", flop.Seats)
	}
	if flop.Seats[0].CurrentHand.HandClass != "three_of_a_kind" {
		t.Fatalf("hand class = %s, want three_of_a_kind", flop.Seats[0].CurrentHand.HandClass)
	}
	if len(flop.Seats[0].CurrentHand.BestCards) != 5 {
		t.Fatalf("best cards = %v, want five cards", flop.Seats[0].CurrentHand.BestCards)
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
