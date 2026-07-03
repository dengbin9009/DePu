package storage

import "testing"

func TestOpenMySQLConfig(t *testing.T) {
	store, err := OpenWithConfig(Config{Driver: DriverMySQL, DSN: "root@tcp(127.0.0.1:3306)/depu_multiplayer?parseTime=true&multiStatements=true"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.Driver() != DriverMySQL {
		t.Fatalf("driver = %s, want %s", store.Driver(), DriverMySQL)
	}
}
