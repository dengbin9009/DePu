package storage

import "testing"

func TestOpenMySQLConfig(t *testing.T) {
	store, err := openStorageTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.Driver() != DriverMySQL {
		t.Fatalf("driver = %s, want %s", store.Driver(), DriverMySQL)
	}
}
