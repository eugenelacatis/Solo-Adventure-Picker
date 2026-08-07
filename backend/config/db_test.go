package config

import "testing"

func TestInitDB_CreatesExpectedTables(t *testing.T) {
	db := InitDB(":memory:")
	defer db.Close()

	tables := []string{"adventures", "users", "journal_entries", "visited_adventures"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after InitDB: %v", table, err)
		}
	}
}

func TestInitDB_IsIdempotent(t *testing.T) {
	db := InitDB(":memory:")
	defer db.Close()

	// Re-running schema creation against the same connection must not error.
	if err := createSchema(db); err != nil {
		t.Errorf("createSchema returned error on second call: %v", err)
	}
}
