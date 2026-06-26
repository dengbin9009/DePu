package pot

import (
	"reflect"
	"testing"
)

func TestBuildSidePots(t *testing.T) {
	pots := Build([]Contribution{
		{SeatNo: 1, Amount: 100, Live: true},
		{SeatNo: 2, Amount: 250, Live: true},
		{SeatNo: 3, Amount: 250, Live: false},
		{SeatNo: 4, Amount: 500, Live: true},
	})

	want := []Pot{
		{ID: "pot-1", Amount: 400, EligibleSeats: []int{1, 2, 4}},
		{ID: "pot-2", Amount: 450, EligibleSeats: []int{2, 4}},
		{ID: "pot-3", Amount: 250, EligibleSeats: []int{4}},
	}
	if !reflect.DeepEqual(pots, want) {
		t.Fatalf("pots = %#v, want %#v", pots, want)
	}
}

func TestSplitOddChipFromButtonLeft(t *testing.T) {
	awards := SplitAward(101, []int{2, 4}, []int{1, 2, 3, 4}, 4)
	want := map[int]int{2: 51, 4: 50}
	if !reflect.DeepEqual(awards, want) {
		t.Fatalf("awards = %#v, want %#v", awards, want)
	}
}
