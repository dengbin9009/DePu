package game

import "testing"

func TestShowdownSettlesAwards(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		DealMode:   DealDebug,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SetDebugCards(map[int][]string{
		1: []string{"As", "Ah"},
		2: []string{"Ks", "Kh"},
		3: []string{"Qs", "Qh"},
	}, []string{"Qd", "Qc", "2s", "3h", "4d"}); err != nil {
		t.Fatal(err)
	}
	g.Stage = StageRiver
	g.CurrentBet = 0
	for i := range g.Seats {
		g.Seats[i].StreetCommitted = 0
		g.Seats[i].HasActed = true
	}
	g.advanceStreet()
	if g.Stage != StageFinished {
		t.Fatalf("stage = %s, want finished", g.Stage)
	}
	if len(g.Showdown) != 3 {
		t.Fatalf("showdown results = %d, want 3", len(g.Showdown))
	}
	if g.Seat(3).Stack <= 900 {
		t.Fatalf("winner stack = %d, want more than post-blind stack", g.Seat(3).Stack)
	}
}

func TestDebugCardsRejectsFinishedGame(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		DealMode:   DealDebug,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.Stage = StageFinished
	if err := g.SetDebugCards(map[int][]string{1: []string{"As", "Ah"}}, nil); err == nil {
		t.Fatal("expected debug edit on finished game to fail")
	}
}
