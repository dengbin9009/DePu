package storage

import (
	"testing"

	"github.com/dengbin9009/DePu/backend/internal/game"
)

func TestSaveLoadGameAndHistory(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	g, err := game.New(game.Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		Seats: []game.SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
		DealMode: game.DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(g); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != g.ID || loaded.Version != g.Version {
		t.Fatalf("loaded game mismatch: %#v vs %#v", loaded, g)
	}
	history, err := store.History(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != len(g.Actions) {
		t.Fatalf("history length = %d, want %d", len(history), len(g.Actions))
	}

	replayed, err := store.SnapshotAt(g.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != g.ID {
		t.Fatalf("replayed game id = %s, want %s", replayed.ID, g.ID)
	}
	if replayed.Version == 0 {
		t.Fatal("replayed snapshot should preserve a version")
	}
}

func TestSaveReturnsErrorWhenStorageUnavailable(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	g, err := game.New(game.Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		BettingStructure: game.BettingStructure{
			Type:       game.BettingBlinds,
			SmallBlind: 50,
			BigBlind:   100,
		},
		Seats: []game.SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
		DealMode: game.DealDebug,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SetDebugCards(map[int][]string{1: []string{"As", "Ah"}}, []string{"Ks", "Kh", "Qh"}); err != nil {
		t.Fatal(err)
	}
	g.Stage = game.StageRiver
	for i := range g.Seats {
		g.Seats[i].HasActed = true
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(g); err == nil {
		t.Fatal("expected save to fail when storage is unavailable")
	}
}
