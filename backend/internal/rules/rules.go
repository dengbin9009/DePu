package rules

import "strings"

type HandClass string

const (
	StraightFlush HandClass = "straight_flush"
	FourOfAKind   HandClass = "four_of_a_kind"
	FullHouse     HandClass = "full_house"
	Flush         HandClass = "flush"
	Straight      HandClass = "straight"
	ThreeOfAKind  HandClass = "three_of_a_kind"
	TwoPair       HandClass = "two_pair"
	OnePair       HandClass = "one_pair"
	HighCard      HandClass = "high_card"
)

type RuleSet struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Ranks       []string    `json:"ranks"`
	Ranking     []HandClass `json:"handRanking"`
	Wheel       []int       `json:"wheel"`
	SmallBlind  int         `json:"smallBlind"`
	BigBlind    int         `json:"bigBlind"`
	Description string      `json:"description"`
}

var suitCodes = []string{"s", "h", "d", "c"}

var ruleSets = map[string]RuleSet{
	"long-holdem": {
		ID:         "long-holdem",
		Name:       "长牌德州扑克",
		Ranks:      []string{"2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K", "A"},
		Wheel:      []int{14, 5, 4, 3, 2},
		SmallBlind: 50,
		BigBlind:   100,
		Ranking:    []HandClass{StraightFlush, FourOfAKind, FullHouse, Flush, Straight, ThreeOfAKind, TwoPair, OnePair, HighCard},
	},
	"short-deck": {
		ID:          "short-deck",
		Name:        "短牌德州扑克",
		Ranks:       []string{"6", "7", "8", "9", "T", "J", "Q", "K", "A"},
		Wheel:       []int{14, 9, 8, 7, 6},
		SmallBlind:  50,
		BigBlind:    100,
		Ranking:     []HandClass{StraightFlush, FourOfAKind, Flush, FullHouse, Straight, ThreeOfAKind, TwoPair, OnePair, HighCard},
		Description: "v1 使用小盲/大盲结构；短牌 ante + button blind 作为后续规则集扩展。",
	},
}

func All() []RuleSet {
	return []RuleSet{ruleSets["long-holdem"], ruleSets["short-deck"]}
}

func Get(id string) (RuleSet, bool) {
	rs, ok := ruleSets[id]
	return rs, ok
}

func (r RuleSet) Deck() []string {
	deck := make([]string, 0, len(r.Ranks)*len(suitCodes))
	for _, rank := range r.Ranks {
		for _, suit := range suitCodes {
			deck = append(deck, rank+suit)
		}
	}
	return deck
}

func (r RuleSet) ContainsCard(card string) bool {
	if len(card) < 2 {
		return false
	}
	card = strings.TrimSpace(card)
	for _, candidate := range r.Deck() {
		if strings.EqualFold(candidate, card) {
			return true
		}
	}
	return false
}

func (r RuleSet) RankValue(rank string) int {
	switch strings.ToUpper(rank) {
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "T":
		return 10
	case "J":
		return 11
	case "Q":
		return 12
	case "K":
		return 13
	case "A":
		return 14
	default:
		return 0
	}
}

func (r RuleSet) CompareHandClass(a, b HandClass) int {
	return classScore(r, a) - classScore(r, b)
}

func classScore(r RuleSet, class HandClass) int {
	for i, c := range r.Ranking {
		if c == class {
			return len(r.Ranking) - i
		}
	}
	return 0
}
