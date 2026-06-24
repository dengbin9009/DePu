package rules

import "testing"

func TestRuleSetsExposeLongAndShortDecks(t *testing.T) {
	long, ok := Get("long-holdem")
	if !ok {
		t.Fatal("expected long-holdem ruleset")
	}
	if got := len(long.Deck()); got != 52 {
		t.Fatalf("long deck size = %d, want 52", got)
	}
	if !long.ContainsCard("2s") || !long.ContainsCard("As") {
		t.Fatal("long deck should include 2s and As")
	}

	short, ok := Get("short-deck")
	if !ok {
		t.Fatal("expected short-deck ruleset")
	}
	if got := len(short.Deck()); got != 36 {
		t.Fatalf("short deck size = %d, want 36", got)
	}
	if short.ContainsCard("2s") {
		t.Fatal("short deck should not include 2s")
	}
	if !short.ContainsCard("6s") || !short.ContainsCard("As") {
		t.Fatal("short deck should include 6s and As")
	}
	if short.CompareHandClass(Flush, FullHouse) <= 0 {
		t.Fatal("short deck should rank flush above full house")
	}
}
