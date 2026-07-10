package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/game"
	_ "github.com/go-sql-driver/mysql"
)

type Driver string

const (
	DriverMySQL Driver = "mysql"
)

type Config struct {
	Driver Driver
	DSN    string
}

type UserRecord struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Nickname     string `json:"nickname"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type UserStatsRecord struct {
	HandsPlayed  int     `json:"handsPlayed"`
	TotalProfit  int     `json:"totalProfit"`
	LastPlayedAt *string `json:"lastPlayedAt"`
}

type WalletRecord struct {
	UserID    string `json:"userId"`
	Balance   int    `json:"balance"`
	UpdatedAt string `json:"updatedAt"`
}

type WalletTransactionRecord struct {
	ID            string `json:"id"`
	UserID        string `json:"userId"`
	Type          string `json:"type"`
	Amount        int    `json:"amount"`
	BalanceAfter  int    `json:"balanceAfter"`
	ReferenceType string `json:"referenceType,omitempty"`
	ReferenceID   string `json:"referenceId,omitempty"`
	Note          string `json:"note,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

type RoomMemberRecord struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}

type RoomSeatRecord struct {
	SeatNo     int     `json:"seatNo"`
	SeatStatus string  `json:"seatStatus"`
	UserID     *string `json:"userId,omitempty"`
	Nickname   *string `json:"nickname,omitempty"`
	BuyInChips *int    `json:"buyInChips,omitempty"`
}

type RoomRecord struct {
	ID                string             `json:"id"`
	InviteCode        string             `json:"inviteCode"`
	OwnerUserID       string             `json:"ownerUserId"`
	Status            string             `json:"status"`
	RuleSetID         string             `json:"ruleSetId,omitempty"`
	Name              string             `json:"name,omitempty"`
	Mode              string             `json:"mode,omitempty"`
	Variant           string             `json:"variant,omitempty"`
	Ante              int                `json:"ante,omitempty"`
	MinBuyIn          int                `json:"minBuyIn,omitempty"`
	MaxBuyIn          int                `json:"maxBuyIn,omitempty"`
	BuyInCap          int                `json:"buyInCap,omitempty"`
	DurationMinutes   int                `json:"durationMinutes,omitempty"`
	Level             int                `json:"level,omitempty"`
	SeatCount         int                `json:"seatCount,omitempty"`
	MinPlayersToStart int                `json:"minPlayersToStart,omitempty"`
	Members           []RoomMemberRecord `json:"members"`
	Seats             []RoomSeatRecord   `json:"seats"`
	CurrentGameID     string             `json:"-"`
}

type CreateRoomOptions struct {
	RuleSetID         string
	Name              string
	Mode              string
	Variant           string
	Ante              int
	HasAnte           bool
	MinBuyIn          int
	HasMinBuyIn       bool
	MaxBuyIn          int
	HasMaxBuyIn       bool
	BuyInCap          int
	HasBuyInCap       bool
	DurationMinutes   int
	HasDuration       bool
	SeatCount         int
	HasSeatCount      bool
	MinPlayersToStart int
	HasMinPlayers     bool
}

type FieldError struct {
	Code  string
	Field string
}

func (e FieldError) Error() string { return e.Code }

func fieldError(code, field string) FieldError {
	return FieldError{Code: code, Field: field}
}

type HandParticipantRecord struct {
	UserID        string   `json:"userId"`
	Nickname      string   `json:"nickname"`
	SeatNo        int      `json:"seatNo"`
	Profit        int      `json:"profit"`
	AwardAmount   int      `json:"awardAmount"`
	HandCommitted int      `json:"handCommitted"`
	ResultType    string   `json:"resultType"`
	HoleCards     []string `json:"holeCards,omitempty"`
	BestCards     []string `json:"bestCards,omitempty"`
	HandClass     string   `json:"handClass,omitempty"`
}

type HandResultRecord struct {
	HandID         string                  `json:"handId"`
	RoomID         string                  `json:"roomId"`
	GameID         string                  `json:"gameId"`
	HandNo         int                     `json:"handNo"`
	RuleSetID      string                  `json:"ruleSetId"`
	CompletedAt    string                  `json:"completedAt"`
	WinnerSummary  string                  `json:"winnerSummary"`
	PotSummary     string                  `json:"potSummary"`
	BoardCards     []string                `json:"boardCards"`
	Participants   []HandParticipantRecord `json:"participants"`
	TotalPot       int                     `json:"totalPot"`
	WinningUserIDs []string                `json:"winningUserIds,omitempty"`
}

type UserHandRecord struct {
	HandID        string `json:"handId"`
	RoomID        string `json:"roomId"`
	HandNo        int    `json:"handNo"`
	CompletedAt   string `json:"completedAt"`
	Nickname      string `json:"nickname"`
	Profit        int    `json:"profit"`
	WinnerSummary string `json:"winnerSummary"`
}

type RoomLeaderboardRecord struct {
	UserID        string `json:"userId"`
	Nickname      string `json:"nickname"`
	HandsPlayed   int    `json:"handsPlayed"`
	HandsWon      int    `json:"handsWon"`
	NetProfit     int    `json:"netProfit"`
	BiggestPotWon int    `json:"biggestPotWon"`
	LastSettledAt string `json:"lastSettledAt"`
}

type HandReplayActionRecord struct {
	Type   string `json:"type"`
	SeatNo int    `json:"seatNo"`
	Amount int    `json:"amount"`
}

type HandReplayPlayerRecord struct {
	SeatNo        int      `json:"seatNo"`
	Nickname      string   `json:"nickname"`
	Stack         int      `json:"stack"`
	Status        string   `json:"status"`
	HandCommitted int      `json:"handCommitted"`
	HoleCards     []string `json:"holeCards,omitempty"`
}

type HandReplayStepRecord struct {
	Seq         int                      `json:"seq"`
	Stage       string                   `json:"stage"`
	CurrentSeat int                      `json:"currentSeat"`
	BoardCards  []string                 `json:"boardCards"`
	Pot         int                      `json:"pot"`
	Action      *HandReplayActionRecord  `json:"action,omitempty"`
	Players     []HandReplayPlayerRecord `json:"players"`
}

type HandReplayRecord struct {
	HandID string                 `json:"handId"`
	RoomID string                 `json:"roomId"`
	GameID string                 `json:"gameId"`
	Steps  []HandReplayStepRecord `json:"steps"`
}

type Store struct {
	db     *sql.DB
	driver Driver
}

func Open(path string) (*Store, error) {
	return OpenWithConfig(Config{Driver: DriverMySQL, DSN: path})
}

func OpenWithConfig(cfg Config) (*Store, error) {
	driver := cfg.Driver
	if driver == "" {
		driver = DriverMySQL
	}
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, errors.New("missing mysql dsn")
	}
	if driver != DriverMySQL {
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db, driver: driver}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Driver() Driver { return s.driver }
func (s *Store) Close() error   { return s.db.Close() }

func (s *Store) CreateUser(username, passwordHash, nickname string, initialBalance int) (*UserRecord, *WalletRecord, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(passwordHash) == "" || strings.TrimSpace(nickname) == "" {
		return nil, nil, errors.New("missing required user fields")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	userID := fmt.Sprintf("user_%d", time.Now().UTC().UnixNano())
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`insert into users(id, username, password_hash, status, created_at, updated_at) values(?, ?, ?, ?, ?, ?)`, userID, username, passwordHash, "active", now, now)
	if err != nil {
		return nil, nil, err
	}
	_, err = tx.Exec(`insert into user_profiles(user_id, nickname, hands_played, total_profit, last_played_at, updated_at) values(?, ?, 0, 0, null, ?)`, userID, nickname, now)
	if err != nil {
		return nil, nil, err
	}
	_, err = tx.Exec(`insert into wallets(user_id, balance, updated_at) values(?, ?, ?)`, userID, initialBalance, now)
	if err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}
	return &UserRecord{ID: userID, Username: username, PasswordHash: passwordHash, Nickname: nickname, Status: "active", CreatedAt: now, UpdatedAt: now}, &WalletRecord{UserID: userID, Balance: initialBalance, UpdatedAt: now}, nil
}

func (s *Store) FindUserByUsername(username string) (*UserRecord, error) {
	var rec UserRecord
	var nickname string
	row := s.db.QueryRow(`select u.id, u.username, u.password_hash, u.status, u.created_at, u.updated_at, p.nickname from users u join user_profiles p on p.user_id = u.id where u.username = ?`, username)
	if err := row.Scan(&rec.ID, &rec.Username, &rec.PasswordHash, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt, &nickname); err != nil {
		return nil, err
	}
	rec.Nickname = nickname
	return &rec, nil
}

func (s *Store) FindUserByID(userID string) (*UserRecord, error) {
	var rec UserRecord
	var nickname string
	row := s.db.QueryRow(`select u.id, u.username, u.password_hash, u.status, u.created_at, u.updated_at, p.nickname from users u join user_profiles p on p.user_id = u.id where u.id = ?`, userID)
	if err := row.Scan(&rec.ID, &rec.Username, &rec.PasswordHash, &rec.Status, &rec.CreatedAt, &rec.UpdatedAt, &nickname); err != nil {
		return nil, err
	}
	rec.Nickname = nickname
	return &rec, nil
}

func (s *Store) UpdateNickname(userID, nickname string) error {
	_, err := s.db.Exec(`update user_profiles set nickname = ?, updated_at = ? where user_id = ?`, nickname, time.Now().UTC().Format(time.RFC3339Nano), userID)
	return err
}

func (s *Store) UserStats(userID string) (*UserStatsRecord, error) {
	var stats UserStatsRecord
	var lastPlayedAt sql.NullString
	if err := s.db.QueryRow(`select hands_played, total_profit, last_played_at from user_profiles where user_id = ?`, userID).Scan(&stats.HandsPlayed, &stats.TotalProfit, &lastPlayedAt); err != nil {
		return nil, err
	}
	if lastPlayedAt.Valid {
		stats.LastPlayedAt = &lastPlayedAt.String
	}
	return &stats, nil
}

func (s *Store) WalletByUserID(userID string) (*WalletRecord, error) {
	var wallet WalletRecord
	if err := s.db.QueryRow(`select user_id, balance, updated_at from wallets where user_id = ?`, userID).Scan(&wallet.UserID, &wallet.Balance, &wallet.UpdatedAt); err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (s *Store) ListWalletTransactions(userID string, limit int) ([]WalletTransactionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`select id, user_id, type, amount, balance_after, reference_type, reference_id, note, created_at from wallet_transactions where user_id = ? order by created_at desc limit ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WalletTransactionRecord
	for rows.Next() {
		var rec WalletTransactionRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Type, &rec.Amount, &rec.BalanceAfter, &rec.ReferenceType, &rec.ReferenceID, &rec.Note, &rec.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) AddWalletTransaction(userID, typ string, amount int, referenceType, referenceID, note string) (*WalletRecord, *WalletTransactionRecord, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(typ) == "" {
		return nil, nil, errors.New("missing wallet transaction fields")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	var balance int
	if err = tx.QueryRow(`select balance from wallets where user_id = ?`, userID).Scan(&balance); err != nil {
		return nil, nil, err
	}
	balance += amount
	if balance < 0 {
		return nil, nil, errors.New("wallet balance cannot be negative")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.Exec(`update wallets set balance = ?, updated_at = ? where user_id = ?`, balance, now, userID); err != nil {
		return nil, nil, err
	}
	id := fmt.Sprintf("txn_%d", time.Now().UTC().UnixNano())
	if _, err = tx.Exec(`insert into wallet_transactions(id, user_id, type, amount, balance_after, reference_type, reference_id, note, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, userID, typ, amount, balance, referenceType, referenceID, note, now); err != nil {
		return nil, nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}
	return &WalletRecord{UserID: userID, Balance: balance, UpdatedAt: now}, &WalletTransactionRecord{ID: id, UserID: userID, Type: typ, Amount: amount, BalanceAfter: balance, ReferenceType: referenceType, ReferenceID: referenceID, Note: note, CreatedAt: now}, nil
}

func (s *Store) CreateRoom(ownerUserID, ruleSetID string, seatCount, minPlayersToStart int) (*RoomRecord, error) {
	return s.CreateRoomWithOptions(ownerUserID, CreateRoomOptions{
		RuleSetID:         ruleSetID,
		SeatCount:         seatCount,
		MinPlayersToStart: minPlayersToStart,
	})
}

func normalizeCreateRoomOptions(opt CreateRoomOptions) (CreateRoomOptions, error) {
	opt.RuleSetID = strings.TrimSpace(opt.RuleSetID)
	opt.Name = strings.TrimSpace(opt.Name)
	opt.Mode = strings.TrimSpace(opt.Mode)
	opt.Variant = strings.TrimSpace(opt.Variant)
	if opt.RuleSetID == "" {
		opt.RuleSetID = "short-deck"
	}
	if opt.Name == "" {
		opt.Name = "德扑之星"
	}
	if opt.Mode == "" {
		opt.Mode = "training"
	}
	if opt.Variant == "" {
		if opt.RuleSetID == "short-deck" {
			opt.Variant = "short_holdem"
		} else {
			opt.Variant = "holdem"
		}
	}
	if opt.SeatCount == 0 && !opt.HasSeatCount {
		opt.SeatCount = 6
	}
	if opt.MinPlayersToStart == 0 && !opt.HasMinPlayers {
		opt.MinPlayersToStart = 2
	}
	if opt.Ante == 0 && !opt.HasAnte {
		opt.Ante = 20
	}
	if opt.MinBuyIn == 0 && !opt.HasMinBuyIn {
		opt.MinBuyIn = 1000
	}
	if opt.MaxBuyIn == 0 && !opt.HasMaxBuyIn {
		opt.MaxBuyIn = 1000000
	}
	if opt.BuyInCap == 0 && !opt.HasBuyInCap {
		opt.BuyInCap = 1000000
	}
	if opt.DurationMinutes == 0 && !opt.HasDuration {
		opt.DurationMinutes = 120
	}
	if opt.Mode != "training" {
		return opt, fieldError("unsupported_room_mode", "mode")
	}
	if opt.Variant != "short_holdem" && opt.Variant != "holdem" {
		return opt, fieldError("unsupported_variant", "variant")
	}
	if opt.SeatCount < 2 || opt.SeatCount > 9 {
		return opt, fieldError("invalid_room_config", "seatCount")
	}
	if opt.MinPlayersToStart < 2 || opt.MinPlayersToStart > opt.SeatCount {
		return opt, fieldError("invalid_room_config", "minPlayersToStart")
	}
	if opt.Ante <= 0 {
		return opt, fieldError("invalid_room_config", "ante")
	}
	if opt.MinBuyIn <= 0 {
		return opt, fieldError("invalid_room_config", "minBuyIn")
	}
	if opt.MaxBuyIn <= 0 || opt.MaxBuyIn < opt.MinBuyIn {
		return opt, fieldError("invalid_room_config", "maxBuyIn")
	}
	if opt.BuyInCap < opt.MaxBuyIn {
		return opt, fieldError("invalid_room_config", "buyInCap")
	}
	if opt.DurationMinutes <= 0 {
		return opt, fieldError("invalid_room_config", "durationMinutes")
	}
	return opt, nil
}

func (s *Store) CreateRoomWithOptions(ownerUserID string, opt CreateRoomOptions) (*RoomRecord, error) {
	opt, err := normalizeCreateRoomOptions(opt)
	if err != nil {
		return nil, err
	}
	roomID := fmt.Sprintf("room_%d", time.Now().UTC().UnixNano())
	inviteCode := fmt.Sprintf("R%06d", time.Now().UTC().UnixNano()%1000000)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	owner, err := s.FindUserByID(ownerUserID)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`insert into rooms(id, invite_code, owner_user_id, status, rule_set_id, name, mode, variant, ante, min_buy_in, max_buy_in, buy_in_cap, duration_minutes, level, seat_count, min_players_to_start, created_at, updated_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, roomID, inviteCode, ownerUserID, "waiting", opt.RuleSetID, opt.Name, opt.Mode, opt.Variant, opt.Ante, opt.MinBuyIn, opt.MaxBuyIn, opt.BuyInCap, opt.DurationMinutes, 1, opt.SeatCount, opt.MinPlayersToStart, now, now); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`insert into room_members(room_id, user_id, role, joined_at) values(?, ?, ?, ?)`, roomID, ownerUserID, "owner", now); err != nil {
		return nil, err
	}
	for seatNo := 1; seatNo <= opt.SeatCount; seatNo++ {
		if _, err = tx.Exec(`insert into room_seats(room_id, seat_no, seat_status, updated_at) values(?, ?, ?, ?)`, roomID, seatNo, "empty", now); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	room, err := s.RoomByID(roomID)
	if err != nil {
		return nil, err
	}
	if len(room.Members) == 0 {
		room.Members = []RoomMemberRecord{{UserID: ownerUserID, Nickname: owner.Nickname, Role: "owner", JoinedAt: now}}
	}
	return room, nil
}

func (s *Store) JoinRoomByInviteCode(userID, inviteCode string) (*RoomRecord, error) {
	room, err := s.RoomByInviteCode(strings.ToUpper(strings.TrimSpace(inviteCode)))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = s.db.Exec(`insert ignore into room_members(room_id, user_id, role, joined_at) values(?, ?, ?, ?)`, room.ID, userID, "player", now)
	if err != nil {
		return nil, err
	}
	return s.RoomByID(room.ID)
}

func (s *Store) RoomByInviteCode(inviteCode string) (*RoomRecord, error) {
	var roomID string
	if err := s.db.QueryRow(`select id from rooms where invite_code = ?`, strings.ToUpper(strings.TrimSpace(inviteCode))).Scan(&roomID); err != nil {
		return nil, err
	}
	return s.RoomByID(roomID)
}

func (s *Store) RoomByID(roomID string) (*RoomRecord, error) {
	var room RoomRecord
	if err := s.db.QueryRow(`select id, invite_code, owner_user_id, status, rule_set_id, name, mode, variant, ante, min_buy_in, max_buy_in, buy_in_cap, duration_minutes, level, seat_count, min_players_to_start, coalesce(current_game_id, '') from rooms where id = ?`, roomID).Scan(&room.ID, &room.InviteCode, &room.OwnerUserID, &room.Status, &room.RuleSetID, &room.Name, &room.Mode, &room.Variant, &room.Ante, &room.MinBuyIn, &room.MaxBuyIn, &room.BuyInCap, &room.DurationMinutes, &room.Level, &room.SeatCount, &room.MinPlayersToStart, &room.CurrentGameID); err != nil {
		return nil, err
	}
	membersRows, err := s.db.Query(`select m.user_id, p.nickname, m.role, m.joined_at from room_members m join user_profiles p on p.user_id = m.user_id where m.room_id = ? order by m.joined_at asc`, roomID)
	if err != nil {
		return nil, err
	}
	defer membersRows.Close()
	for membersRows.Next() {
		var m RoomMemberRecord
		if err := membersRows.Scan(&m.UserID, &m.Nickname, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		room.Members = append(room.Members, m)
	}
	seatRows, err := s.db.Query(`select s.seat_no, s.seat_status, s.user_id, p.nickname, s.buy_in_chips from room_seats s left join user_profiles p on p.user_id = s.user_id where s.room_id = ? order by s.seat_no asc`, roomID)
	if err != nil {
		return nil, err
	}
	defer seatRows.Close()
	for seatRows.Next() {
		var seat RoomSeatRecord
		if err := seatRows.Scan(&seat.SeatNo, &seat.SeatStatus, &seat.UserID, &seat.Nickname, &seat.BuyInChips); err != nil {
			return nil, err
		}
		room.Seats = append(room.Seats, seat)
	}
	return &room, nil
}

func (s *Store) TakeSeat(roomID, userID string, seatNo, buyInChips int) (*RoomRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var membership string
	if err = tx.QueryRow(`select user_id from room_members where room_id = ? and user_id = ?`, roomID, userID).Scan(&membership); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("room membership required")
		}
		return nil, err
	}
	var balance int
	if err = tx.QueryRow(`select balance from wallets where user_id = ?`, userID).Scan(&balance); err != nil {
		return nil, err
	}
	if buyInChips <= 0 {
		return nil, errors.New("invalid buy-in")
	}
	var minBuyIn, maxBuyIn int
	if err = tx.QueryRow(`select min_buy_in, max_buy_in from rooms where id = ?`, roomID).Scan(&minBuyIn, &maxBuyIn); err != nil {
		return nil, err
	}
	if buyInChips < minBuyIn || buyInChips > maxBuyIn {
		return nil, fieldError("invalid_buy_in", "buyInChips")
	}
	if balance < buyInChips {
		return nil, errors.New("insufficient coins")
	}
	var occupiedSeatNo int
	if err = tx.QueryRow(`select seat_no from room_seats where room_id = ? and user_id = ? limit 1`, roomID, userID).Scan(&occupiedSeatNo); err == nil {
		return nil, errors.New("user already seated")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var existing sql.NullString
	if err = tx.QueryRow(`select user_id from room_seats where room_id = ? and seat_no = ?`, roomID, seatNo).Scan(&existing); err != nil {
		return nil, err
	}
	if existing.Valid {
		return nil, errors.New("seat already taken")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	balance -= buyInChips
	if _, err = tx.Exec(`update wallets set balance = ?, updated_at = ? where user_id = ?`, balance, now, userID); err != nil {
		return nil, err
	}
	txnID := fmt.Sprintf("txn_%d", time.Now().UTC().UnixNano())
	note := fmt.Sprintf("room buy-in %s seat %d", roomID, seatNo)
	if _, err = tx.Exec(`insert into wallet_transactions(id, user_id, type, amount, balance_after, reference_type, reference_id, note, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?)`, txnID, userID, "buy_in", -buyInChips, balance, "room", roomID, note, now); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`update room_seats set user_id = ?, buy_in_chips = ?, seat_status = ?, updated_at = ? where room_id = ? and seat_no = ?`, userID, buyInChips, "occupied", now, roomID, seatNo); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.RoomByID(roomID)
}

func (s *Store) LeaveSeat(roomID, userID string, seatNo int) (*RoomRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var roomStatus string
	if err = tx.QueryRow(`select status from rooms where id = ?`, roomID).Scan(&roomStatus); err != nil {
		return nil, err
	}
	if roomStatus != "waiting" {
		return nil, errors.New("room is not waiting")
	}
	var currentUser sql.NullString
	if err = tx.QueryRow(`select user_id from room_seats where room_id = ? and seat_no = ?`, roomID, seatNo).Scan(&currentUser); err != nil {
		return nil, err
	}
	if !currentUser.Valid || currentUser.String != userID {
		return nil, errors.New("seat not owned by user")
	}
	var buyIn sql.NullInt64
	if err = tx.QueryRow(`select buy_in_chips from room_seats where room_id = ? and seat_no = ?`, roomID, seatNo).Scan(&buyIn); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if buyIn.Valid && buyIn.Int64 > 0 {
		var balance int
		if err = tx.QueryRow(`select balance from wallets where user_id = ?`, userID).Scan(&balance); err != nil {
			return nil, err
		}
		balance += int(buyIn.Int64)
		if _, err = tx.Exec(`update wallets set balance = ?, updated_at = ? where user_id = ?`, balance, now, userID); err != nil {
			return nil, err
		}
		txnID := fmt.Sprintf("txn_%d", time.Now().UTC().UnixNano())
		note := fmt.Sprintf("room leave refund %s seat %d", roomID, seatNo)
		if _, err = tx.Exec(`insert into wallet_transactions(id, user_id, type, amount, balance_after, reference_type, reference_id, note, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?)`, txnID, userID, "leave_refund", int(buyIn.Int64), balance, "room", roomID, note, now); err != nil {
			return nil, err
		}
	}
	if _, err = tx.Exec(`update room_seats set user_id = null, buy_in_chips = null, seat_status = ?, updated_at = ? where room_id = ? and seat_no = ?`, "empty", now, roomID, seatNo); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`update rooms set updated_at = ? where id = ?`, now, roomID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.RoomByID(roomID)
}

func (s *Store) LeaveRoom(roomID, userID string) (*RoomRecord, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var roomStatus, ownerUserID string
	var seatCount int
	if err = tx.QueryRow(`select status, owner_user_id, seat_count from rooms where id = ?`, roomID).Scan(&roomStatus, &ownerUserID, &seatCount); err != nil {
		return nil, err
	}
	if roomStatus == "playing" {
		return nil, errors.New("room is not waiting")
	}

	var currentSeatNo int
	var buyIn sql.NullInt64
	if err = tx.QueryRow(`select seat_no, buy_in_chips from room_seats where room_id = ? and user_id = ? limit 1`, roomID, userID).Scan(&currentSeatNo, &buyIn); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if currentSeatNo > 0 {
		if buyIn.Valid && buyIn.Int64 > 0 {
			var balance int
			if err = tx.QueryRow(`select balance from wallets where user_id = ?`, userID).Scan(&balance); err != nil {
				return nil, err
			}
			balance += int(buyIn.Int64)
			if _, err = tx.Exec(`update wallets set balance = ?, updated_at = ? where user_id = ?`, balance, now, userID); err != nil {
				return nil, err
			}
			txnID := fmt.Sprintf("txn_%d", time.Now().UTC().UnixNano())
			note := fmt.Sprintf("room leave refund %s seat %d", roomID, currentSeatNo)
			if _, err = tx.Exec(`insert into wallet_transactions(id, user_id, type, amount, balance_after, reference_type, reference_id, note, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?)`, txnID, userID, "leave_refund", int(buyIn.Int64), balance, "room", roomID, note, now); err != nil {
				return nil, err
			}
		}
		if _, err = tx.Exec(`update room_seats set user_id = null, buy_in_chips = null, seat_status = ?, updated_at = ? where room_id = ? and seat_no = ?`, "empty", now, roomID, currentSeatNo); err != nil {
			return nil, err
		}
	}

	if _, err = tx.Exec(`delete from room_members where room_id = ? and user_id = ?`, roomID, userID); err != nil {
		return nil, err
	}

	nextOwnerID := ownerUserID
	nextStatus := roomStatus
	if ownerUserID == userID {
		nextOwnerID = ""
		if currentSeatNo == 0 {
			currentSeatNo = 1
		}
		for offset := 1; offset <= seatCount; offset++ {
			candidateSeatNo := ((currentSeatNo - 1 + offset) % seatCount) + 1
			var candidate sql.NullString
			if err = tx.QueryRow(`select user_id from room_seats where room_id = ? and seat_no = ?`, roomID, candidateSeatNo).Scan(&candidate); err != nil {
				return nil, err
			}
			if candidate.Valid && candidate.String != "" {
				nextOwnerID = candidate.String
				break
			}
		}
		if nextOwnerID == "" {
			nextStatus = "closed"
		}
	}

	if _, err = tx.Exec(`update room_members set role = ? where room_id = ?`, "player", roomID); err != nil {
		return nil, err
	}
	if nextOwnerID != "" {
		if _, err = tx.Exec(`update room_members set role = ? where room_id = ? and user_id = ?`, "owner", roomID, nextOwnerID); err != nil {
			return nil, err
		}
	}
	if _, err = tx.Exec(`update rooms set owner_user_id = ?, status = ?, updated_at = ? where id = ?`, nextOwnerID, nextStatus, now, roomID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.RoomByID(roomID)
}

func (s *Store) SetRoomCurrentGame(roomID, gameID string) error {
	_, err := s.db.Exec(`update rooms set current_game_id = ?, status = ?, updated_at = ? where id = ?`, gameID, "playing", time.Now().UTC().Format(time.RFC3339Nano), roomID)
	return err
}

func (s *Store) ArchiveHandResult(roomID string, g *game.Game) error {
	if g == nil || g.Stage != game.StageFinished {
		return errors.New("game is not finished")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var nextHandNo int
	if err = tx.QueryRow(`select coalesce(max(hand_no), 0) + 1 from hand_results where room_id = ?`, roomID).Scan(&nextHandNo); err != nil {
		return err
	}
	handID := fmt.Sprintf("hand_%d", time.Now().UTC().UnixNano())
	completedAt := time.Now().UTC().Format(time.RFC3339Nano)
	boardJSON, err := json.Marshal(g.Board)
	if err != nil {
		return err
	}

	seatUser := map[int]struct{ userID, nickname string }{}
	seatRows, err := tx.Query(`select rs.seat_no, coalesce(rs.user_id, ''), coalesce(up.nickname, '') from room_seats rs left join user_profiles up on up.user_id = rs.user_id where rs.room_id = ?`, roomID)
	if err != nil {
		return err
	}
	for seatRows.Next() {
		var seatNo int
		var userID, nickname string
		if err = seatRows.Scan(&seatNo, &userID, &nickname); err != nil {
			seatRows.Close()
			return err
		}
		seatUser[seatNo] = struct{ userID, nickname string }{userID: userID, nickname: nickname}
	}
	if err = seatRows.Err(); err != nil {
		seatRows.Close()
		return err
	}
	seatRows.Close()

	awardsBySeat := map[int]int{}
	totalPot := 0
	for _, seat := range g.Seats {
		totalPot += seat.HandCommitted
	}
	winningNames := []string{}
	for _, showdown := range g.Showdown {
		award := 0
		for _, amount := range showdown.PotAwards {
			award += amount
		}
		awardsBySeat[showdown.SeatNo] = award
		if award > 0 {
			winningNames = append(winningNames, g.Seat(showdown.SeatNo).Name)
		}
	}
	if len(g.Showdown) == 0 {
		for _, seat := range g.Seats {
			if seat.Status != "folded" {
				awardsBySeat[seat.SeatNo] = totalPot
				winningNames = append(winningNames, seat.Name)
				break
			}
		}
	}
	winnerSummary := strings.Join(winningNames, ", ")
	potSummary := fmt.Sprintf("total=%d", totalPot)
	if _, err = tx.Exec(`insert into hand_results(id, room_id, game_id, hand_no, rule_set_id, completed_at, winner_summary, pot_summary, board_cards_json, total_pot) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, handID, roomID, g.ID, nextHandNo, g.RuleSetID, completedAt, winnerSummary, potSummary, string(boardJSON), totalPot); err != nil {
		return err
	}

	for _, seat := range g.Seats {
		profile := seatUser[seat.SeatNo]
		profit := awardsBySeat[seat.SeatNo] - seat.HandCommitted
		bestCardsJSON := "[]"
		handClass := ""
		for _, showdown := range g.Showdown {
			if showdown.SeatNo == seat.SeatNo {
				bestCardsBytes, marshalErr := json.Marshal(showdown.BestCards)
				if marshalErr != nil {
					return marshalErr
				}
				bestCardsJSON = string(bestCardsBytes)
				handClass = string(showdown.HandClass)
				break
			}
		}
		holeCardsJSON, marshalErr := json.Marshal(seat.HoleCards)
		if marshalErr != nil {
			return marshalErr
		}
		resultType := seat.Status
		if awardsBySeat[seat.SeatNo] > 0 {
			resultType = "won"
		}
		if _, err = tx.Exec(`insert into hand_participants(hand_id, room_id, game_id, user_id, nickname_snapshot, seat_no, profit, result_type, hole_cards_json, best_cards_json, hand_class, hand_committed, award_amount, completed_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, handID, roomID, g.ID, profile.userID, profile.nickname, seat.SeatNo, profit, resultType, string(holeCardsJSON), bestCardsJSON, handClass, seat.HandCommitted, awardsBySeat[seat.SeatNo], completedAt); err != nil {
			return err
		}
		if profile.userID != "" && profit != 0 {
			var balance int
			if err = tx.QueryRow(`select balance from wallets where user_id = ?`, profile.userID).Scan(&balance); err != nil {
				return err
			}
			balance += profit
			if balance < 0 {
				return errors.New("wallet balance cannot be negative")
			}
			if _, err = tx.Exec(`update wallets set balance = ?, updated_at = ? where user_id = ?`, balance, completedAt, profile.userID); err != nil {
				return err
			}
			txnID := fmt.Sprintf("txn_%d", time.Now().UTC().UnixNano()+int64(seat.SeatNo))
			note := fmt.Sprintf("hand result %s", handID)
			if _, err = tx.Exec(`insert into wallet_transactions(id, user_id, type, amount, balance_after, reference_type, reference_id, note, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?)`, txnID, profile.userID, "hand_result", profit, balance, "hand", handID, note, completedAt); err != nil {
				return err
			}
		}
		if profile.userID != "" {
			if _, err = tx.Exec(`update user_profiles set hands_played = hands_played + 1, total_profit = total_profit + ?, last_played_at = ?, updated_at = ? where user_id = ?`, profit, completedAt, completedAt, profile.userID); err != nil {
				return err
			}
		}
	}

	if _, err = tx.Exec(`update rooms set status = ?, current_game_id = ?, updated_at = ? where id = ?`, "waiting", "", completedAt, roomID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) RecentHandResultsByRoom(roomID string, limit int) ([]HandResultRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`select id, room_id, game_id, hand_no, rule_set_id, completed_at, winner_summary, pot_summary, board_cards_json, total_pot from hand_results where room_id = ? order by hand_no desc limit ?`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []HandResultRecord{}
	for rows.Next() {
		var rec HandResultRecord
		var boardJSON string
		if err := rows.Scan(&rec.HandID, &rec.RoomID, &rec.GameID, &rec.HandNo, &rec.RuleSetID, &rec.CompletedAt, &rec.WinnerSummary, &rec.PotSummary, &boardJSON, &rec.TotalPot); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(boardJSON), &rec.BoardCards)
		participants, partErr := s.handParticipants(rec.HandID)
		if partErr != nil {
			return nil, partErr
		}
		rec.Participants = participants
		for _, participant := range participants {
			if participant.Profit > 0 && participant.UserID != "" {
				rec.WinningUserIDs = append(rec.WinningUserIDs, participant.UserID)
			}
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (s *Store) UserHands(userID string, limit int) ([]UserHandRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`select hp.hand_id, hp.room_id, hr.hand_no, hr.completed_at, hp.nickname_snapshot, hp.profit, hr.winner_summary from hand_participants hp join hand_results hr on hr.id = hp.hand_id where hp.user_id = ? order by hr.completed_at desc limit ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []UserHandRecord{}
	for rows.Next() {
		var rec UserHandRecord
		if err := rows.Scan(&rec.HandID, &rec.RoomID, &rec.HandNo, &rec.CompletedAt, &rec.Nickname, &rec.Profit, &rec.WinnerSummary); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (s *Store) RoomLeaderboard(roomID string, limit int) ([]RoomLeaderboardRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		select
			hp.user_id,
			max(hp.nickname_snapshot),
			count(*),
			sum(case when hp.profit > 0 then 1 else 0 end),
			sum(hp.profit),
			max(case when hp.profit > 0 then hr.total_pot else 0 end),
			max(hr.completed_at)
		from hand_participants hp
		join hand_results hr on hr.id = hp.hand_id
		where hp.room_id = ?
		group by hp.user_id
		order by sum(hp.profit) desc, count(*) desc
		limit ?`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RoomLeaderboardRecord{}
	for rows.Next() {
		var rec RoomLeaderboardRecord
		if err := rows.Scan(&rec.UserID, &rec.Nickname, &rec.HandsPlayed, &rec.HandsWon, &rec.NetProfit, &rec.BiggestPotWon, &rec.LastSettledAt); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (s *Store) FormalHandReplay(roomID, handID string) (*HandReplayRecord, error) {
	var gameID string
	if err := s.db.QueryRow(`select game_id from hand_results where room_id = ? and id = ?`, roomID, handID).Scan(&gameID); err != nil {
		return nil, err
	}
	actions, err := s.History(gameID)
	if err != nil {
		return nil, err
	}
	replay := &HandReplayRecord{HandID: handID, RoomID: roomID, GameID: gameID}
	if step, err := s.formalReplayStep(gameID, 0, nil); err == nil {
		replay.Steps = append(replay.Steps, step)
	} else {
		return nil, err
	}
	for _, action := range actions {
		actionCopy := action
		step, err := s.formalReplayStep(gameID, action.Seq, &actionCopy)
		if err != nil {
			return nil, err
		}
		replay.Steps = append(replay.Steps, step)
	}
	return replay, nil
}

func (s *Store) formalReplayStep(gameID string, seq int, action *game.Action) (HandReplayStepRecord, error) {
	g, err := s.SnapshotAt(gameID, seq)
	if err != nil {
		return HandReplayStepRecord{}, err
	}
	step := HandReplayStepRecord{
		Seq:         seq,
		Stage:       string(g.Stage),
		CurrentSeat: g.CurrentSeat,
		BoardCards:  append([]string{}, g.Board...),
		Pot:         replayPotTotal(g),
		Players:     make([]HandReplayPlayerRecord, 0, len(g.Seats)),
	}
	if action != nil {
		step.Action = &HandReplayActionRecord{Type: string(action.Type), SeatNo: action.SeatNo, Amount: action.Amount}
	}
	showHoleCards := g.Stage == game.StageFinished
	for _, seat := range g.Seats {
		player := HandReplayPlayerRecord{
			SeatNo:        seat.SeatNo,
			Nickname:      seat.Name,
			Stack:         seat.Stack,
			Status:        seat.Status,
			HandCommitted: seat.HandCommitted,
		}
		if showHoleCards {
			player.HoleCards = append([]string{}, seat.HoleCards...)
		}
		step.Players = append(step.Players, player)
	}
	return step, nil
}

func replayPotTotal(g *game.Game) int {
	total := 0
	for _, seat := range g.Seats {
		total += seat.HandCommitted
	}
	return total
}

func (s *Store) handParticipants(handID string) ([]HandParticipantRecord, error) {
	rows, err := s.db.Query(`select user_id, nickname_snapshot, seat_no, profit, award_amount, hand_committed, result_type, hole_cards_json, best_cards_json, hand_class from hand_participants where hand_id = ? order by seat_no asc`, handID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []HandParticipantRecord{}
	for rows.Next() {
		var rec HandParticipantRecord
		var holeCardsJSON, bestCardsJSON, handClass string
		if err := rows.Scan(&rec.UserID, &rec.Nickname, &rec.SeatNo, &rec.Profit, &rec.AwardAmount, &rec.HandCommitted, &rec.ResultType, &holeCardsJSON, &bestCardsJSON, &handClass); err != nil {
			return nil, err
		}
		rec.HandClass = handClass
		_ = json.Unmarshal([]byte(holeCardsJSON), &rec.HoleCards)
		_ = json.Unmarshal([]byte(bestCardsJSON), &rec.BestCards)
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (s *Store) Save(g *game.Game) error {
	if g == nil {
		return errors.New("nil game")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	body, err := json.Marshal(g)
	if err != nil {
		return err
	}
	initialSnapshot := g.InitialSnapshotJSON
	if initialSnapshot == "" {
		if err = tx.QueryRow(`select snapshot from snapshots where game_id = ? and seq = 0`, g.ID).Scan(&initialSnapshot); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if initialSnapshot == "" {
		initialSnapshot = string(body)
	}
	_, err = tx.Exec(saveGameUpsertSQL(), g.ID, g.RuleSetID, string(g.Stage), g.Version, string(body), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`delete from actions where game_id = ?`, g.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(`delete from snapshots where game_id = ?`, g.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(`insert into snapshots(game_id, seq, snapshot, created_at) values(?, ?, ?, ?)`, g.ID, 0, initialSnapshot, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	for _, action := range g.Actions {
		payload, marshalErr := json.Marshal(action.Payload)
		if marshalErr != nil {
			return marshalErr
		}
		summary, marshalErr := json.Marshal(action.StateSummary)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = tx.Exec(`insert into actions(game_id, seq, stage, seat_no, type, amount, payload, summary, created_at) values(?, ?, ?, ?, ?, ?, ?, ?, ?)`, g.ID, action.Seq, string(action.Stage), nullableSeat(action.SeatNo), string(action.Type), action.Amount, string(payload), string(summary), action.CreatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	for _, action := range g.Actions {
		snapshotJSON := action.SnapshotJSON
		if snapshotJSON == "" {
			snapshotJSON = string(body)
		}
		_, err = tx.Exec(`insert into snapshots(game_id, seq, snapshot, created_at) values(?, ?, ?, ?)`, g.ID, action.Seq, snapshotJSON, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func saveGameUpsertSQL() string {
	return `insert into games(id, ruleset_id, stage, version, snapshot, updated_at) values(?, ?, ?, ?, ?, ?) on duplicate key update ruleset_id=values(ruleset_id), stage=values(stage), version=values(version), snapshot=values(snapshot), updated_at=values(updated_at)`
}

func (s *Store) Load(id string) (*game.Game, error) {
	var snapshot string
	if err := s.db.QueryRow(`select snapshot from games where id = ?`, id).Scan(&snapshot); err != nil {
		return nil, err
	}
	var g game.Game
	if err := json.Unmarshal([]byte(snapshot), &g); err != nil {
		return nil, err
	}
	_ = s.db.QueryRow(`select snapshot from snapshots where game_id = ? and seq = 0`, id).Scan(&g.InitialSnapshotJSON)
	return &g, nil
}

func (s *Store) History(id string) ([]game.Action, error) {
	rows, err := s.db.Query(`select seq, stage, coalesce(seat_no, 0), type, amount, payload, summary, created_at from actions where game_id = ? order by seq asc`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var actions []game.Action
	for rows.Next() {
		var action game.Action
		var stage, typ, payload, summary, created string
		if err := rows.Scan(&action.Seq, &stage, &action.SeatNo, &typ, &action.Amount, &payload, &summary, &created); err != nil {
			return nil, err
		}
		action.Stage = game.Stage(stage)
		action.Type = game.ActionType(typ)
		_ = json.Unmarshal([]byte(payload), &action.Payload)
		_ = json.Unmarshal([]byte(summary), &action.StateSummary)
		action.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (s *Store) SnapshotAt(id string, seq int) (*game.Game, error) {
	var snapshot string
	if seq < 0 {
		return nil, errors.New("replay sequence out of range")
	}
	if seq == 0 {
		if err := s.db.QueryRow(`select snapshot from snapshots where game_id = ? order by seq asc limit 1`, id).Scan(&snapshot); err != nil {
			return nil, err
		}
	} else {
		var latest int
		if err := s.db.QueryRow(`select coalesce(max(seq), 0) from actions where game_id = ?`, id).Scan(&latest); err != nil {
			return nil, err
		}
		if seq > latest {
			return nil, errors.New("replay sequence out of range")
		}
		if err := s.db.QueryRow(`select snapshot from snapshots where game_id = ? and seq = ?`, id, seq).Scan(&snapshot); err != nil {
			return nil, err
		}
	}
	var g game.Game
	if err := json.Unmarshal([]byte(snapshot), &g); err != nil {
		return nil, err
	}
	_ = s.db.QueryRow(`select snapshot from snapshots where game_id = ? and seq = 0`, id).Scan(&g.InitialSnapshotJSON)
	return &g, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`create table if not exists games (id varchar(64) primary key, ruleset_id varchar(64) not null, stage varchar(32) not null, version integer not null, snapshot longtext not null, updated_at varchar(64) not null)`,
		`create table if not exists actions (game_id varchar(64) not null, seq integer not null, stage varchar(32) not null, seat_no integer null, type varchar(32) not null, amount integer not null, payload longtext not null, summary longtext not null, created_at varchar(64) not null, primary key(game_id, seq))`,
		`create table if not exists snapshots (game_id varchar(64) not null, seq integer not null, snapshot longtext not null, created_at varchar(64) not null, primary key(game_id, seq))`,
		`create table if not exists users (id varchar(64) primary key, username varchar(128) not null unique, password_hash varchar(255) not null, status varchar(32) not null, created_at varchar(64) not null, updated_at varchar(64) not null)`,
		`create table if not exists user_profiles (user_id varchar(64) primary key, nickname varchar(128) not null unique, hands_played integer not null default 0, total_profit integer not null default 0, last_played_at varchar(64) null, updated_at varchar(64) not null)`,
		`create table if not exists wallets (user_id varchar(64) primary key, balance integer not null, updated_at varchar(64) not null)`,
		`create table if not exists wallet_transactions (id varchar(64) primary key, user_id varchar(64) not null, type varchar(64) not null, amount integer not null, balance_after integer not null, reference_type varchar(64), reference_id varchar(64), note varchar(255), created_at varchar(64) not null)`,
		`create table if not exists rooms (id varchar(64) primary key, invite_code varchar(32) not null unique, owner_user_id varchar(64) not null, status varchar(32) not null, rule_set_id varchar(64) not null, name varchar(128) not null default '德扑之星', mode varchar(32) not null default 'training', variant varchar(32) not null default 'short_holdem', ante integer not null default 20, min_buy_in integer not null default 1000, max_buy_in integer not null default 1000000, buy_in_cap integer not null default 1000000, duration_minutes integer not null default 120, level integer not null default 1, seat_count integer not null, min_players_to_start integer not null, current_game_id varchar(64) null, created_at varchar(64) not null, updated_at varchar(64) not null)`,
		`create table if not exists room_members (room_id varchar(64) not null, user_id varchar(64) not null, role varchar(32) not null, joined_at varchar(64) not null, primary key(room_id, user_id))`,
		`create table if not exists room_seats (room_id varchar(64) not null, seat_no integer not null, user_id varchar(64) null, buy_in_chips integer null, seat_status varchar(32) not null, updated_at varchar(64) not null, primary key(room_id, seat_no))`,
		`create table if not exists hand_results (id varchar(64) primary key, room_id varchar(64) not null, game_id varchar(64) not null, hand_no integer not null, rule_set_id varchar(64) not null, completed_at varchar(64) not null, winner_summary varchar(255) not null, pot_summary varchar(255) not null, board_cards_json longtext not null, total_pot integer not null)`,
		`create table if not exists hand_participants (hand_id varchar(64) not null, room_id varchar(64) not null, game_id varchar(64) not null, user_id varchar(64) not null, nickname_snapshot varchar(128) not null, seat_no integer not null, profit integer not null, result_type varchar(32) not null, hole_cards_json longtext not null, best_cards_json longtext not null, hand_class varchar(64) not null, hand_committed integer not null, award_amount integer not null, completed_at varchar(64) not null, primary key(hand_id, seat_no))`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	alterStmts := []string{
		`alter table rooms add column name varchar(128) not null default '德扑之星'`,
		`alter table rooms add column mode varchar(32) not null default 'training'`,
		`alter table rooms add column variant varchar(32) not null default 'short_holdem'`,
		`alter table rooms add column ante integer not null default 20`,
		`alter table rooms add column min_buy_in integer not null default 1000`,
		`alter table rooms add column max_buy_in integer not null default 1000000`,
		`alter table rooms add column buy_in_cap integer not null default 1000000`,
		`alter table rooms add column duration_minutes integer not null default 120`,
		`alter table rooms add column level integer not null default 1`,
	}
	for _, stmt := range alterStmts {
		if _, err := s.db.Exec(stmt); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return err
		}
	}
	return nil
}

func nullableSeat(seatNo int) any {
	if seatNo == 0 {
		return nil
	}
	return seatNo
}
