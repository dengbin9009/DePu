package testmysql

import (
	"os"
	"strings"
	"testing"
)

func TestRunIDIsStableAndSafe(t *testing.T) {
	t.Setenv("DEPU_TEST_RUN_ID", "Loop Round 1 / 多账号")

	first := RunID()
	second := RunID()
	if first != second {
		t.Fatalf("RunID() changed within one process: %q != %q", first, second)
	}
	if first != "loop_round_1" {
		t.Fatalf("RunID() = %q, want loop_round_1", first)
	}
}

func TestCreateDatabaseProvidesIsolatedDSNAndCleanup(t *testing.T) {
	adminDSN := strings.TrimSpace(os.Getenv("DEPU_TEST_MYSQL_ADMIN_DSN"))
	if adminDSN == "" {
		adminDSN = "root@tcp(127.0.0.1:3306)/?parseTime=true&multiStatements=true"
	}

	first, err := CreateDatabase(adminDSN, "multi_account")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Cleanup()

	second, err := CreateDatabase(adminDSN, "multi_account")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Cleanup()

	if first.Name == second.Name {
		t.Fatalf("database names must be unique, both were %q", first.Name)
	}
	if !strings.Contains(first.Name, RunID()) || !strings.Contains(second.Name, RunID()) {
		t.Fatalf("database names must include run id %q: %q %q", RunID(), first.Name, second.Name)
	}
	if first.DSN == adminDSN || second.DSN == adminDSN {
		t.Fatal("isolated database DSN must not reuse the admin DSN")
	}

	if err := first.Ping(); err != nil {
		t.Fatalf("first database unavailable before cleanup: %v", err)
	}
	if err := second.Ping(); err != nil {
		t.Fatalf("second database unavailable before cleanup: %v", err)
	}
	if err := first.Cleanup(); err != nil {
		t.Fatalf("cleanup first database: %v", err)
	}
	if err := first.Ping(); err == nil {
		t.Fatal("first database still accepts connections after cleanup")
	}
	if err := second.Ping(); err != nil {
		t.Fatalf("cleaning first database affected second database: %v", err)
	}
}
