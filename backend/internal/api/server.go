package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
	CreateUser(username, passwordHash, nickname string, initialBalance int) (*storage.UserRecord, *storage.WalletRecord, error)
	FindUserByUsername(username string) (*storage.UserRecord, error)
	FindUserByID(userID string) (*storage.UserRecord, error)
	UpdateNickname(userID, nickname string) error
	UserStats(userID string) (*storage.UserStatsRecord, error)
	WalletByUserID(userID string) (*storage.WalletRecord, error)
	ListWalletTransactions(userID string, limit int) ([]storage.WalletTransactionRecord, error)
	AddWalletTransaction(userID, typ string, amount int, referenceType, referenceID, note string) (*storage.WalletRecord, *storage.WalletTransactionRecord, error)
	CreateRoom(ownerUserID, ruleSetID string, seatCount, minPlayersToStart int) (*storage.RoomRecord, error)
	JoinRoomByInviteCode(userID, inviteCode string) (*storage.RoomRecord, error)
	RoomByID(roomID string) (*storage.RoomRecord, error)
	TakeSeat(roomID, userID string, seatNo, buyInChips int) (*storage.RoomRecord, error)
	LeaveSeat(roomID, userID string, seatNo int) (*storage.RoomRecord, error)
	SetRoomCurrentGame(roomID, gameID string) error
	ArchiveHandResult(roomID string, g *game.Game) error
	RecentHandResultsByRoom(roomID string, limit int) ([]storage.HandResultRecord, error)
	UserHands(userID string, limit int) ([]storage.UserHandRecord, error)
}

type Server struct {
	store    Store
	sessions map[string]string
	hub      *socketHub
	mu       sync.RWMutex
}

func NewServer() *Server {
	driver := strings.TrimSpace(os.Getenv("DEPU_DB_DRIVER"))
	dsn := strings.TrimSpace(os.Getenv("DEPU_DSN"))
	if driver == "" {
		driver = string(storage.DriverMySQL)
	}
	if dsn == "" && driver == string(storage.DriverMySQL) {
		dsn = "root@tcp(127.0.0.1:3306)/depu_multiplayer?parseTime=true&multiStatements=true"
	}
	store, err := storage.OpenWithConfig(storage.Config{Driver: storage.Driver(driver), DSN: dsn})
	if err != nil {
		// fallback for existing local setups that still rely on sqlite path
		if dbPath := strings.TrimSpace(os.Getenv("DEPU_DB_PATH")); dbPath != "" {
			store, err = storage.Open(dbPath)
		}
		if err != nil {
			panic(err)
		}
	}
	return &Server{store: store, sessions: map[string]string{}, hub: newSocketHub()}
}

func NewServerWithStore(store Store) *Server {
	return &Server{store: store, sessions: map[string]string{}, hub: newSocketHub()}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/api/rulesets", s.rulesets)
	mux.HandleFunc("/api/auth/register", s.register)
	mux.HandleFunc("/api/auth/login", s.login)
	mux.HandleFunc("/api/me", s.me)
	mux.HandleFunc("/api/me/profile", s.updateProfile)
	mux.HandleFunc("/api/me/wallet", s.wallet)
	mux.HandleFunc("/api/me/hands", s.myHands)
	mux.HandleFunc("/api/recharge/options", s.rechargeOptions)
	mux.HandleFunc("/api/recharge", s.recharge)
	mux.HandleFunc("/api/socket", s.socketEndpoint)
	mux.HandleFunc("/api/rooms", s.rooms)
	mux.HandleFunc("/api/rooms/join", s.joinRoom)
	mux.HandleFunc("/api/rooms/", s.roomByID)
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

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type updateProfileRequest struct {
	Nickname string `json:"nickname"`
}

func (s *Server) hashPassword(password string) string {
	sum := sha256.Sum256([]byte("depu:" + password))
	return hex.EncodeToString(sum[:])
}

func (s *Server) createSession(userID string) string {
	token := fmt.Sprintf("tok_%d", time.Now().UTC().UnixNano())
	s.mu.Lock()
	s.sessions[token] = userID
	s.mu.Unlock()
	return token
}

func (s *Server) requireUser(r *http.Request) (*storage.UserRecord, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, errors.New("missing bearer token")
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	s.mu.RLock()
	userID, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("invalid session")
	}
	return s.store.FindUserByID(userID)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "invalid_password", "password must be at least 8 characters", "password")
		return
	}
	if _, err := s.store.FindUserByUsername(req.Username); err == nil {
		writeError(w, http.StatusConflict, "duplicate_username", "username already exists", "username")
		return
	}
	if _, _, err := s.store.CreateUser(req.Username, s.hashPassword(req.Password), req.Nickname, 5000); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "nickname") || strings.Contains(msg, "UNIQUE") {
			writeError(w, http.StatusConflict, "duplicate_nickname", "nickname already exists", "nickname")
			return
		}
		writeError(w, http.StatusInternalServerError, "storage_error", msg, "")
		return
	}
	user, _ := s.store.FindUserByUsername(req.Username)
	token := s.createSession(user.ID)
	writeJSON(w, http.StatusCreated, map[string]any{"user": map[string]any{"id": user.ID, "username": user.Username, "nickname": user.Nickname}, "token": token})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	user, err := s.store.FindUserByUsername(req.Username)
	if err != nil || user.PasswordHash != s.hashPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid username or password", "")
		return
	}
	token := s.createSession(user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"user": map[string]any{"id": user.ID, "username": user.Username, "nickname": user.Nickname}, "token": token})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	wallet, err := s.store.WalletByUserID(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	stats, err := s.store.UserStats(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "username": user.Username, "nickname": user.Nickname, "walletBalance": wallet.Balance, "handsPlayed": stats.HandsPlayed, "totalProfit": stats.TotalProfit, "lastPlayedAt": stats.LastPlayedAt})
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	if err := s.store.UpdateNickname(user.ID, req.Nickname); err != nil {
		if strings.Contains(err.Error(), "nickname") || strings.Contains(err.Error(), "UNIQUE") {
			writeError(w, http.StatusConflict, "duplicate_nickname", "nickname already exists", "nickname")
			return
		}
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	updated, _ := s.store.FindUserByID(user.ID)
	wallet, _ := s.store.WalletByUserID(user.ID)
	stats, _ := s.store.UserStats(user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"id": updated.ID, "username": updated.Username, "nickname": updated.Nickname, "walletBalance": wallet.Balance, "handsPlayed": stats.HandsPlayed, "totalProfit": stats.TotalProfit, "lastPlayedAt": stats.LastPlayedAt})
}

func (s *Server) wallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	wallet, err := s.store.WalletByUserID(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	txns, err := s.store.ListWalletTransactions(user.ID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"balance": wallet.Balance, "transactions": txns})
}

func (s *Server) rechargeOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"options": []map[string]any{{"code": "small", "label": "小额金币包", "amount": 1000}, {"code": "medium", "label": "中额金币包", "amount": 5000}, {"code": "large", "label": "大额金币包", "amount": 10000}}})
}

func (s *Server) recharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	var req struct {
		OptionCode string `json:"optionCode"`
		Confirm    bool   `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	amounts := map[string]int{"small": 1000, "medium": 5000, "large": 10000}
	amount, ok := amounts[req.OptionCode]
	if !ok || !req.Confirm {
		writeError(w, http.StatusBadRequest, "invalid_recharge", "invalid recharge request", "optionCode")
		return
	}
	wallet, txn, err := s.store.AddWalletTransaction(user.ID, "recharge_simulated", amount, "recharge", req.OptionCode, "simulated recharge")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "simulated_success", "walletBalance": wallet.Balance, "transaction": txn})
}

func (s *Server) rooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	var req struct {
		RuleSetID         string `json:"ruleSetId"`
		SeatCount         int    `json:"seatCount"`
		MinPlayersToStart int    `json:"minPlayersToStart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	room, err := s.store.CreateRoom(user.ID, req.RuleSetID, req.SeatCount, req.MinPlayersToStart)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	var req struct {
		InviteCode string `json:"inviteCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
		return
	}
	room, err := s.store.JoinRoomByInviteCode(user.ID, req.InviteCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "invalid_invite_code", "invalid invite code", "inviteCode")
			return
		}
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, room)
}

func (s *Server) roomByID(w http.ResponseWriter, r *http.Request) {
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	_ = user
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/rooms/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "room_not_found", "room not found", "")
		return
	}
	roomID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		room, err := s.store.RoomByID(roomID)
		if err != nil {
			writeError(w, http.StatusNotFound, "room_not_found", "room not found", "")
			return
		}
		writeJSON(w, http.StatusOK, room)
		return
	}
	if len(parts) == 2 && parts[1] == "start" && r.Method == http.MethodPost {
		currentUser, err := s.requireUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
			return
		}
		room, err := s.store.RoomByID(roomID)
		if err != nil {
			writeError(w, http.StatusNotFound, "room_not_found", "room not found", "")
			return
		}
		if room.OwnerUserID != currentUser.ID {
			writeError(w, http.StatusForbidden, "not_room_owner", "only room owner can start", "")
			return
		}
		var seats []game.SeatConfig
		for _, seat := range room.Seats {
			if seat.UserID != nil && seat.Nickname != nil && seat.BuyInChips != nil {
				seats = append(seats, game.SeatConfig{SeatNo: seat.SeatNo, Name: *seat.Nickname, Stack: *seat.BuyInChips})
			}
		}
		if len(seats) < room.MinPlayersToStart {
			writeError(w, http.StatusForbidden, "insufficient_coins", "not enough players to start", "")
			return
		}
		g, err := game.New(game.Config{RuleSetID: room.RuleSetID, ButtonSeat: 1, SmallBlind: 50, BigBlind: 100, Seats: seats, DealMode: game.DealRandom})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		if err := s.store.Save(g); err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		if err := s.store.SetRoomCurrentGame(roomID, g.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"roomId": roomID, "handId": g.ID, "status": string(g.Stage), "currentSeat": g.CurrentSeat, "pot": g.CurrentBet, "boardCards": g.Board, "players": snapshot(g).Seats, "availableActions": g.LegalActions()})
		return
	}
	if len(parts) == 2 && parts[1] == "current-hand" && r.Method == http.MethodGet {
		room, err := s.store.RoomByID(roomID)
		if err != nil || room.CurrentGameID == "" {
			writeError(w, http.StatusNotFound, "room_not_found", "current hand not found", "")
			return
		}
		g, err := s.store.Load(room.CurrentGameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "room_not_found", "current hand not found", "")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"roomId": roomID, "handId": g.ID, "status": string(g.Stage), "currentSeat": g.CurrentSeat, "pot": g.CurrentBet, "boardCards": g.Board, "players": snapshot(g).Seats, "availableActions": g.LegalActions()})
		return
	}
	if len(parts) == 2 && parts[1] == "actions" && r.Method == http.MethodPost {
		currentUser, err := s.requireUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
			return
		}
		room, err := s.store.RoomByID(roomID)
		if err != nil || room.CurrentGameID == "" {
			writeError(w, http.StatusNotFound, "room_not_found", "current hand not found", "")
			return
		}
		g, err := s.store.Load(room.CurrentGameID)
		if err != nil {
			writeError(w, http.StatusNotFound, "room_not_found", "current hand not found", "")
			return
		}
		allowed := false
		for _, seat := range room.Seats {
			if seat.SeatNo == g.CurrentSeat && seat.UserID != nil && *seat.UserID == currentUser.ID {
				allowed = true
				break
			}
		}
		if !allowed {
			writeError(w, http.StatusForbidden, "not_your_turn", "not your turn", "")
			return
		}
		var req struct {
			Action string `json:"action"`
			Amount int    `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
			return
		}
		if err := g.Apply(game.Command{SeatNo: g.CurrentSeat, Type: game.ActionType(req.Action), Amount: req.Amount}); err != nil {
			writeError(w, http.StatusConflict, "invalid_action", err.Error(), "")
			return
		}
		if g.Stage == game.StageFinished {
			if err := s.store.ArchiveHandResult(roomID, g); err != nil {
				writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
				return
			}
		}
		if err := s.store.Save(g); err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"roomId": roomID, "handId": g.ID, "status": string(g.Stage), "currentSeat": g.CurrentSeat, "pot": g.CurrentBet, "boardCards": g.Board, "players": snapshot(g).Seats, "availableActions": g.LegalActions()})
		return
	}
	if len(parts) == 3 && parts[1] == "hands" && parts[2] == "recent" && r.Method == http.MethodGet {
		if _, err := s.store.RoomByID(roomID); err != nil {
			writeError(w, http.StatusNotFound, "room_not_found", "room not found", "")
			return
		}
		items, err := s.store.RecentHandResultsByRoom(roomID, 10)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
		return
	}
	if len(parts) == 3 && parts[1] == "seats" && r.Method == http.MethodDelete {
		seatNo, _ := strconv.Atoi(parts[2])
		currentUser, err := s.requireUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
			return
		}
		room, err := s.store.LeaveSeat(roomID, currentUser.ID, seatNo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		writeJSON(w, http.StatusOK, room)
		return
	}
	if len(parts) == 3 && parts[1] == "seats" && r.Method == http.MethodPost {
		seatNo, _ := strconv.Atoi(parts[2])
		var req struct {
			BuyInChips int `json:"buyInChips"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_json", err.Error(), "")
			return
		}
		currentUser, err := s.requireUser(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
			return
		}
		room, err := s.store.TakeSeat(roomID, currentUser.ID, seatNo, req.BuyInChips)
		if err != nil {
			if strings.Contains(err.Error(), "taken") {
				writeError(w, http.StatusConflict, "seat_taken", "seat already taken", "seatNo")
				return
			}
			if strings.Contains(err.Error(), "insufficient") {
				writeError(w, http.StatusConflict, "insufficient_coins", "insufficient coins", "buyInChips")
				return
			}
			writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
			return
		}
		writeJSON(w, http.StatusOK, room)
		return
	}
	writeError(w, http.StatusNotFound, "room_not_found", "route not found", "")
}

func (s *Server) myHands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	user, err := s.requireUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", "")
		return
	}
	items, err := s.store.UserHands(user.ID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "storage_error", err.Error(), "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
