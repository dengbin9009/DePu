package game

import "testing"

func TestNewGamePostsBlindsAndSetsPreflopActor(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
			{SeatNo: 4, Name: "UTG", Stack: 1000},
		},
		DealMode: DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.Stage != StagePreflop {
		t.Fatalf("stage = %s, want preflop", g.Stage)
	}
	if g.CurrentSeat != 4 {
		t.Fatalf("current seat = %d, want 4", g.CurrentSeat)
	}
	if g.Seat(2).StreetCommitted != 50 || g.Seat(3).StreetCommitted != 100 {
		t.Fatalf("blind commits sb=%d bb=%d", g.Seat(2).StreetCommitted, g.Seat(3).StreetCommitted)
	}
	if len(g.Seat(1).HoleCards) != 2 {
		t.Fatalf("button hole cards = %d, want 2", len(g.Seat(1).HoleCards))
	}
	if len(g.Actions) == 0 {
		t.Fatal("new game should record initial deal/blind actions")
	}
}

func TestApplyActionsAdvancesToFlop(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
			{SeatNo: 4, Name: "UTG", Stack: 1000},
		},
		DealMode: DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []Command{
		{SeatNo: 4, Type: ActionCall},
		{SeatNo: 1, Type: ActionCall},
		{SeatNo: 2, Type: ActionCall},
		{SeatNo: 3, Type: ActionCheck},
	} {
		if err := g.Apply(action); err != nil {
			t.Fatalf("apply %#v: %v", action, err)
		}
	}
	if g.Stage != StageFlop {
		t.Fatalf("stage = %s, want flop", g.Stage)
	}
	if len(g.Board) != 3 {
		t.Fatalf("board = %v, want 3 flop cards", g.Board)
	}
	if g.CurrentSeat != 2 {
		t.Fatalf("flop first actor = %d, want 2", g.CurrentSeat)
	}
}

func TestIllegalActionDoesNotMutateVersion(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
		DealMode: DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	version := g.Version
	if err := g.Apply(Command{SeatNo: 1, Type: ActionCheck}); err == nil {
		t.Fatal("expected wrong actor action to fail")
	}
	if g.Version != version {
		t.Fatalf("version changed after illegal action: %d -> %d", version, g.Version)
	}
}
