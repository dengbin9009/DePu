package pot

import "sort"

type Contribution struct {
	SeatNo int
	Amount int
	Live   bool
}

type Pot struct {
	ID            string `json:"id"`
	Amount        int    `json:"amount"`
	EligibleSeats []int  `json:"eligibleSeats"`
}

func Build(contributions []Contribution) []Pot {
	levelsMap := map[int]bool{}
	for _, c := range contributions {
		if c.Amount > 0 {
			levelsMap[c.Amount] = true
		}
	}
	levels := make([]int, 0, len(levelsMap))
	for level := range levelsMap {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	var pots []Pot
	prev := 0
	for _, level := range levels {
		participants := 0
		var eligible []int
		for _, c := range contributions {
			if c.Amount >= level {
				participants++
				if c.Live {
					eligible = append(eligible, c.SeatNo)
				}
			}
		}
		amount := (level - prev) * participants
		if amount > 0 {
			sort.Ints(eligible)
			pots = append(pots, Pot{ID: "pot-" + itoa(len(pots)+1), Amount: amount, EligibleSeats: eligible})
		}
		prev = level
	}
	return pots
}

func SplitAward(amount int, winners []int, seatOrder []int, buttonSeat int) map[int]int {
	awards := map[int]int{}
	if len(winners) == 0 {
		return awards
	}
	base := amount / len(winners)
	remaining := amount % len(winners)
	for _, winner := range winners {
		awards[winner] = base
	}
	winnerSet := map[int]bool{}
	for _, winner := range winners {
		winnerSet[winner] = true
	}
	for _, seat := range orderFromButtonLeft(seatOrder, buttonSeat) {
		if remaining == 0 {
			break
		}
		if winnerSet[seat] {
			awards[seat]++
			remaining--
		}
	}
	return awards
}

func orderFromButtonLeft(seatOrder []int, buttonSeat int) []int {
	if len(seatOrder) == 0 {
		return nil
	}
	start := 0
	for i, seat := range seatOrder {
		if seat == buttonSeat {
			start = (i + 1) % len(seatOrder)
			break
		}
	}
	out := make([]int, 0, len(seatOrder))
	for i := 0; i < len(seatOrder); i++ {
		out = append(out, seatOrder[(start+i)%len(seatOrder)])
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
