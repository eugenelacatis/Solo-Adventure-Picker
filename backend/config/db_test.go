package config

import (
	"database/sql"
	"os"
	"testing"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/solo_adventure_picker?sslmode=disable"
	}
	db := InitDB(dsn)
	t.Cleanup(func() {
		db.Exec(`TRUNCATE TABLE sessions, journal_entries, visited_adventures, users, adventures RESTART IDENTITY CASCADE`)
		db.Close()
	})
	return db
}

func TestInitDB_CreatesExpectedTables(t *testing.T) {
	db := testDB(t)

	tables := []string{"adventures", "users", "journal_entries", "visited_adventures", "sessions", "schema_migrations"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found after InitDB: %v", table, err)
		}
	}
}

func TestInitDB_UsersTableHasAuthColumns(t *testing.T) {
	db := testDB(t)

	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'users'`)
	if err != nil {
		t.Fatalf("failed to inspect users table: %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
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
	db := testDB(t)

	// Re-running schema creation against the same connection must not error.
	if err := createSchema(db); err != nil {
		t.Errorf("createSchema returned error on second call: %v", err)
	}
}

func TestInitDB_NewDatabaseHasCreatedAtColumns(t *testing.T) {
	db := testDB(t)

	for _, table := range []string{"visited_adventures", "journal_entries"} {
		var name string
		err := db.QueryRow(`SELECT column_name FROM information_schema.columns WHERE table_name = $1 AND column_name = 'created_at'`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing created_at column: %v", table, err)
		}
	}
}

func TestInitDB_NewDatabaseHasVisitedUniqueIndex(t *testing.T) {
	db := testDB(t)

	var name string
	err := db.QueryRow(`SELECT indexname FROM pg_indexes WHERE indexname = 'idx_visited_user_adventure'`).Scan(&name)
	if err != nil {
		t.Errorf("idx_visited_user_adventure index not found: %v", err)
	}
}
