package game

import "testing"

func TestShortDeckAnteShowdownAwardsMainPotOddChipAndReturnsUncalledButtonBlind(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "short-deck",
		ButtonSeat: 3,
		BettingStructure: BettingStructure{
			Type:        BettingAnte,
			Ante:        1,
			ButtonBlind: 1,
		},
		DealMode: DealDebug,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "A", Stack: 100},
			{SeatNo: 2, Name: "B", Stack: 100},
			{SeatNo: 3, Name: "BTN", Stack: 100},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SetDebugCards(map[int][]string{
		1: []string{"As", "Ad"},
		2: []string{"Ah", "Ac"},
		3: []string{"Ks", "Kh"},
	}, []string{"6c", "8d", "Ts", "Qh", "7c"}); err != nil {
		t.Fatal(err)
	}
	g.Stage = StageRiver
	g.CurrentBet = 0
	g.refreshPots()
	g.settleShowdown()

	var oneAwards, twoAwards, threeAwards int
	for _, result := range g.Showdown {
		if result.SeatNo == 1 {
			oneAwards = sumAwards(result.PotAwards)
		}
		if result.SeatNo == 2 {
			twoAwards = sumAwards(result.PotAwards)
		}
		if result.SeatNo == 3 {
			threeAwards = sumAwards(result.PotAwards)
		}
	}
	if oneAwards != 2 || twoAwards != 1 {
		t.Fatalf("main pot odd split = %d/%d, want 2/1", oneAwards, twoAwards)
	}
	if threeAwards != 1 {
		t.Fatalf("uncalled buttonBlind side pot return = %d, want 1", threeAwards)
	}
}

func TestShortDeckAnteFoldWinAwardsPot(t *testing.T) {
	g, err := New(Config{
		RuleSetID:  "short-deck",
		ButtonSeat: 1,
		BettingStructure: BettingStructure{
			Type:        BettingAnte,
			Ante:        10,
			ButtonBlind: 50,
		},
		DealMode: DealRandom,
		Seats: []SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "A", Stack: 1000},
			{SeatNo: 3, Name: "B", Stack: 1000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(Command{SeatNo: 2, Type: ActionFold}); err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(Command{SeatNo: 3, Type: ActionFold}); err != nil {
		t.Fatal(err)
	}
	if g.Stage != StageFinished {
		t.Fatalf("stage = %s, want finished", g.Stage)
	}
	if g.Seat(1).Stack != 1020 {
		t.Fatalf("winner stack = %d, want 1020 after collecting ante pot", g.Seat(1).Stack)
	}
	if len(g.Showdown) != 1 || sumAwards(g.Showdown[0].PotAwards) != 80 {
		t.Fatalf("fold settlement = %#v, want one 80-chip award", g.Showdown)
	}
}

func sumAwards(awards map[string]int) int {
	total := 0
	for _, amount := range awards {
		total += amount
	}
	return total
}
