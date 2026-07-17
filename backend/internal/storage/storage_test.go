package storage

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/game"
	"github.com/dengbin9009/DePu/backend/internal/testmysql"
)

func openStorageTestStore(t *testing.T) (*Store, error) {
	t.Helper()
	database, err := testmysql.CreateDatabase(testmysql.AdminDSN(), "storage")
	if err != nil {
		t.Skipf("mysql storage test database unavailable: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Cleanup(); err != nil {
			t.Errorf("cleanup mysql storage test database %s: %v", database.Name, err)
		}
	})
	store, err := OpenWithConfig(Config{Driver: DriverMySQL, DSN: database.DSN})
	if err != nil {
		t.Skipf("mysql storage test store unavailable: %v", err)
	}
	return store, nil
}

func TestSaveLoadGameAndHistory(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	g, err := game.New(game.Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		SmallBlind: 50,
		BigBlind:   100,
		Seats: []game.SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
		DealMode: game.DealRandom,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(g); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != g.ID || loaded.Version != g.Version {
		t.Fatalf("loaded game mismatch: %#v vs %#v", loaded, g)
	}
	history, err := store.History(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != len(g.Actions) {
		t.Fatalf("history length = %d, want %d", len(history), len(g.Actions))
	}

	replayed, err := store.SnapshotAt(g.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != g.ID {
		t.Fatalf("replayed game id = %s, want %s", replayed.ID, g.ID)
	}
	if replayed.Version == 0 {
		t.Fatal("replayed snapshot should preserve a version")
	}
}

func TestSaveUsesMySQLUpsertSyntax(t *testing.T) {
	if got := saveGameUpsertSQL(); !strings.Contains(got, "on duplicate key update") {
		t.Fatalf("expected mysql upsert syntax, got %s", got)
	}
}

func TestMigrateAddsVersionToExistingRoomsTable(t *testing.T) {
	database, err := testmysql.CreateDatabase(testmysql.AdminDSN(), "storage_room_version_migration")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.Cleanup(); err != nil {
			t.Errorf("cleanup mysql storage test database %s: %v", database.Name, err)
		}
	})

	db, err := sql.Open("mysql", database.DSN)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`create table rooms (
		id varchar(64) primary key,
		invite_code varchar(32) not null unique,
		owner_user_id varchar(64) not null,
		status varchar(32) not null,
		rule_set_id varchar(64) not null,
		name varchar(128) not null default '德扑之星',
		mode varchar(32) not null default 'training',
		variant varchar(32) not null default 'short_holdem',
		ante integer not null default 20,
		min_buy_in integer not null default 1000,
		max_buy_in integer not null default 1000000,
		buy_in_cap integer not null default 1000000,
		duration_minutes integer not null default 120,
		level integer not null default 1,
		seat_count integer not null,
		min_players_to_start integer not null,
		current_game_id varchar(64) null,
		created_at varchar(64) not null,
		updated_at varchar(64) not null
	)`)
	if closeErr := db.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}

	store, err := OpenWithConfig(Config{Driver: DriverMySQL, DSN: database.DSN})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	var columns int
	if err := store.db.QueryRow(`select count(*) from information_schema.columns where table_schema = database() and table_name = 'rooms' and column_name = 'version'`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 1 {
		t.Fatalf("rooms.version columns=%d, want 1", columns)
	}
}

func TestTakeSeatConflictReleasesTransaction(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	store.db.SetMaxOpenConns(1)

	owner, _, err := store.CreateUser("txn_owner", "hash", "事务房主", 3000)
	if err != nil {
		t.Fatal(err)
	}
	player, _, err := store.CreateUser("txn_player", "hash", "事务玩家", 3000)
	if err != nil {
		t.Fatal(err)
	}
	room, err := store.CreateRoom(owner.ID, "long-holdem", 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.JoinRoomByInviteCode(player.ID, room.InviteCode); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeSeat(room.ID, owner.ID, 1, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeSeat(room.ID, player.ID, 1, 1000); err == nil || !strings.Contains(err.Error(), "seat already taken") {
		t.Fatalf("take occupied seat error = %v, want seat already taken", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := store.TakeSeat(room.ID, player.ID, 2, 1000)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("take open seat after conflict: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("take seat blocked after conflict; transaction was not released")
	}
}

func TestTakeSeatConcurrentSameSeatChargesOnlyWinner(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	owner, _, err := store.CreateUser("seat_race_owner", "hash", "抢座房主", 3000)
	if err != nil {
		t.Fatal(err)
	}
	first, _, err := store.CreateUser("seat_race_first", "hash", "抢座甲", 3000)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := store.CreateUser("seat_race_second", "hash", "抢座乙", 3000)
	if err != nil {
		t.Fatal(err)
	}
	room, err := store.CreateRoom(owner.ID, "long-holdem", 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []*UserRecord{first, second} {
		if _, err := store.JoinRoomByInviteCode(user.ID, room.InviteCode); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.db.Exec(`create trigger delay_concurrent_take before update on room_seats for each row set @depu_delay_take = sleep(if(new.user_id is not null, 0.2, 0))`); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, userID := range []string{first.ID, second.ID} {
		go func(takingUserID string) {
			ready.Done()
			<-start
			_, takeErr := store.TakeSeat(room.ID, takingUserID, 2, 1000)
			results <- takeErr
		}(userID)
	}
	ready.Wait()
	close(start)

	successes := 0
	conflicts := 0
	for requestIndex := 0; requestIndex < 2; requestIndex++ {
		takeErr := <-results
		switch {
		case takeErr == nil:
			successes++
		case strings.Contains(takeErr.Error(), "seat already taken"):
			conflicts++
		default:
			t.Fatalf("concurrent take seat error=%v", takeErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent take seat successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}

	updated, err := store.RoomByID(room.ID)
	if err != nil {
		t.Fatal(err)
	}
	winnerID := ""
	for _, seat := range updated.Seats {
		if seat.SeatNo == 2 && seat.UserID != nil {
			winnerID = *seat.UserID
		}
	}
	if winnerID == "" {
		t.Fatal("concurrent take seat left target seat empty")
	}
	for _, user := range []*UserRecord{first, second} {
		wallet, err := store.WalletByUserID(user.ID)
		if err != nil {
			t.Fatal(err)
		}
		want := 3000
		if user.ID == winnerID {
			want = 2000
		}
		if wallet.Balance != want {
			t.Fatalf("wallet %s balance=%d, want %d", user.ID, wallet.Balance, want)
		}
	}
}

func TestLeaveSeatConcurrentRequestsRefundOnlyOnce(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	owner, _, err := store.CreateUser("stand_race_owner", "hash", "离座房主", 3000)
	if err != nil {
		t.Fatal(err)
	}
	room, err := store.CreateRoom(owner.ID, "long-holdem", 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeSeat(room.ID, owner.ID, 1, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`create trigger delay_concurrent_stand before update on wallets for each row set @depu_delay_stand = sleep(if(new.balance > old.balance, 0.2, 0))`); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for requestIndex := 0; requestIndex < 2; requestIndex++ {
		go func() {
			ready.Done()
			<-start
			_, leaveErr := store.LeaveSeat(room.ID, owner.ID, 1)
			results <- leaveErr
		}()
	}
	ready.Wait()
	close(start)

	successes := 0
	conflicts := 0
	for requestIndex := 0; requestIndex < 2; requestIndex++ {
		leaveErr := <-results
		switch {
		case leaveErr == nil:
			successes++
		case strings.Contains(leaveErr.Error(), "seat not owned"):
			conflicts++
		default:
			t.Fatalf("concurrent leave seat error=%v", leaveErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent leave seat successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
	wallet, err := store.WalletByUserID(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wallet.Balance != 3000 {
		t.Fatalf("wallet balance=%d, want one refund to 3000", wallet.Balance)
	}
	transactions, err := store.ListWalletTransactions(owner.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	refunds := 0
	for _, transaction := range transactions {
		if transaction.Type == "leave_refund" && transaction.ReferenceID == room.ID {
			refunds++
		}
	}
	if refunds != 1 {
		t.Fatalf("leave seat refunds=%d, want 1", refunds)
	}
}

func TestTakeSeatRejectsPlayingRoom(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	owner, _, err := store.CreateUser("playing_seat_owner", "hash", "进行中房主", 3000)
	if err != nil {
		t.Fatal(err)
	}
	player, _, err := store.CreateUser("playing_seat_player", "hash", "进行中玩家", 3000)
	if err != nil {
		t.Fatal(err)
	}
	room, err := store.CreateRoom(owner.ID, "long-holdem", 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.JoinRoomByInviteCode(player.ID, room.InviteCode); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`update rooms set status = 'playing' where id = ?`, room.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeSeat(room.ID, player.ID, 2, 1000); err == nil || !strings.Contains(err.Error(), "room is not waiting") {
		t.Fatalf("take seat playing room error=%v, want room is not waiting", err)
	}
}

func TestLeaveRoomConcurrentRequestsRefundOnlyOnce(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	owner, _, err := store.CreateUser("leave_owner", "hash", "退出房主", 3000)
	if err != nil {
		t.Fatal(err)
	}
	player, _, err := store.CreateUser("leave_player", "hash", "退出玩家", 3000)
	if err != nil {
		t.Fatal(err)
	}
	room, err := store.CreateRoom(owner.ID, "long-holdem", 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.JoinRoomByInviteCode(player.ID, room.InviteCode); err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeSeat(room.ID, player.ID, 2, 1000); err != nil {
		t.Fatal(err)
	}

	if _, err := store.db.Exec(`create trigger delay_leave_refund before update on wallets for each row set @depu_delay_leave_refund = sleep(if(new.balance > old.balance, 0.2, 0))`); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errorsByRequest := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for requestIndex := 0; requestIndex < 2; requestIndex++ {
		go func() {
			ready.Done()
			<-start
			_, leaveErr := store.LeaveRoom(room.ID, player.ID)
			errorsByRequest <- leaveErr
		}()
	}
	ready.Wait()
	close(start)

	for requestIndex := 0; requestIndex < 2; requestIndex++ {
		if err := <-errorsByRequest; err != nil {
			t.Fatalf("concurrent leave room: %v", err)
		}
	}

	wallet, err := store.WalletByUserID(player.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wallet.Balance != 3000 {
		t.Fatalf("wallet balance=%d, want one 1000 chip refund", wallet.Balance)
	}
	transactions, err := store.ListWalletTransactions(player.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	refundCount := 0
	for _, transaction := range transactions {
		if transaction.Type == "leave_refund" && transaction.ReferenceID == room.ID {
			refundCount++
		}
	}
	if refundCount != 1 {
		t.Fatalf("leave refund transactions=%d, want 1", refundCount)
	}

	updated, err := store.RoomByID(room.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range updated.Members {
		if member.UserID == player.ID {
			t.Fatalf("player membership remains after leave: %#v", member)
		}
	}
	for _, seat := range updated.Seats {
		if seat.UserID != nil && *seat.UserID == player.ID || seat.SeatNo == 2 && seat.SeatStatus != "empty" {
			t.Fatalf("player seat remains after leave: %#v", seat)
		}
	}
}

func TestLeaveRoomClosingRoomClearsCurrentGameReference(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	owner, _, err := store.CreateUser("close_owner", "hash", "关闭房间房主", 3000)
	if err != nil {
		t.Fatal(err)
	}
	room, err := store.CreateRoom(owner.ID, "long-holdem", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.TakeSeat(room.ID, owner.ID, 1, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`update rooms set current_game_id = ? where id = ?`, "stale_game", room.ID); err != nil {
		t.Fatal(err)
	}

	closed, err := store.LeaveRoom(room.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != "closed" || closed.OwnerUserID != "" || closed.CurrentGameID != "" {
		t.Fatalf("closed room=%#v, want no owner or current game", closed)
	}
}

func TestSaveReturnsErrorWhenStorageUnavailable(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	g, err := game.New(game.Config{
		RuleSetID:  "long-holdem",
		ButtonSeat: 1,
		BettingStructure: game.BettingStructure{
			Type:       game.BettingBlinds,
			SmallBlind: 50,
			BigBlind:   100,
		},
		Seats: []game.SeatConfig{
			{SeatNo: 1, Name: "BTN", Stack: 1000},
			{SeatNo: 2, Name: "SB", Stack: 1000},
			{SeatNo: 3, Name: "BB", Stack: 1000},
		},
		DealMode: game.DealDebug,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.SetDebugCards(map[int][]string{1: []string{"As", "Ah"}}, []string{"Ks", "Kh", "Qh"}); err != nil {
		t.Fatal(err)
	}
	g.Stage = game.StageRiver
	for i := range g.Seats {
		g.Seats[i].HasActed = true
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(g); err == nil {
		t.Fatal("expected save to fail when storage is unavailable")
	}
}
