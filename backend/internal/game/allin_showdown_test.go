package game

import "testing"

func TestAllInPlayersAutoRunBoardToShowdown(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		BettingStructure: BettingStructure{
			Type:       BettingBlinds,
			SmallBlind: 50,
			BigBlind:   100,
		},
		DealMode: DealRandom,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 50},
			{SeatNo: 2, Name: "SB", Stack: 50},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.Stage != StageFinished {
		t.Fatalf("short all-in heads-up stage = %s, want finished after automatic runout", g.Stage)
	}
	if len(g.Board) != 5 {
		t.Fatalf("board = %v, want 5 cards", g.Board)
	}
	if len(g.Showdown) == 0 {
		t.Fatal("expected showdown after automatic runout")
	}
}
