package storage

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/game"
)

func openStorageTestStore(t *testing.T) (*Store, error) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("DEPU_TEST_MYSQL_DSN"))
	if dsn == "" {
		dsn = createStorageMySQLTestDatabase(t)
	}
	store, err := OpenWithConfig(Config{Driver: DriverMySQL, DSN: dsn})
	if err != nil {
		t.Skipf("mysql storage test store unavailable: %v", err)
	}
	return store, nil
}

func createStorageMySQLTestDatabase(t *testing.T) string {
	t.Helper()
	adminDSN := strings.TrimSpace(os.Getenv("DEPU_TEST_MYSQL_ADMIN_DSN"))
	if adminDSN == "" {
		adminDSN = "root@tcp(127.0.0.1:3306)/?parseTime=true&multiStatements=true"
	}
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Skipf("mysql admin connection unavailable: %v", err)
		return ""
	}
	if err := adminDB.Ping(); err != nil {
		_ = adminDB.Close()
		t.Skipf("mysql admin ping failed: %v", err)
		return ""
	}
	dbName := fmt.Sprintf("depu_storage_test_%d", time.Now().UTC().UnixNano())
	if _, err := adminDB.Exec("create database `" + dbName + "` character set utf8mb4 collate utf8mb4_unicode_ci"); err != nil {
		_ = adminDB.Close()
		t.Skipf("create mysql test database failed: %v", err)
		return ""
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec("drop database if exists `" + dbName + "`")
		_ = adminDB.Close()
	})
	return "root@tcp(127.0.0.1:3306)/" + dbName + "?parseTime=true&multiStatements=true"
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
