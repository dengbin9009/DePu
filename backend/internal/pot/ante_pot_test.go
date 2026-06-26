package pot

import (
	"reflect"
	"testing"
)

func TestAnteButtonBlindAndShortStackUseActualCommittedForSidePots(t *testing.T) {
	pots := Build([]Contribution{
		{SeatNo: 1, Amount: 40, Live: true},
		{SeatNo: 2, Amount: 10, Live: true},
		{SeatNo: 3, Amount: 60, Live: true},
		{SeatNo: 4, Amount: 60, Live: false},
	})

	want := []Pot{
		{ID: "pot-1", Amount: 40, EligibleSeats: []int{1, 2, 3}},
		{ID: "pot-2", Amount: 90, EligibleSeats: []int{1, 3}},
		{ID: "pot-3", Amount: 40, EligibleSeats: []int{3}},
	}
	if !reflect.DeepEqual(pots, want) {
		t.Fatalf("pots = %#v, want %#v", pots, want)
	}
}
