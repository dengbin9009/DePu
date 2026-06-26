package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/api"
	"github.com/dengbin9009/DePu/backend/internal/storage"
)

func TestLocalPerformanceBudget(t *testing.T) {
	store, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := api.NewServerWithStore(store)

	createBody := []byte(`{
        "rulesetId":"short-deck",
        "buttonSeat":1,
        "bettingStructure":{"type":"ante","ante":10,"buttonBlind":50},
        "dealMode":"random",
        "seats":[
            {"seatNo":1,"name":"BTN","stack":1000},
            {"seatNo":2,"name":"A","stack":1000},
            {"seatNo":3,"name":"B","stack":1000},
            {"seatNo":4,"name":"C","stack":1000}
        ]
    }`)

	started := time.Now()
	res := httptest.NewRecorder()
	server.Routes().ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(createBody)))
	if res.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", res.Code, res.Body.String())
	}
	var created struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	createBudget := time.Since(started)
	if createBudget > 200*time.Millisecond {
		t.Fatalf("create budget = %s, want <= 200ms", createBudget)
	}

	actionBudgetStart := time.Now()
	actionRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(actionRes, httptest.NewRequest(http.MethodPost, "/api/games/"+created.ID+"/actions", bytes.NewReader([]byte(`{"seatNo":4,"type":"call","version":`+strconv.Itoa(created.Version)+`}`))))
	if actionRes.Code != http.StatusOK && actionRes.Code != http.StatusConflict {
		t.Fatalf("action status = %d body=%s", actionRes.Code, actionRes.Body.String())
	}
	if time.Since(actionBudgetStart) > 200*time.Millisecond {
		t.Fatalf("action budget = %s, want <= 200ms", time.Since(actionBudgetStart))
	}

	historyBudgetStart := time.Now()
	historyRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(historyRes, httptest.NewRequest(http.MethodGet, "/api/games/"+created.ID+"/history", nil))
	if historyRes.Code != http.StatusOK {
		t.Fatalf("history status = %d body=%s", historyRes.Code, historyRes.Body.String())
	}
	if time.Since(historyBudgetStart) > 200*time.Millisecond {
		t.Fatalf("history budget = %s, want <= 200ms", time.Since(historyBudgetStart))
	}

	replayBudgetStart := time.Now()
	replayRes := httptest.NewRecorder()
	server.Routes().ServeHTTP(replayRes, httptest.NewRequest(http.MethodPost, "/api/games/"+created.ID+"/replay", bytes.NewReader([]byte(`{"toSeq":0}`))))
	if replayRes.Code != http.StatusOK {
		t.Fatalf("replay status = %d body=%s", replayRes.Code, replayRes.Body.String())
	}
	if time.Since(replayBudgetStart) > time.Second {
		t.Fatalf("replay budget = %s, want <= 1s", time.Since(replayBudgetStart))
	}
}
