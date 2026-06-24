package game

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/handeval"
	"github.com/dengbin9009/DePu/backend/internal/pot"
	"github.com/dengbin9009/DePu/backend/internal/rules"
)

type Stage string

const (
	StageWaiting  Stage = "waiting"
	StagePreflop  Stage = "preflop"
	StageFlop     Stage = "flop"
	StageTurn     Stage = "turn"
	StageRiver    Stage = "river"
	StageShowdown Stage = "showdown"
	StageFinished Stage = "finished"
)

type ActionType string

const (
	ActionFold   ActionType = "fold"
	ActionCheck  ActionType = "check"
	ActionCall   ActionType = "call"
	ActionBet    ActionType = "bet"
	ActionRaise  ActionType = "raise"
	ActionAllIn  ActionType = "all_in"
	ActionDeal   ActionType = "deal"
	ActionPost   ActionType = "post"
	ActionSet    ActionType = "debug_set_cards"
	ActionSettle ActionType = "settle"
)

type DealMode string

const (
	DealRandom DealMode = "random"
	DealDebug  DealMode = "debug"
)

type Config struct {
	RuleSetID  string
	ButtonSeat int
	SmallBlind int
	BigBlind   int
	Seats      []SeatConfig
	DealMode   DealMode
}

type SeatConfig struct {
	SeatNo int
	Name   string
	Stack  int
}

type Game struct {
	ID          string           `json:"id"`
	RuleSetID   string           `json:"rulesetId"`
	Stage       Stage            `json:"stage"`
	ButtonSeat  int              `json:"buttonSeat"`
	SmallBlind  int              `json:"smallBlind"`
	BigBlind    int              `json:"bigBlind"`
	Deck        []string         `json:"deck"`
	Board       []string         `json:"board"`
	CurrentSeat int              `json:"currentSeat"`
	MinRaise    int              `json:"minRaise"`
	CurrentBet  int              `json:"currentBet"`
	Seats       []Seat           `json:"seats"`
	Pots        []pot.Pot        `json:"pots"`
	Showdown    []ShowdownResult `json:"showdown"`
	Actions     []Action         `json:"actions"`
	Version     int              `json:"version"`
	DealMode    DealMode         `json:"dealMode"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type Seat struct {
	SeatNo          int      `json:"seatNo"`
	Name            string   `json:"name"`
	Stack           int      `json:"stack"`
	HoleCards       []string `json:"holeCards"`
	Status          string   `json:"status"`
	StreetCommitted int      `json:"streetCommitted"`
	HandCommitted   int      `json:"handCommitted"`
	HasActed        bool     `json:"hasActed"`
}

type Action struct {
	Seq          int            `json:"seq"`
	Stage        Stage          `json:"stage"`
	SeatNo       int            `json:"seatNo"`
	Type         ActionType     `json:"type"`
	Amount       int            `json:"amount"`
	Payload      map[string]any `json:"payload"`
	StateSummary string         `json:"summary"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type ShowdownResult struct {
	SeatNo     int             `json:"seatNo"`
	BestCards  []string        `json:"bestCards"`
	HandClass  rules.HandClass `json:"handClass"`
	RankVector []int           `json:"rankVector"`
	Awards     map[string]int  `json:"awards"`
}

type Command struct {
	SeatNo int
	Type   ActionType
	Amount int
}

func New(cfg Config) (*Game, error) {
	rs, ok := rules.Get(cfg.RuleSetID)
	if !ok {
		return nil, errors.New("unknown ruleset")
	}
	if len(cfg.Seats) < 2 || len(cfg.Seats) > 10 {
		return nil, errors.New("seats must be between 2 and 10")
	}
	if cfg.DealMode == "" {
		cfg.DealMode = DealRandom
	}
	now := time.Now().UTC()
	g := &Game{
		ID:         fmt.Sprintf("game-%d", now.UnixNano()),
		RuleSetID:  cfg.RuleSetID,
		Stage:      StagePreflop,
		ButtonSeat: cfg.ButtonSeat,
		SmallBlind: cfg.SmallBlind,
		BigBlind:   cfg.BigBlind,
		MinRaise:   cfg.BigBlind,
		CurrentBet: cfg.BigBlind,
		DealMode:   cfg.DealMode,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	for _, seat := range cfg.Seats {
		if seat.Stack <= 0 {
			return nil, errors.New("seat stack must be positive")
		}
		g.Seats = append(g.Seats, Seat{SeatNo: seat.SeatNo, Name: seat.Name, Stack: seat.Stack, Status: "active"})
	}
	sort.Slice(g.Seats, func(i, j int) bool { return g.Seats[i].SeatNo < g.Seats[j].SeatNo })
	g.Deck = rs.Deck()
	rand.New(rand.NewSource(1)).Shuffle(len(g.Deck), func(i, j int) {
		g.Deck[i], g.Deck[j] = g.Deck[j], g.Deck[i]
	})
	if cfg.DealMode == DealRandom {
		for i := range g.Seats {
			g.Seats[i].HoleCards = []string{g.draw(), g.draw()}
		}
		g.record(0, ActionDeal, 0, map[string]any{"mode": "random"})
	}
	if err := g.postBlinds(); err != nil {
		return nil, err
	}
	g.CurrentSeat = g.preflopFirstActor()
	g.refreshPots()
	return g, nil
}

func (g *Game) Seat(seatNo int) *Seat {
	for i := range g.Seats {
		if g.Seats[i].SeatNo == seatNo {
			return &g.Seats[i]
		}
	}
	return nil
}

func (g *Game) Apply(cmd Command) error {
	if g.Stage == StageFinished {
		return errors.New("game is finished")
	}
	seat := g.Seat(cmd.SeatNo)
	if seat == nil {
		return errors.New("unknown seat")
	}
	if seat.SeatNo != g.CurrentSeat {
		return errors.New("not current actor")
	}
	if seat.Status != "active" {
		return errors.New("seat cannot act")
	}

	switch cmd.Type {
	case ActionFold:
		seat.Status = "folded"
		seat.HasActed = true
		g.record(seat.SeatNo, ActionFold, 0, nil)
	case ActionCheck:
		if seat.StreetCommitted != g.CurrentBet {
			return errors.New("cannot check facing bet")
		}
		seat.HasActed = true
		g.record(seat.SeatNo, ActionCheck, 0, nil)
	case ActionCall:
		toCall := g.CurrentBet - seat.StreetCommitted
		if toCall <= 0 {
			return errors.New("nothing to call")
		}
		g.commit(seat, min(toCall, seat.Stack))
		seat.HasActed = true
		g.record(seat.SeatNo, ActionCall, toCall, nil)
	case ActionBet:
		if g.CurrentBet != 0 {
			return errors.New("cannot bet facing bet")
		}
		if cmd.Amount < g.BigBlind {
			return errors.New("bet below big blind")
		}
		g.commit(seat, cmd.Amount)
		g.CurrentBet = seat.StreetCommitted
		g.MinRaise = cmd.Amount
		g.resetActedExcept(seat.SeatNo)
		seat.HasActed = true
		g.record(seat.SeatNo, ActionBet, cmd.Amount, nil)
	case ActionRaise:
		if cmd.Amount <= g.CurrentBet {
			return errors.New("raise must exceed current bet")
		}
		raiseBy := cmd.Amount - g.CurrentBet
		if raiseBy < g.MinRaise && cmd.Amount < seat.StreetCommitted+seat.Stack {
			return errors.New("raise below min raise")
		}
		g.commit(seat, cmd.Amount-seat.StreetCommitted)
		if raiseBy >= g.MinRaise {
			g.MinRaise = raiseBy
			g.resetActedExcept(seat.SeatNo)
		}
		g.CurrentBet = seat.StreetCommitted
		seat.HasActed = true
		g.record(seat.SeatNo, ActionRaise, cmd.Amount, nil)
	case ActionAllIn:
		allInTo := seat.StreetCommitted + seat.Stack
		g.commit(seat, seat.Stack)
		if allInTo > g.CurrentBet {
			raiseBy := allInTo - g.CurrentBet
			if raiseBy >= g.MinRaise {
				g.MinRaise = raiseBy
				g.resetActedExcept(seat.SeatNo)
			}
			g.CurrentBet = allInTo
		}
		seat.Status = "all_in"
		seat.HasActed = true
		g.record(seat.SeatNo, ActionAllIn, allInTo, nil)
	default:
		return errors.New("unsupported action")
	}

	g.refreshPots()
	if g.onlyOneLive() {
		g.Stage = StageFinished
		g.record(0, ActionSettle, 0, map[string]any{"reason": "folds"})
		return nil
	}
	if g.bettingRoundComplete() {
		g.advanceStreet()
	} else {
		g.CurrentSeat = g.nextActorAfter(seat.SeatNo)
	}
	g.UpdatedAt = time.Now().UTC()
	return nil
}

func (g *Game) SetDebugCards(holeCards map[int][]string, board []string) error {
	if g.Stage == StageFinished || g.Stage == StageShowdown {
		return errors.New("cannot edit settled game")
	}
	rs, ok := rules.Get(g.RuleSetID)
	if !ok {
		return errors.New("unknown ruleset")
	}
	seen := map[string]bool{}
	for _, cards := range holeCards {
		if len(cards) != 2 {
			return errors.New("each seat needs two hole cards")
		}
		for _, card := range cards {
			if !rs.ContainsCard(card) {
				return errors.New("card is not in ruleset deck: " + card)
			}
			if seen[card] {
				return errors.New("duplicate card: " + card)
			}
			seen[card] = true
		}
	}
	for _, card := range board {
		if !rs.ContainsCard(card) {
			return errors.New("card is not in ruleset deck: " + card)
		}
		if seen[card] {
			return errors.New("duplicate card: " + card)
		}
		seen[card] = true
	}
	for seatNo, cards := range holeCards {
		seat := g.Seat(seatNo)
		if seat == nil {
			return errors.New("unknown seat")
		}
		seat.HoleCards = append([]string(nil), cards...)
	}
	g.Board = append([]string(nil), board...)
	g.record(0, ActionSet, 0, map[string]any{"board": board})
	return nil
}

func (g *Game) LegalActions() []ActionType {
	seat := g.Seat(g.CurrentSeat)
	if seat == nil || seat.Status != "active" {
		return nil
	}
	actions := []ActionType{ActionFold, ActionAllIn}
	if seat.StreetCommitted == g.CurrentBet {
		actions = append(actions, ActionCheck)
		if g.CurrentBet == 0 {
			actions = append(actions, ActionBet)
		}
	} else {
		actions = append(actions, ActionCall, ActionRaise)
	}
	return actions
}

func (g *Game) draw() string {
	card := g.Deck[0]
	g.Deck = g.Deck[1:]
	return card
}

func (g *Game) postBlinds() error {
	sb := g.nextSeat(g.ButtonSeat)
	bb := g.nextSeat(sb)
	if len(g.Seats) == 2 {
		sb = g.ButtonSeat
		bb = g.nextSeat(g.ButtonSeat)
	}
	if seat := g.Seat(sb); seat != nil {
		g.commit(seat, min(g.SmallBlind, seat.Stack))
		g.record(sb, ActionPost, g.SmallBlind, map[string]any{"blind": "small"})
	}
	if seat := g.Seat(bb); seat != nil {
		g.commit(seat, min(g.BigBlind, seat.Stack))
		g.record(bb, ActionPost, g.BigBlind, map[string]any{"blind": "big"})
	}
	return nil
}

func (g *Game) commit(seat *Seat, amount int) {
	if amount > seat.Stack {
		amount = seat.Stack
	}
	seat.Stack -= amount
	seat.StreetCommitted += amount
	seat.HandCommitted += amount
	if seat.Stack == 0 {
		seat.Status = "all_in"
	}
}

func (g *Game) preflopFirstActor() int {
	if len(g.Seats) == 2 {
		return g.ButtonSeat
	}
	bb := g.nextSeat(g.nextSeat(g.ButtonSeat))
	return g.nextSeat(bb)
}

func (g *Game) nextSeat(seatNo int) int {
	order := g.seatOrder()
	for i, seat := range order {
		if seat == seatNo {
			return order[(i+1)%len(order)]
		}
	}
	return order[0]
}

func (g *Game) seatOrder() []int {
	order := make([]int, 0, len(g.Seats))
	for _, seat := range g.Seats {
		order = append(order, seat.SeatNo)
	}
	sort.Ints(order)
	return order
}

func (g *Game) nextActorAfter(seatNo int) int {
	next := g.nextSeat(seatNo)
	for next != seatNo {
		seat := g.Seat(next)
		if seat != nil && seat.Status == "active" {
			return next
		}
		next = g.nextSeat(next)
	}
	return 0
}

func (g *Game) firstPostflopActor() int {
	return g.nextActorAfter(g.ButtonSeat)
}

func (g *Game) resetActedExcept(seatNo int) {
	for i := range g.Seats {
		if g.Seats[i].SeatNo != seatNo && g.Seats[i].Status == "active" {
			g.Seats[i].HasActed = false
		}
	}
}

func (g *Game) bettingRoundComplete() bool {
	for _, seat := range g.Seats {
		if seat.Status != "active" {
			continue
		}
		if !seat.HasActed || seat.StreetCommitted != g.CurrentBet {
			return false
		}
	}
	return true
}

func (g *Game) advanceStreet() {
	for i := range g.Seats {
		g.Seats[i].StreetCommitted = 0
		g.Seats[i].HasActed = false
	}
	g.CurrentBet = 0
	g.MinRaise = g.BigBlind

	switch g.Stage {
	case StagePreflop:
		g.Board = append(g.Board, g.draw(), g.draw(), g.draw())
		g.Stage = StageFlop
		g.record(0, ActionDeal, 0, map[string]any{"street": "flop", "board": g.Board})
	case StageFlop:
		g.Board = append(g.Board, g.draw())
		g.Stage = StageTurn
		g.record(0, ActionDeal, 0, map[string]any{"street": "turn", "board": g.Board})
	case StageTurn:
		g.Board = append(g.Board, g.draw())
		g.Stage = StageRiver
		g.record(0, ActionDeal, 0, map[string]any{"street": "river", "board": g.Board})
	case StageRiver:
		g.Stage = StageShowdown
		g.settleShowdown()
		g.record(0, ActionSettle, 0, map[string]any{"reason": "showdown"})
		g.Stage = StageFinished
	}
	g.CurrentSeat = g.firstPostflopActor()
}

func (g *Game) settleShowdown() {
	rs, ok := rules.Get(g.RuleSetID)
	if !ok {
		return
	}
	results := map[int]handeval.Hand{}
	for _, seat := range g.Seats {
		if seat.Status == "folded" || seat.Status == "out" {
			continue
		}
		cards := append([]string{}, seat.HoleCards...)
		cards = append(cards, g.Board...)
		if len(cards) < 5 {
			continue
		}
		hand, err := handeval.Evaluate(rs, cards)
		if err == nil {
			results[seat.SeatNo] = hand
		}
	}
	seatOrder := g.seatOrder()
	awardsBySeat := map[int]map[string]int{}
	for _, p := range g.Pots {
		var winners []int
		var best handeval.Hand
		hasBest := false
		for _, seatNo := range p.EligibleSeats {
			hand, ok := results[seatNo]
			if !ok {
				continue
			}
			if !hasBest || handeval.Compare(rs, hand, best) > 0 {
				best = hand
				winners = []int{seatNo}
				hasBest = true
			} else if handeval.Compare(rs, hand, best) == 0 {
				winners = append(winners, seatNo)
			}
		}
		for seatNo, amount := range pot.SplitAward(p.Amount, winners, seatOrder, g.ButtonSeat) {
			if awardsBySeat[seatNo] == nil {
				awardsBySeat[seatNo] = map[string]int{}
			}
			awardsBySeat[seatNo][p.ID] = amount
			if seat := g.Seat(seatNo); seat != nil {
				seat.Stack += amount
			}
		}
	}
	g.Showdown = nil
	for seatNo, hand := range results {
		g.Showdown = append(g.Showdown, ShowdownResult{
			SeatNo:     seatNo,
			BestCards:  hand.BestCards,
			HandClass:  hand.Class,
			RankVector: hand.RankVector,
			Awards:     awardsBySeat[seatNo],
		})
	}
	sort.Slice(g.Showdown, func(i, j int) bool { return g.Showdown[i].SeatNo < g.Showdown[j].SeatNo })
}

func (g *Game) onlyOneLive() bool {
	live := 0
	for _, seat := range g.Seats {
		if seat.Status == "active" || seat.Status == "all_in" {
			live++
		}
	}
	return live == 1
}

func (g *Game) refreshPots() {
	contribs := make([]pot.Contribution, 0, len(g.Seats))
	for _, seat := range g.Seats {
		contribs = append(contribs, pot.Contribution{SeatNo: seat.SeatNo, Amount: seat.HandCommitted, Live: seat.Status != "folded" && seat.Status != "out"})
	}
	g.Pots = pot.Build(contribs)
}

func (g *Game) record(seatNo int, typ ActionType, amount int, payload map[string]any) {
	g.Version++
	g.Actions = append(g.Actions, Action{
		Seq:          len(g.Actions) + 1,
		Stage:        g.Stage,
		SeatNo:       seatNo,
		Type:         typ,
		Amount:       amount,
		Payload:      payload,
		StateSummary: fmt.Sprintf("%s:%d", g.Stage, g.Version),
		CreatedAt:    time.Now().UTC(),
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
