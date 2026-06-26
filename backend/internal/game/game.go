package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
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
	ActionPost   ActionType = "forced_bet"
	ActionSet    ActionType = "debug_set_cards"
	ActionSettle ActionType = "settle"
)

type DealMode string

const (
	DealRandom DealMode = "random"
	DealDebug  DealMode = "debug"
)

type Config struct {
	RuleSetID        string
	ButtonSeat       int
	SmallBlind       int
	BigBlind         int
	BettingStructure BettingStructure
	Seats            []SeatConfig
	DealMode         DealMode
}

type BettingStructureType = rules.BettingStructureType

const (
	BettingBlinds BettingStructureType = rules.BettingBlinds
	BettingAnte   BettingStructureType = rules.BettingAnte
)

type BettingStructure struct {
	Type        BettingStructureType `json:"type"`
	SmallBlind  int                  `json:"smallBlind,omitempty"`
	BigBlind    int                  `json:"bigBlind,omitempty"`
	Ante        int                  `json:"ante,omitempty"`
	ButtonBlind int                  `json:"buttonBlind,omitempty"`
}

type SeatConfig struct {
	SeatNo int
	Name   string
	Stack  int
}

type Game struct {
	ID                  string           `json:"id"`
	RuleSetID           string           `json:"rulesetId"`
	Betting             BettingStructure `json:"bettingStructure"`
	Stage               Stage            `json:"stage"`
	ButtonSeat          int              `json:"buttonSeat"`
	SmallBlind          int              `json:"smallBlind"`
	BigBlind            int              `json:"bigBlind"`
	Deck                []string         `json:"deck"`
	Board               []string         `json:"board"`
	CurrentSeat         int              `json:"currentSeat"`
	MinRaise            int              `json:"minRaise"`
	CurrentBet          int              `json:"currentBet"`
	Seats               []Seat           `json:"seats"`
	Pots                []pot.Pot        `json:"pots"`
	Showdown            []ShowdownResult `json:"showdown"`
	Actions             []Action         `json:"actions"`
	InitialSnapshotJSON string           `json:"-"`
	IsReplay            bool             `json:"isReplay"`
	DebugLocked         bool             `json:"debugLocked"`
	Version             int              `json:"version"`
	DealMode            DealMode         `json:"dealMode"`
	CreatedAt           time.Time        `json:"createdAt"`
	UpdatedAt           time.Time        `json:"updatedAt"`
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
	StateSummary StateSummary   `json:"stateSummary"`
	SnapshotJSON string         `json:"-"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type StateSummary struct {
	Stage       Stage    `json:"stage"`
	CurrentSeat int      `json:"currentSeat,omitempty"`
	CurrentBet  int      `json:"currentBet"`
	PotTotal    int      `json:"potTotal"`
	Board       []string `json:"board"`
	ActiveSeats []int    `json:"activeSeats"`
	AllInSeats  []int    `json:"allInSeats"`
	FoldedSeats []int    `json:"foldedSeats"`
	IsReplay    bool     `json:"isReplay"`
}

type ShowdownResult struct {
	SeatNo     int             `json:"seatNo"`
	BestCards  []string        `json:"bestCards"`
	HandClass  rules.HandClass `json:"handClass"`
	RankVector []int           `json:"rankVector"`
	PotAwards  map[string]int  `json:"potAwards"`
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
	if err := validateSeats(cfg.Seats, cfg.ButtonSeat); err != nil {
		return nil, err
	}
	betting, err := normalizeBettingStructure(rs, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.DealMode == "" {
		cfg.DealMode = DealRandom
	}
	now := time.Now().UTC()
	g := &Game{
		ID:         fmt.Sprintf("game-%d", now.UnixNano()),
		RuleSetID:  cfg.RuleSetID,
		Betting:    betting,
		Stage:      StagePreflop,
		ButtonSeat: cfg.ButtonSeat,
		SmallBlind: betting.SmallBlind,
		BigBlind:   betting.BigBlind,
		MinRaise:   max(1, betting.BigBlind),
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
	if err := g.postForcedBets(); err != nil {
		return nil, err
	}
	g.CurrentSeat = g.preflopFirstActor()
	g.refreshPots()
	if g.allRemainingPlayersAllIn() {
		g.runoutAndSettle("all_in")
	}
	g.InitialSnapshotJSON = g.snapshotJSON()
	return g, nil
}

func validateSeats(seats []SeatConfig, buttonSeat int) error {
	if len(seats) < 2 || len(seats) > 10 {
		return errors.New("seats must be between 2 and 10")
	}
	seenSeats := map[int]bool{}
	seenNames := map[string]bool{}
	for _, seat := range seats {
		if seat.SeatNo < 1 || seat.SeatNo > len(seats) {
			return fmt.Errorf("seatNo must be continuous 1..%d", len(seats))
		}
		if seenSeats[seat.SeatNo] {
			return errors.New("duplicate seatNo")
		}
		seenSeats[seat.SeatNo] = true
		name := strings.TrimSpace(seat.Name)
		if name == "" {
			return errors.New("player name must be non-empty")
		}
		if seenNames[name] {
			return errors.New("duplicate player name")
		}
		seenNames[name] = true
		if seat.Stack <= 0 {
			return errors.New("seat stack must be positive")
		}
	}
	if !seenSeats[buttonSeat] {
		return errors.New("buttonSeat must exist")
	}
	for i := 1; i <= len(seats); i++ {
		if !seenSeats[i] {
			return fmt.Errorf("seatNo must be continuous 1..%d", len(seats))
		}
	}
	return nil
}

func normalizeBettingStructure(rs rules.RuleSet, cfg Config) (BettingStructure, error) {
	betting := cfg.BettingStructure
	if betting.Type == "" {
		betting.Type = BettingBlinds
		betting.SmallBlind = cfg.SmallBlind
		betting.BigBlind = cfg.BigBlind
	}
	if !rs.AllowsBettingStructure(rules.BettingStructureType(betting.Type)) {
		return BettingStructure{}, errors.New("betting structure is not allowed for ruleset")
	}
	switch betting.Type {
	case BettingBlinds:
		if betting.SmallBlind <= 0 {
			betting.SmallBlind = cfg.SmallBlind
		}
		if betting.BigBlind <= 0 {
			betting.BigBlind = cfg.BigBlind
		}
		if betting.SmallBlind <= 0 || betting.BigBlind <= 0 || betting.SmallBlind >= betting.BigBlind {
			return BettingStructure{}, errors.New("smallBlind must be positive and less than bigBlind")
		}
	case BettingAnte:
		if betting.Ante <= 0 || betting.ButtonBlind <= 0 {
			return BettingStructure{}, errors.New("ante and buttonBlind must be positive")
		}
	default:
		return BettingStructure{}, errors.New("unknown betting structure")
	}
	return betting, nil
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
		g.DebugLocked = true
		seat.Status = "folded"
		seat.HasActed = true
		g.record(seat.SeatNo, ActionFold, 0, nil)
	case ActionCheck:
		if seat.StreetCommitted != g.CurrentBet {
			return errors.New("cannot check facing bet")
		}
		g.DebugLocked = true
		seat.HasActed = true
		g.record(seat.SeatNo, ActionCheck, 0, nil)
	case ActionCall:
		toCall := g.CurrentBet - seat.StreetCommitted
		if toCall <= 0 {
			return errors.New("nothing to call")
		}
		g.DebugLocked = true
		g.commit(seat, min(toCall, seat.Stack))
		seat.HasActed = true
		g.record(seat.SeatNo, ActionCall, toCall, nil)
	case ActionBet:
		if g.CurrentBet != 0 {
			return errors.New("cannot bet facing bet")
		}
		if cmd.Amount < g.minimumOpenBet() {
			return errors.New("bet below minimum")
		}
		g.DebugLocked = true
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
		g.DebugLocked = true
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
		g.DebugLocked = true
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
		g.settleByFolds()
		g.Stage = StageFinished
		g.CurrentSeat = 0
		g.CurrentBet = 0
		g.record(0, ActionSettle, 0, map[string]any{"reason": "folds"})
		return nil
	}
	if g.allRemainingPlayersAllIn() {
		g.runoutAndSettle("all_in")
		g.UpdatedAt = time.Now().UTC()
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
	if g.DebugLocked {
		return errors.New("debug cards are locked")
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
	if len(board) > 5 {
		return errors.New("board cannot exceed five cards")
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
	g.rebuildDeckExcluding(seen)
	for i := range g.Seats {
		for len(g.Seats[i].HoleCards) < 2 {
			g.Seats[i].HoleCards = append(g.Seats[i].HoleCards, g.draw())
		}
	}
	g.record(0, ActionSet, 0, map[string]any{"board": board})
	g.InitialSnapshotJSON = g.snapshotJSON()
	return nil
}

func (g *Game) rebuildDeckExcluding(excluded map[string]bool) {
	rs, ok := rules.Get(g.RuleSetID)
	if !ok {
		return
	}
	g.Deck = g.Deck[:0]
	for _, card := range rs.Deck() {
		if !excluded[card] {
			g.Deck = append(g.Deck, card)
		}
	}
	rand.New(rand.NewSource(2)).Shuffle(len(g.Deck), func(i, j int) {
		g.Deck[i], g.Deck[j] = g.Deck[j], g.Deck[i]
	})
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

func (g *Game) postForcedBets() error {
	switch g.Betting.Type {
	case BettingBlinds:
		sb := g.nextSeat(g.ButtonSeat)
		bb := g.nextSeat(sb)
		if len(g.Seats) == 2 {
			sb = g.ButtonSeat
			bb = g.nextSeat(g.ButtonSeat)
		}
		if seat := g.Seat(sb); seat != nil {
			paid := g.commitForced(seat, g.Betting.SmallBlind)
			g.record(sb, ActionPost, paid, map[string]any{"blind": "small", "required": g.Betting.SmallBlind})
		}
		if seat := g.Seat(bb); seat != nil {
			paid := g.commitForced(seat, g.Betting.BigBlind)
			g.CurrentBet = max(g.CurrentBet, paid)
			g.record(bb, ActionPost, paid, map[string]any{"blind": "big", "required": g.Betting.BigBlind})
		}
	case BettingAnte:
		highest := 0
		for i := range g.Seats {
			paid := g.commitForced(&g.Seats[i], g.Betting.Ante)
			highest = max(highest, g.Seats[i].StreetCommitted)
			g.record(g.Seats[i].SeatNo, ActionPost, paid, map[string]any{"blind": "ante", "required": g.Betting.Ante})
		}
		if seat := g.Seat(g.ButtonSeat); seat != nil && seat.Stack > 0 {
			paid := g.commitForced(seat, g.Betting.ButtonBlind)
			highest = max(highest, seat.StreetCommitted)
			g.record(seat.SeatNo, ActionPost, paid, map[string]any{"blind": "buttonBlind", "required": g.Betting.ButtonBlind})
		}
		g.CurrentBet = highest
		g.MinRaise = max(1, g.Betting.ButtonBlind)
	default:
		return errors.New("unsupported betting structure")
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

func (g *Game) commitForced(seat *Seat, amount int) int {
	before := seat.Stack
	g.commit(seat, amount)
	return before - seat.Stack
}

func (g *Game) preflopFirstActor() int {
	if g.Betting.Type == BettingAnte {
		return g.nextActorAfter(g.ButtonSeat)
	}
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

func (g *Game) resetStreetState() {
	for i := range g.Seats {
		g.Seats[i].StreetCommitted = 0
		g.Seats[i].HasActed = false
	}
	g.CurrentBet = 0
	g.MinRaise = g.minimumOpenBet()
}

func (g *Game) advanceStreet() {
	g.resetStreetState()

	switch g.Stage {
	case StagePreflop:
		g.Stage = StageFlop
		g.dealBoardTo(3)
		g.record(0, ActionDeal, 0, map[string]any{"street": "flop", "board": append([]string(nil), g.Board...)})
	case StageFlop:
		g.Stage = StageTurn
		g.dealBoardTo(4)
		g.record(0, ActionDeal, 0, map[string]any{"street": "turn", "board": append([]string(nil), g.Board...)})
	case StageTurn:
		g.Stage = StageRiver
		g.dealBoardTo(5)
		g.record(0, ActionDeal, 0, map[string]any{"street": "river", "board": append([]string(nil), g.Board...)})
	case StageRiver:
		g.Stage = StageShowdown
		g.settleShowdown()
		g.record(0, ActionSettle, 0, map[string]any{"reason": "showdown"})
		g.Stage = StageFinished
	}
	if g.Stage == StageFinished {
		g.CurrentSeat = 0
	} else {
		g.CurrentSeat = g.firstPostflopActor()
	}
}

func (g *Game) dealBoardTo(target int) {
	for len(g.Board) < target {
		g.Board = append(g.Board, g.draw())
	}
}

func (g *Game) runoutAndSettle(reason string) {
	g.resetStreetState()
	g.refreshPots()
	if len(g.Board) < 3 {
		g.Stage = StageFlop
		g.dealBoardTo(3)
		g.record(0, ActionDeal, 0, map[string]any{"street": "flop", "board": append([]string(nil), g.Board...)})
	}
	if len(g.Board) < 4 {
		g.Stage = StageTurn
		g.dealBoardTo(4)
		g.record(0, ActionDeal, 0, map[string]any{"street": "turn", "board": append([]string(nil), g.Board...)})
	}
	if len(g.Board) < 5 {
		g.Stage = StageRiver
		g.dealBoardTo(5)
		g.record(0, ActionDeal, 0, map[string]any{"street": "river", "board": append([]string(nil), g.Board...)})
	}
	g.Stage = StageShowdown
	g.settleShowdown()
	g.CurrentSeat = 0
	g.record(0, ActionSettle, 0, map[string]any{"reason": reason})
	g.Stage = StageFinished
	g.CurrentSeat = 0
	g.CurrentBet = 0
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
			PotAwards:  awardsBySeat[seatNo],
		})
	}
	sort.Slice(g.Showdown, func(i, j int) bool { return g.Showdown[i].SeatNo < g.Showdown[j].SeatNo })
}

func (g *Game) settleByFolds() {
	g.Showdown = nil
	for i := range g.Seats {
		if g.Seats[i].Status != "active" && g.Seats[i].Status != "all_in" {
			continue
		}
		awards := map[string]int{}
		for _, p := range g.Pots {
			if p.Amount <= 0 {
				continue
			}
			awards[p.ID] = p.Amount
			g.Seats[i].Stack += p.Amount
		}
		g.Showdown = append(g.Showdown, ShowdownResult{
			SeatNo:    g.Seats[i].SeatNo,
			PotAwards: awards,
		})
		return
	}
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

func (g *Game) allRemainingPlayersAllIn() bool {
	live := 0
	active := 0
	for _, seat := range g.Seats {
		switch seat.Status {
		case "active":
			live++
			active++
		case "all_in":
			live++
		}
	}
	return live > 1 && active == 0
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
	action := Action{
		Seq:          len(g.Actions) + 1,
		Stage:        g.Stage,
		SeatNo:       seatNo,
		Type:         typ,
		Amount:       amount,
		Payload:      payload,
		StateSummary: g.summary(),
		CreatedAt:    time.Now().UTC(),
	}
	action.SnapshotJSON = g.snapshotJSON()
	g.Actions = append(g.Actions, action)
}

func (g *Game) summary() StateSummary {
	summary := StateSummary{
		Stage:       g.Stage,
		CurrentSeat: g.CurrentSeat,
		CurrentBet:  g.CurrentBet,
		Board:       append([]string(nil), g.Board...),
		IsReplay:    g.IsReplay,
	}
	for _, p := range g.Pots {
		summary.PotTotal += p.Amount
	}
	for _, seat := range g.Seats {
		switch seat.Status {
		case "active":
			summary.ActiveSeats = append(summary.ActiveSeats, seat.SeatNo)
		case "all_in":
			summary.AllInSeats = append(summary.AllInSeats, seat.SeatNo)
		case "folded":
			summary.FoldedSeats = append(summary.FoldedSeats, seat.SeatNo)
		}
	}
	return summary
}

func (g *Game) minimumOpenBet() int {
	if g.Betting.Type == BettingAnte {
		return max(1, g.Betting.ButtonBlind)
	}
	return max(1, g.Betting.BigBlind)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (g *Game) snapshotJSON() string {
	body, err := json.Marshal(gameSnapshotBody{
		ID:          g.ID,
		RuleSetID:   g.RuleSetID,
		Betting:     g.Betting,
		Stage:       g.Stage,
		ButtonSeat:  g.ButtonSeat,
		SmallBlind:  g.SmallBlind,
		BigBlind:    g.BigBlind,
		Deck:        append([]string(nil), g.Deck...),
		Board:       append([]string(nil), g.Board...),
		CurrentSeat: g.CurrentSeat,
		MinRaise:    g.MinRaise,
		CurrentBet:  g.CurrentBet,
		Seats:       append([]Seat(nil), g.Seats...),
		Pots:        append([]pot.Pot(nil), g.Pots...),
		Showdown:    append([]ShowdownResult(nil), g.Showdown...),
		IsReplay:    g.IsReplay,
		DebugLocked: g.DebugLocked,
		Version:     g.Version,
		DealMode:    g.DealMode,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	})
	if err != nil {
		return ""
	}
	return string(body)
}

type gameSnapshotBody struct {
	ID          string           `json:"id"`
	RuleSetID   string           `json:"rulesetId"`
	Betting     BettingStructure `json:"bettingStructure"`
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
	IsReplay    bool             `json:"isReplay"`
	DebugLocked bool             `json:"debugLocked"`
	Version     int              `json:"version"`
	DealMode    DealMode         `json:"dealMode"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}
