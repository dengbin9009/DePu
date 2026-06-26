package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dengbin9009/DePu/backend/internal/game"
	"github.com/dengbin9009/DePu/backend/internal/handeval"
	"github.com/dengbin9009/DePu/backend/internal/rules"
	"github.com/dengbin9009/DePu/backend/internal/storage"
)

type Store interface {
	Save(*game.Game) error
	Load(string) (*game.Game, error)
	SnapshotAt(string, int) (*game.Game, error)
	History(string) ([]game.Action, error)
}

type Server struct {
	store Store
}

func NewServer() *Server {
	dbPath := os.Getenv("DEPU_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "depu.db")
	}
	_ = os.MkdirAll(filepath.Dir(dbPath), 0755)
	store, err := storage.Open(dbPath)
	if err != nil {
		panic(err)
	}
	return &Server{store: store}
}

func NewServerWithStore(store Store) *Server {
	return &Server{store: store}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/rulesets", s.rulesets)
	mux.HandleFunc("/api/games", s.games)
	mux.HandleFunc("/api/games/", s.gameByID)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) rulesets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	writeJSON(w, http.StatusOK, rules.All())
}

func (s *Server) games(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	cfg := game.Config{
		RuleSetID:        req.RuleSetID,
		ButtonSeat:       req.ButtonSeat,
		SmallBlind:       req.SmallBlind,
		BigBlind:         req.BigBlind,
		BettingStructure: req.BettingStructure,
		DealMode:         game.DealMode(req.DealMode),
	}
	for _, seat := range req.Seats {
		cfg.Seats = append(cfg.Seats, game.SeatConfig{SeatNo: seat.SeatNo, Name: seat.Name, Stack: seat.Stack})
	}
	g, err := game.New(cfg)
	if err != nil {
		code, field := classifyCreateError(err)
		writeError(w, http.StatusBadRequest, code, err.Error(), field)
		return
	}
	if err := s.store.Save(g); err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusCreated, snapshot(g))
}

func (s *Server) gameByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/games/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not_found", "game not found", "")
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		g, err := s.store.Load(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "game not found", "")
			return
		}
		writeJSON(w, http.StatusOK, snapshot(g))
		return
	}
	if len(parts) >= 2 {
		switch parts[1] {
		case "actions":
			s.submitAction(w, r, id)
			return
		case "debug":
			if len(parts) >= 3 && parts[2] == "cards" {
				s.debugCards(w, r, id)
				return
			}
		case "history":
			s.history(w, r, id)
			return
		case "replay":
			s.replay(w, r, id)
			return
		}
	}
	writeError(w, http.StatusNotFound, "not_found", "route not found", "")
}

func (s *Server) debugCards(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req DebugCardsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	g, err := s.store.Load(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "game not found", "")
		return
	}
	if req.Version != g.Version {
		writeError(w, http.StatusConflict, "version_conflict", "state version conflict", "version")
		return
	}
	holeCards := map[int][]string{}
	for seatNo, cards := range req.HoleCards {
		parsed, parseErr := strconv.Atoi(seatNo)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_seat", "seat key must be numeric", "holeCards")
			return
		}
		holeCards[parsed] = cards
	}
	if err := g.SetDebugCards(holeCards, req.Board); err != nil {
		code := "invalid_card"
		if strings.Contains(err.Error(), "duplicate") {
			code = "duplicate_card"
		}
		if strings.Contains(err.Error(), "locked") {
			code = "debug_locked"
		}
		writeError(w, http.StatusBadRequest, code, err.Error(), "")
		return
	}
	if err := s.store.Save(g); err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, snapshot(g))
}

func (s *Server) submitAction(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req SubmitActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	g, err := s.store.Load(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "game not found", "")
		return
	}
	if req.Version != g.Version {
		writeError(w, http.StatusConflict, "version_conflict", "state version conflict", "version")
		return
	}
	if err := g.Apply(game.Command{SeatNo: req.SeatNo, Type: game.ActionType(req.Type), Amount: req.Amount}); err != nil {
		writeError(w, http.StatusConflict, "invalid_action", err.Error(), "")
		return
	}
	if err := s.store.Save(g); err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, snapshot(g))
}

func (s *Server) history(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	actions, err := s.store.History(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "game not found", "")
		return
	}
	writeJSON(w, http.StatusOK, actions)
}

func (s *Server) replay(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req struct {
		ToSeq int `json:"toSeq"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	g, err := s.store.SnapshotAt(id, req.ToSeq)
	if err != nil {
		writeError(w, http.StatusBadRequest, "replay_out_of_range", err.Error(), "toSeq")
		return
	}
	g.IsReplay = true
	g.Version = -g.Version
	writeJSON(w, http.StatusOK, snapshot(g))
}

type CreateGameRequest struct {
	RuleSetID        string                `json:"rulesetId"`
	ButtonSeat       int                   `json:"buttonSeat"`
	SmallBlind       int                   `json:"smallBlind"`
	BigBlind         int                   `json:"bigBlind"`
	BettingStructure game.BettingStructure `json:"bettingStructure"`
	DealMode         string                `json:"dealMode"`
	Seats            []struct {
		SeatNo int    `json:"seatNo"`
		Name   string `json:"name"`
		Stack  int    `json:"stack"`
	} `json:"seats"`
}

type SubmitActionRequest struct {
	SeatNo  int    `json:"seatNo"`
	Type    string `json:"type"`
	Amount  int    `json:"amount"`
	Version int    `json:"version"`
}

type DebugCardsRequest struct {
	Version   int                 `json:"version"`
	HoleCards map[string][]string `json:"holeCards"`
	Board     []string            `json:"board"`
}

type GameSnapshot struct {
	ID               string                `json:"id"`
	RuleSetID        string                `json:"rulesetId"`
	BettingStructure game.BettingStructure `json:"bettingStructure"`
	DealMode         game.DealMode         `json:"dealMode"`
	Stage            game.Stage            `json:"stage"`
	ButtonSeat       int                   `json:"buttonSeat"`
	CurrentSeat      int                   `json:"currentSeat"`
	CurrentBet       int                   `json:"currentBet"`
	MinRaise         int                   `json:"minRaise"`
	Board            []string              `json:"board"`
	Seats            []SeatSnapshot        `json:"seats"`
	Pots             any                   `json:"pots"`
	Showdown         any                   `json:"showdown,omitempty"`
	LegalActions     []string              `json:"legalActions"`
	IsReplay         bool                  `json:"isReplay"`
	DebugLocked      bool                  `json:"debugLocked"`
	Version          int                   `json:"version"`
}

type SeatSnapshot struct {
	game.Seat
	CurrentHand *CurrentHandSnapshot `json:"currentHand"`
}

type CurrentHandSnapshot struct {
	HandClass  rules.HandClass `json:"handClass"`
	BestCards  []string        `json:"bestCards"`
	RankVector []int           `json:"rankVector"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func snapshot(g *game.Game) GameSnapshot {
	actions := make([]string, 0)
	for _, action := range g.LegalActions() {
		actions = append(actions, string(action))
	}
	return GameSnapshot{
		ID:               g.ID,
		RuleSetID:        g.RuleSetID,
		BettingStructure: g.Betting,
		DealMode:         g.DealMode,
		Stage:            g.Stage,
		ButtonSeat:       g.ButtonSeat,
		CurrentSeat:      g.CurrentSeat,
		CurrentBet:       g.CurrentBet,
		MinRaise:         g.MinRaise,
		Board:            g.Board,
		Seats:            seatSnapshots(g),
		Pots:             g.Pots,
		Showdown:         g.Showdown,
		LegalActions:     actions,
		IsReplay:         g.IsReplay,
		DebugLocked:      g.DebugLocked,
		Version:          g.Version,
	}
}

func seatSnapshots(g *game.Game) []SeatSnapshot {
	seats := make([]SeatSnapshot, 0, len(g.Seats))
	for _, seat := range g.Seats {
		seats = append(seats, SeatSnapshot{Seat: seat, CurrentHand: currentHandForSeat(g, seat)})
	}
	return seats
}

func currentHandForSeat(g *game.Game, seat game.Seat) *CurrentHandSnapshot {
	if len(g.Board) < 3 || len(seat.HoleCards) < 2 || seat.Status == "folded" || seat.Status == "out" {
		return nil
	}
	rs, ok := rules.Get(g.RuleSetID)
	if !ok {
		return nil
	}
	cards := append([]string{}, seat.HoleCards...)
	cards = append(cards, g.Board...)
	hand, err := handeval.Evaluate(rs, cards)
	if err != nil {
		return nil
	}
	return &CurrentHandSnapshot{
		HandClass:  hand.Class,
		BestCards:  hand.BestCards,
		RankVector: hand.RankVector,
	}
}

func classifyCreateError(err error) (string, string) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "buttonSeat"):
		return "invalid_button", "buttonSeat"
	case strings.Contains(msg, "seatNo"), strings.Contains(msg, "seat stack"), strings.Contains(msg, "seats"):
		return "invalid_seat", "seats"
	case strings.Contains(msg, "player name"):
		return "invalid_player_name", "seats.name"
	case strings.Contains(msg, "betting"), strings.Contains(msg, "smallBlind"), strings.Contains(msg, "ante"):
		return "invalid_betting_structure", "bettingStructure"
	default:
		return "invalid_action", ""
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message, field string) {
	writeJSON(w, status, ErrorResponse{Code: code, Message: message, Field: field})
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
