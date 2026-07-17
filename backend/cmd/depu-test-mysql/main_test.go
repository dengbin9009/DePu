package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/testmysql"
	"github.com/go-sql-driver/mysql"
)

func TestFailedChildCommandStillCleansDatabase(t *testing.T) {
	databaseNameFile := filepath.Join(t.TempDir(), "database-name")
	command := exec.Command(
		"go", "run", ".", "-label", "failed_child", "--",
		"sh", "-c", `printf '%s' "$DEPU_TEST_DATABASE" > "$DEPU_DATABASE_NAME_FILE"; exit 7`,
	)
	command.Env = append(os.Environ(), "DEPU_DATABASE_NAME_FILE="+databaseNameFile)
	if err := command.Run(); err == nil {
		t.Fatal("child command unexpectedly succeeded")
	}

	databaseNameBytes, err := os.ReadFile(databaseNameFile)
	if err != nil {
		t.Fatal(err)
	}
	databaseName := strings.TrimSpace(string(databaseNameBytes))
	if databaseName == "" {
		t.Fatal("child command did not receive DEPU_TEST_DATABASE")
	}

	assertDatabaseRemoved(t, databaseName)
}

func TestInterruptedChildCommandStillCleansDatabase(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "depu-test-mysql")
	buildCommand := exec.Command("go", "build", "-o", binaryPath, ".")
	if output, err := buildCommand.CombinedOutput(); err != nil {
		t.Fatalf("build test mysql command: %v\n%s", err, output)
	}

	databaseNameFile := filepath.Join(t.TempDir(), "database-name")
	childPIDFile := filepath.Join(t.TempDir(), "child-pid")
	command := exec.Command(
		binaryPath, "-label", "interrupted_child", "--",
		"sh", "-c", `printf '%s' "$DEPU_TEST_DATABASE" > "$DEPU_DATABASE_NAME_FILE"; printf '%s' "$$" > "$DEPU_CHILD_PID_FILE"; trap 'exit 130' INT TERM; while :; do sleep 1; done`,
	)
	command.Env = append(os.Environ(),
		"DEPU_DATABASE_NAME_FILE="+databaseNameFile,
		"DEPU_CHILD_PID_FILE="+childPIDFile,
	)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitForFile(databaseNameFile); err != nil {
		_ = command.Process.Kill()
		t.Fatal(err)
	}
	if err := waitForFile(childPIDFile); err != nil {
		_ = command.Process.Kill()
		t.Fatal(err)
	}

	childPIDBytes, err := os.ReadFile(childPIDFile)
	if err != nil {
		t.Fatal(err)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(childPIDBytes)))
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Kill(childPID, syscall.SIGKILL)

	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("interrupted command unexpectedly succeeded")
	}

	databaseNameBytes, err := os.ReadFile(databaseNameFile)
	if err != nil {
		t.Fatal(err)
	}
	assertDatabaseRemoved(t, strings.TrimSpace(string(databaseNameBytes)))
}

func assertDatabaseRemoved(t *testing.T, databaseName string) {
	t.Helper()
	adminConfig, err := mysql.ParseDSN(testmysql.AdminDSN())
	if err != nil {
		t.Fatal(err)
	}
	adminConfig.DBName = ""
	adminDB, err := sql.Open("mysql", adminConfig.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer adminDB.Close()

	var databaseCount int
	if err := adminDB.QueryRow(
		`select count(*) from information_schema.schemata where schema_name = ?`,
		databaseName,
	).Scan(&databaseCount); err != nil {
		t.Fatal(err)
	}
	if databaseCount != 0 {
		_, _ = adminDB.Exec("drop database if exists `" + databaseName + "`")
		t.Fatalf("temporary database %s remained after failed child command", databaseName)
	}
}

func waitForFile(path string) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if contents, err := os.ReadFile(path); err == nil && len(contents) > 0 {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return os.ErrNotExist
}
