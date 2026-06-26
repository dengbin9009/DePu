package game

import "testing"

func TestNewGameRejectsNonContinuousSeatsAndDuplicateNames(t *testing.T) {
	_, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		BettingStructure: BettingStructure{
			Type:       BettingBlinds,
			SmallBlind: 50,
			BigBlind:   100,
		},
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 3, Name: "BTN", Stack: 1000},
		},
		DealMode: DealRandom,
	})
	if err == nil {
		t.Fatal("expected non-continuous seats and duplicate names to fail")
	}
}

func TestShortDeckAnteStructureInitializesLiveBets(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "short-deck",
		ButtonSeat: 2,
		BettingStructure: BettingStructure{
			Type:        BettingAnte,
			Ante:        10,
			ButtonBlind: 50,
		},
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "A", Stack: 1000},
			{SeatNo: 2, Name: "B", Stack: 1000},
			{SeatNo: 3, Name: "C", Stack: 1000},
			{SeatNo: 4, Name: "D", Stack: 1000},
		},
		DealMode: DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.CurrentBet != 60 {
		t.Fatalf("current bet = %d, want ante + buttonBlind 60", g.CurrentBet)
	}
	if g.CurrentSeat != 3 {
		t.Fatalf("preflop first actor = %d, want button left seat 3", g.CurrentSeat)
	}
	for _, seat := range g.Seats {
		want := 10
		if seat.SeatNo == 2 {
			want = 60
		}
		if seat.StreetCommitted != want || seat.HandCommitted != want {
			t.Fatalf("seat %d committed street/hand = %d/%d, want %d", seat.SeatNo, seat.StreetCommitted, seat.HandCommitted, want)
		}
	}
}

func TestForcedBetShortStackGoesAllInForActualAmount(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "short-deck",
		ButtonSeat: 1,
		BettingStructure: BettingStructure{
			Type:        BettingAnte,
			Ante:        10,
			ButtonBlind: 50,
		},
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 40},
			{SeatNo: 2, Name: "A", Stack: 1000},
			{SeatNo: 3, Name: "B", Stack: 1000},
		},
		DealMode: DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	button := g.Seat(1)
	if button.Status != "all_in" {
		t.Fatalf("button status = %s, want all_in", button.Status)
	}
	if button.StreetCommitted != 40 || button.HandCommitted != 40 || button.Stack != 0 {
		t.Fatalf("button stack/committed = %d/%d/%d, want 0/40/40", button.Stack, button.StreetCommitted, button.HandCommitted)
	}
	if g.CurrentBet != 40 {
		t.Fatalf("current bet = %d, want actual highest forced bet 40", g.CurrentBet)
	}
}

func TestDebugCardsLockAfterFirstPlayerAction(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		BettingStructure: BettingStructure{
			Type:       BettingBlinds,
			SmallBlind: 50,
			BigBlind:   100,
		},
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
		DealMode: DealDebug,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(Command{SeatNo: 1, Type: ActionCall}); err != nil {
		t.Fatal(err)
	}
	if !g.DebugLocked {
		t.Fatal("expected debug cards to lock after first player action")
	}
	if err := g.SetDebugCards(map[int][]string{1: []string{"As", "Ah"}}, nil); err == nil {
		t.Fatal("expected debug card edit after player action to fail")
	}
}

func TestDebugCardsPartialAssignmentFillsMissingHoleCardsFromRemainingDeck(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "short-deck",
		ButtonSeat: 1,
		BettingStructure: BettingStructure{
			Type:        BettingAnte,
			Ante:        10,
			ButtonBlind: 50,
		},
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "A", Stack: 1000},
			{SeatNo: 3, Name: "B", Stack: 1000},
		},
		DealMode: DealDebug,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SetDebugCards(map[int][]string{1: []string{"As", "Ah"}}, []string{"Ks", "Kh"}); err != nil {
		t.Fatal(err)
	}
	if got := g.Seat(1).HoleCards; got[0] != "As" || got[1] != "Ah" {
		t.Fatalf("fixed hole cards = %v, want As Ah", got)
	}
	for _, seat := range g.Seats {
		if len(seat.HoleCards) != 2 {
			t.Fatalf("seat %d hole cards = %v, want 2 cards", seat.SeatNo, seat.HoleCards)
		}
	}
	seen := map[string]bool{}
	for _, seat := range g.Seats {
		for _, card := range seat.HoleCards {
			if seen[card] {
				t.Fatalf("duplicate hole card after fill: %s", card)
			}
			seen[card] = true
		}
	}
	for _, card := range g.Board {
		if seen[card] {
			t.Fatalf("board card duplicated in hole cards: %s", card)
		}
		seen[card] = true
	}
}
