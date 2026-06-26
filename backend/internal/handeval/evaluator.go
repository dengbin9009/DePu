package handeval

import (
	"errors"
	"sort"
	"strings"

	"github.com/dengbin9009/DePu/backend/internal/rules"
)

type Card struct {
	Rank int
	Suit string
	Code string
}

type Hand struct {
	Class      rules.HandClass `json:"class"`
	BestCards  []string        `json:"bestCards"`
	RankVector []int           `json:"rankVector"`
}

func Evaluate(rs rules.RuleSet, codes []string) (Hand, error) {
	if len(codes) < 5 {
		return Hand{}, errors.New("need at least five cards")
	}
	cards, err := parseCards(rs, codes)
	if err != nil {
		return Hand{}, err
	}

	var best Hand
	hasBest := false
	for _, combo := range combinations(cards, 5) {
		hand := evaluateFive(rs, combo)
		if !hasBest || Compare(rs, hand, best) > 0 {
			best = hand
			hasBest = true
		}
	}
	return best, nil
}

func Compare(rs rules.RuleSet, a, b Hand) int {
	if diff := rs.CompareHandClass(a.Class, b.Class); diff != 0 {
		return diff
	}
	for i := 0; i < len(a.RankVector) && i < len(b.RankVector); i++ {
		if a.RankVector[i] != b.RankVector[i] {
			return a.RankVector[i] - b.RankVector[i]
		}
	}
	return len(a.RankVector) - len(b.RankVector)
}

func parseCards(rs rules.RuleSet, codes []string) ([]Card, error) {
	seen := map[string]bool{}
	cards := make([]Card, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if !rs.ContainsCard(code) {
			return nil, errors.New("card is not in ruleset deck: " + code)
		}
		normalized := strings.ToUpper(code[:1]) + strings.ToLower(code[1:])
		if seen[normalized] {
			return nil, errors.New("duplicate card: " + normalized)
		}
		seen[normalized] = true
		cards = append(cards, Card{Rank: rs.RankValue(normalized[:1]), Suit: normalized[1:], Code: normalized})
	}
	return cards, nil
}

func combinations(cards []Card, size int) [][]Card {
	var out [][]Card
	var walk func(start int, current []Card)
	walk = func(start int, current []Card) {
		if len(current) == size {
			combo := make([]Card, size)
			copy(combo, current)
			out = append(out, combo)
			return
		}
		for i := start; i <= len(cards)-(size-len(current)); i++ {
			walk(i+1, append(current, cards[i]))
		}
	}
	walk(0, nil)
	return out
}

func evaluateFive(rs rules.RuleSet, cards []Card) Hand {
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Rank > cards[j].Rank
	})

	counts := map[int]int{}
	for _, card := range cards {
		counts[card.Rank]++
	}
	flush := true
	for i := 1; i < len(cards); i++ {
		if cards[i].Suit != cards[0].Suit {
			flush = false
			break
		}
	}
	straight, straightHigh := straightHigh(rs, cards)
	codes := make([]string, 0, len(cards))
	for _, card := range cards {
		codes = append(codes, card.Code)
	}

	if flush && straight {
		return Hand{Class: rules.StraightFlush, BestCards: codes, RankVector: []int{straightHigh}}
	}

	groups := rankGroups(counts)
	if groups[0].Count == 4 {
		return Hand{Class: rules.FourOfAKind, BestCards: codes, RankVector: []int{groups[0].Rank, groups[1].Rank}}
	}
	if groups[0].Count == 3 && groups[1].Count == 2 {
		return Hand{Class: rules.FullHouse, BestCards: codes, RankVector: []int{groups[0].Rank, groups[1].Rank}}
	}
	if flush {
		return Hand{Class: rules.Flush, BestCards: codes, RankVector: ranksDesc(cards)}
	}
	if straight {
		return Hand{Class: rules.Straight, BestCards: codes, RankVector: []int{straightHigh}}
	}
	if groups[0].Count == 3 {
		return Hand{Class: rules.ThreeOfAKind, BestCards: codes, RankVector: append([]int{groups[0].Rank}, groupRanks(groups[1:])...)}
	}
	if groups[0].Count == 2 && groups[1].Count == 2 {
		highPair, lowPair := groups[0].Rank, groups[1].Rank
		if lowPair > highPair {
			highPair, lowPair = lowPair, highPair
		}
		return Hand{Class: rules.TwoPair, BestCards: codes, RankVector: []int{highPair, lowPair, groups[2].Rank}}
	}
	if groups[0].Count == 2 {
		return Hand{Class: rules.OnePair, BestCards: codes, RankVector: append([]int{groups[0].Rank}, groupRanks(groups[1:])...)}
	}
	return Hand{Class: rules.HighCard, BestCards: codes, RankVector: ranksDesc(cards)}
}

type group struct {
	Rank  int
	Count int
}

func rankGroups(counts map[int]int) []group {
	groups := make([]group, 0, len(counts))
	for rank, count := range counts {
		groups = append(groups, group{Rank: rank, Count: count})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Count != groups[j].Count {
			return groups[i].Count > groups[j].Count
		}
		return groups[i].Rank > groups[j].Rank
	})
	return groups
}

func groupRanks(groups []group) []int {
	ranks := make([]int, 0, len(groups))
	for _, group := range groups {
		ranks = append(ranks, group.Rank)
	}
	return ranks
}

func ranksDesc(cards []Card) []int {
	ranks := make([]int, 0, len(cards))
	for _, card := range cards {
		ranks = append(ranks, card.Rank)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ranks)))
	return ranks
}

func straightHigh(rs rules.RuleSet, cards []Card) (bool, int) {
	seen := map[int]bool{}
	for _, card := range cards {
		seen[card.Rank] = true
	}
	for _, rank := range rs.Wheel {
		if !seen[rank] {
			goto normal
		}
	}
	return true, rs.Wheel[1]

normal:
	ranks := make([]int, 0, len(seen))
	for rank := range seen {
		ranks = append(ranks, rank)
	}
	sort.Ints(ranks)
	if len(ranks) != 5 {
		return false, 0
	}
	for i := 1; i < len(ranks); i++ {
		if ranks[i] != ranks[i-1]+1 {
			return false, 0
		}
	}
	return true, ranks[len(ranks)-1]
}
