package testmysql

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/go-sql-driver/mysql"
)

const defaultAdminDSN = "root@tcp(127.0.0.1:3306)/?parseTime=true&multiStatements=true"

var (
	databaseSequence  atomic.Uint64
	fallbackRunID     string
	fallbackRunIDOnce sync.Once
)

type Database struct {
	Name string
	DSN  string

	adminDB    *sql.DB
	cleanupErr error
	cleanup    sync.Once
}

func AdminDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("DEPU_TEST_MYSQL_ADMIN_DSN")); dsn != "" {
		return dsn
	}
	if dsn := strings.TrimSpace(os.Getenv("DEPU_TEST_MYSQL_DSN")); dsn != "" {
		return dsn
	}
	return defaultAdminDSN
}

func RunID() string {
	if runID := sanitizeComponent(os.Getenv("DEPU_TEST_RUN_ID"), 20); runID != "" {
		return runID
	}
	fallbackRunIDOnce.Do(func() {
		fallbackRunID = sanitizeComponent(
			fmt.Sprintf("%s_%d", time.Now().UTC().Format("20060102t150405"), os.Getpid()),
			20,
		)
	})
	return fallbackRunID
}

func CreateDatabase(adminDSN, label string) (*Database, error) {
	adminConfig, err := mysql.ParseDSN(strings.TrimSpace(adminDSN))
	if err != nil {
		return nil, fmt.Errorf("parse mysql admin dsn: %w", err)
	}
	adminConfig.DBName = ""
	adminDB, err := sql.Open("mysql", adminConfig.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql admin connection: %w", err)
	}
	if err := adminDB.Ping(); err != nil {
		_ = adminDB.Close()
		return nil, fmt.Errorf("ping mysql admin connection: %w", err)
	}

	databaseName := newDatabaseName(label)
	if _, err := adminDB.Exec("create database `" + databaseName + "` character set utf8mb4 collate utf8mb4_unicode_ci"); err != nil {
		_ = adminDB.Close()
		return nil, fmt.Errorf("create mysql database %s: %w", databaseName, err)
	}

	targetConfig := adminConfig.Clone()
	targetConfig.DBName = databaseName
	database := &Database{Name: databaseName, DSN: targetConfig.FormatDSN(), adminDB: adminDB}
	if err := database.Ping(); err != nil {
		_ = database.Cleanup()
		return nil, fmt.Errorf("ping mysql database %s: %w", databaseName, err)
	}
	return database, nil
}

func (database *Database) Ping() error {
	db, err := sql.Open("mysql", database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

func (database *Database) Cleanup() error {
	database.cleanup.Do(func() {
		if _, err := database.adminDB.Exec("drop database if exists `" + database.Name + "`"); err != nil {
			database.cleanupErr = err
		}
		if err := database.adminDB.Close(); database.cleanupErr == nil && err != nil {
			database.cleanupErr = err
		}
	})
	return database.cleanupErr
}

func newDatabaseName(label string) string {
	sequence := databaseSequence.Add(1)
	return fmt.Sprintf(
		"depu_%s_%s_%s_%s",
		RunID(),
		componentOrDefault(label, "test", 16),
		strconv.FormatInt(time.Now().UTC().UnixNano(), 36),
		strconv.FormatUint(sequence, 36),
	)
}

func componentOrDefault(value, fallback string, maxLength int) string {
	if component := sanitizeComponent(value, maxLength); component != "" {
		return component
	}
	return fallback
}

func sanitizeComponent(value string, maxLength int) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if (unicode.IsLetter(char) || unicode.IsDigit(char)) && char <= unicode.MaxASCII {
			builder.WriteRune(char)
			lastUnderscore = false
		} else if builder.Len() > 0 && !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
		if builder.Len() >= maxLength {
			break
		}
	}
	return strings.Trim(builder.String(), "_")
}
