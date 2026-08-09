package config

import "testing"

func TestInitDB_CreatesExpectedTables(t *testing.T) {
	db := InitDB(":memory:")
	defer db.Close()

	tables := []string{"adventures", "users", "journal_entries", "visited_adventures", "sessions"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after InitDB: %v", table, err)
		}
	}
}

func TestInitDB_UsersTableHasAuthColumns(t *testing.T) {
	db := InitDB(":memory:")
	defer db.Close()

	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		t.Fatalf("failed to inspect users table: %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("failed to scan column info: %v", err)
		}
		columns[name] = true
	}

	for _, want := range []string{"email", "password_hash"} {
		if !columns[want] {
			t.Errorf("users table missing column %q, got columns %v", want, columns)
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
