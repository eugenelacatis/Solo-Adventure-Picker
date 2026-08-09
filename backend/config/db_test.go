package config

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

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

func TestInitDB_NewDatabaseHasCreatedAtColumns(t *testing.T) {
	db := InitDB(":memory:")
	defer db.Close()

	for _, table := range []string{"visited_adventures", "journal_entries"} {
		var name string
		err := db.QueryRow(`SELECT name FROM pragma_table_info(?) WHERE name = 'created_at'`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing created_at column: %v", table, err)
		}
	}
}

func TestInitDB_NewDatabaseHasVisitedUniqueIndex(t *testing.T) {
	db := InitDB(":memory:")
	defer db.Close()

	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND name = 'idx_visited_user_adventure'`).Scan(&name)
	if err != nil {
		t.Errorf("idx_visited_user_adventure index not found: %v", err)
	}
}

// buildLegacyDB constructs a database matching the pre-migration (v1) schema
// directly, bypassing InitDB, so migrateV2 can be exercised against exactly
// the shape of database it needs to fix.
func buildLegacyDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open legacy db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	legacySchema := `
	CREATE TABLE adventures (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, type TEXT,
		region TEXT NOT NULL, scenery TEXT, effort TEXT, duration TEXT,
		description TEXT, xp_value INTEGER, lat REAL, lng REAL
	);
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL UNIQUE,
		total_xp INTEGER NOT NULL DEFAULT 0, email TEXT UNIQUE, password_hash TEXT
	);
	CREATE TABLE sessions (
		token TEXT PRIMARY KEY, user_id TEXT NOT NULL, expires_at DATETIME NOT NULL
	);
	CREATE TABLE journal_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL,
		adventure_id TEXT NOT NULL, text TEXT NOT NULL
	);
	CREATE TABLE visited_adventures (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL,
		adventure_id TEXT NOT NULL, lat REAL NOT NULL, lng REAL NOT NULL
	);
	`
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}
	return db
}

func TestMigrateV2_DedupesVisitedAdventuresAndAddsUniqueIndex(t *testing.T) {
	db := buildLegacyDB(t)

	// Same user/adventure recorded three times, simulating the pre-dedupe bug.
	for i := 0; i < 3; i++ {
		if _, err := db.Exec(
			`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES (?, ?, ?, ?)`,
			"user-1", "adv-1", 37.9, -122.5,
		); err != nil {
			t.Fatalf("failed to seed duplicate visit: %v", err)
		}
	}

	if err := migrate(db); err != nil {
		t.Fatalf("migrate returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM visited_adventures WHERE user_id = ? AND adventure_id = ?`, "user-1", "adv-1").Scan(&count); err != nil {
		t.Fatalf("failed to count visits: %v", err)
	}
	if count != 1 {
		t.Errorf("visited_adventures rows for user-1/adv-1 = %d, want 1 after dedupe", count)
	}

	// The unique index must now reject a fresh duplicate insert.
	_, err := db.Exec(`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES (?, ?, ?, ?)`,
		"user-1", "adv-1", 37.9, -122.5)
	if err == nil {
		t.Error("expected duplicate insert to be rejected by unique index after migration, got no error")
	}
}

func TestMigrateV2_BackfillsXpValueFromType(t *testing.T) {
	db := buildLegacyDB(t)

	if _, err := db.Exec(`INSERT INTO adventures (name, type, region, lat, lng) VALUES (?, ?, ?, ?, ?)`,
		"Mount Tam", "hike", "bay-area", 37.9, -122.5); err != nil {
		t.Fatalf("failed to seed adventure: %v", err)
	}

	if err := migrate(db); err != nil {
		t.Fatalf("migrate returned error: %v", err)
	}

	var xpValue int
	if err := db.QueryRow(`SELECT xp_value FROM adventures WHERE name = ?`, "Mount Tam").Scan(&xpValue); err != nil {
		t.Fatalf("failed to read xp_value: %v", err)
	}
	if xpValue != 150 {
		t.Errorf("xp_value = %d, want 150 for type=hike", xpValue)
	}
}

func TestMigrateV2_AddsCreatedAtColumns(t *testing.T) {
	db := buildLegacyDB(t)

	if err := migrate(db); err != nil {
		t.Fatalf("migrate returned error: %v", err)
	}

	for _, table := range []string{"visited_adventures", "journal_entries"} {
		var name string
		err := db.QueryRow(`SELECT name FROM pragma_table_info(?) WHERE name = 'created_at'`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing created_at column after migration: %v", table, err)
		}
	}
}

// TestInitDB_MigratesLegacyFileDatabaseWithDuplicates exercises the full
// InitDB path (createSchema + migrate together) against a legacy on-disk
// database with pre-existing duplicate visits, not just migrate() in
// isolation. createSchema used to unconditionally create the unique index
// on visited_adventures, which fails outright against duplicate data —
// that only surfaces when createSchema and migrate run in InitDB's actual
// order against a real file, which calling migrate() directly in the other
// tests here never exercised.
func TestInitDB_MigratesLegacyFileDatabaseWithDuplicates(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open legacy db file: %v", err)
	}
	legacySchema := `
	CREATE TABLE adventures (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, type TEXT,
		region TEXT NOT NULL, scenery TEXT, effort TEXT, duration TEXT,
		description TEXT, xp_value INTEGER, lat REAL, lng REAL
	);
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL UNIQUE,
		total_xp INTEGER NOT NULL DEFAULT 0, email TEXT UNIQUE, password_hash TEXT
	);
	CREATE TABLE sessions (
		token TEXT PRIMARY KEY, user_id TEXT NOT NULL, expires_at DATETIME NOT NULL
	);
	CREATE TABLE journal_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL,
		adventure_id TEXT NOT NULL, text TEXT NOT NULL
	);
	CREATE TABLE visited_adventures (
		id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL,
		adventure_id TEXT NOT NULL, lat REAL NOT NULL, lng REAL NOT NULL
	);
	`
	if _, err := legacyDB.Exec(legacySchema); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := legacyDB.Exec(
			`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES (?, ?, ?, ?)`,
			"legacy-user", "adv-1", 37.9, -122.5,
		); err != nil {
			t.Fatalf("failed to seed duplicate visit: %v", err)
		}
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("failed to close legacy db: %v", err)
	}

	db := InitDB(dbPath)
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM visited_adventures WHERE user_id = ? AND adventure_id = ?`, "legacy-user", "adv-1").Scan(&count); err != nil {
		t.Fatalf("failed to count visits: %v", err)
	}
	if count != 1 {
		t.Errorf("visited_adventures rows = %d, want 1 after InitDB migrates a legacy file database", count)
	}
}

func TestMigrateV2_IsIdempotent(t *testing.T) {
	db := buildLegacyDB(t)

	if err := migrate(db); err != nil {
		t.Fatalf("first migrate returned error: %v", err)
	}
	if err := migrate(db); err != nil {
		t.Errorf("second migrate returned error: %v", err)
	}

	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("failed to read user_version: %v", err)
	}
	if version != schemaVersion {
		t.Errorf("user_version = %d, want %d", version, schemaVersion)
	}
}
