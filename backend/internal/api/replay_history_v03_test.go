package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dengbin9009/DePu/backend/internal/game"
)

func TestReplayReturnsReadOnlySnapshotAndRejectsOutOfRange(t *testing.T) {
	server := testServer(t)
	createBody := []byte(`{
		"rulesetId":"long-holdem",
		"buttonSeat":1,
		"bettingStructure":{"type":"blinds","smallBlind":50,"bigBlind":100},
		"dealMode":"random",
		"seats":[
			{"seatNo":1,"name":"BTN","stack":1000},
			{"seatNo":2,"name":"SB","stack":1000},
			{"seatNo":3,"name":"BB","stack":1000}
		]
	}`)
	createRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRes, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(createBody)))
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRes.Code, createRes.Body.String())
	}
	var created GameSnapshot
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	replayRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(replayRes, httptest.NewRequest(http.MethodPost, "/api/games/"+created.ID+"/replay", bytes.NewReader([]byte(`{"toSeq":0}`))))
	if replayRes.Code != http.StatusOK {
		t.Fatalf("replay status = %d body=%s", replayRes.Code, replayRes.Body.String())
	}
	var replayed GameSnapshot
	if err := json.Unmarshal(replayRes.Body.Bytes(), &replayed); err != nil {
		t.Fatal(err)
	}
	if !replayed.IsReplay {
		t.Fatal("replay snapshot should have isReplay=true")
	}

	actionRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(actionRes, httptest.NewRequest(http.MethodPost, "/api/games/"+created.ID+"/actions", bytes.NewReader([]byte(`{"seatNo":3,"type":"call","version":`+itoa(replayed.Version)+`}`))))
	if actionRes.Code != http.StatusConflict {
		t.Fatalf("replay-based submit status = %d body=%s", actionRes.Code, actionRes.Body.String())
	}

	outRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(outRes, httptest.NewRequest(http.MethodPost, "/api/games/"+created.ID+"/replay", bytes.NewReader([]byte(`{"toSeq":999}`))))
	if outRes.Code != http.StatusBadRequest {
		t.Fatalf("out-of-range status = %d body=%s", outRes.Code, outRes.Body.String())
	}
	var errBody ErrorResponse
	if err := json.Unmarshal(outRes.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "replay_out_of_range" {
		t.Fatalf("code = %s, want replay_out_of_range", errBody.Code)
	}
}

func TestHistoryReturnsStructuredStateSummaryAndSystemActions(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := NewServerWithStore(store)
	g, err := game.New(game.Config{
		RuleSetID:  "short-deck",
		ButtonSeat: 1,
		BettingStructure: game.BettingStructure{
			Type:        game.BettingAnte,
			Ante:        10,
			ButtonBlind: 50,
		},
		Seats: []game.SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "A", Stack: 1000},
			{SeatNo: 3, Name: "B", Stack: 1000},
		},
		DealMode: game.DealDebug,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SetDebugCards(map[int][]string{1: []string{"As", "Ah"}}, []string{"Ks", "Kh", "Qh"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(g); err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/games/"+g.ID+"/history", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("history status = %d body=%s", res.Code, res.Body.String())
	}
	var history []struct {
		Type         string            `json:"type"`
		StateSummary game.StateSummary `json:"stateSummary"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	hasForcedBet := false
	hasDebugSet := false
	for _, item := range history {
		if item.Type == string(game.ActionPost) {
			hasForcedBet = true
		}
		if item.Type == string(game.ActionSet) {
			hasDebugSet = true
		}
		if item.StateSummary.Stage == "" {
			t.Fatalf("history item has empty stateSummary: %#v", item)
		}
	}
	if !hasForcedBet || !hasDebugSet {
		t.Fatalf("history missing forced/debug actions: forced=%v debug=%v", hasForcedBet, hasDebugSet)
	}
}
