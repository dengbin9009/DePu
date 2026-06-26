package handeval

import (
	"testing"

	"github.com/dengbin9009/DePu/backend/internal/rules"
)

func TestEvaluateLongDeckWheelStraight(t *testing.T) {
	rs, _ := rules.Get("long-holdem")
	hand, err := Evaluate(rs, []string{"As", "2d", "3c", "4h", "5s", "Kd", "Qh"})
	if err != nil {
		t.Fatal(err)
	}
	if hand.Class != rules.Straight {
		t.Fatalf("class = %v, want straight", hand.Class)
	}
	if hand.RankVector[0] != 5 {
		t.Fatalf("wheel high card = %d, want 5", hand.RankVector[0])
	}
}

func TestEvaluateChoosesBestFiveOfSeven(t *testing.T) {
	rs, _ := rules.Get("long-holdem")
	hand, err := Evaluate(rs, []string{"As", "Ah", "Ad", "Ac", "Kd", "Qd", "Jd"})
	if err != nil {
		t.Fatal(err)
	}
	if hand.Class != rules.FourOfAKind {
		t.Fatalf("class = %v, want four of a kind", hand.Class)
	}
	if len(hand.RankVector) != 2 || hand.RankVector[0] != 14 || hand.RankVector[1] != 13 {
		t.Fatalf("rank vector = %v, want aces with king kicker", hand.RankVector)
	}
}

func TestEvaluateShortDeckWheelStraight(t *testing.T) {
	rs, _ := rules.Get("short-deck")
	hand, err := Evaluate(rs, []string{"As", "6d", "7c", "8h", "9s", "Kd", "Qh"})
	if err != nil {
		t.Fatal(err)
	}
	if hand.Class != rules.Straight {
		t.Fatalf("class = %v, want straight", hand.Class)
	}
	if hand.RankVector[0] != 9 {
		t.Fatalf("short wheel high card = %d, want 9", hand.RankVector[0])
	}
}

func TestShortDeckFlushBeatsFullHouse(t *testing.T) {
	rs, _ := rules.Get("short-deck")
	flush, err := Evaluate(rs, []string{"As", "Ks", "Qs", "9s", "7s", "6d", "8c"})
	if err != nil {
		t.Fatal(err)
	}
	boat, err := Evaluate(rs, []string{"Ah", "Ad", "Ac", "Kh", "Kd", "6s", "7c"})
	if err != nil {
		t.Fatal(err)
	}
	if Compare(rs, flush, boat) <= 0 {
		t.Fatalf("short deck flush should beat full house: flush=%v boat=%v", flush, boat)
	}
}
