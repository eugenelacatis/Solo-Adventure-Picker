package config

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS adventures (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT NOT NULL,
	type        TEXT,
	region      TEXT NOT NULL,
	scenery     TEXT,
	effort      TEXT,
	duration    TEXT,
	description TEXT,
	xp_value    INTEGER,
	lat         REAL,
	lng         REAL
);

CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id       TEXT NOT NULL UNIQUE,
	total_xp      INTEGER NOT NULL DEFAULT 0,
	email         TEXT UNIQUE,
	password_hash TEXT
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	expires_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS journal_entries (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id      TEXT NOT NULL,
	adventure_id TEXT NOT NULL,
	text         TEXT NOT NULL,
	created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS visited_adventures (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id      TEXT NOT NULL,
	adventure_id TEXT NOT NULL,
	lat          REAL NOT NULL,
	lng          REAL NOT NULL,
	created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// The unique index on visited_adventures is created by migrateV2, not here.
// A legacy database may already contain duplicate (user_id, adventure_id)
// rows; creating the index in createSchema would run before migrateV2 gets
// a chance to dedupe them, and CREATE UNIQUE INDEX fails outright against
// duplicate data. migrateV2 dedupes first and then creates the index, which
// is a no-op-safe order for both legacy and brand-new databases (a fresh
// database has zero duplicates, so the dedupe step there does nothing).

// schemaVersion is the current PRAGMA user_version. Bump it and add a case
// to migrate whenever the schema changes so existing databases (which
// CREATE TABLE IF NOT EXISTS alone cannot alter) get brought up to date.
const schemaVersion = 2

// InitDB opens the SQLite database at the given path (or ":memory:" for an
// in-memory database), ensures the base schema exists, and runs any
// outstanding migrations.
func InitDB(path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	if err := createSchema(db); err != nil {
		log.Fatal(err)
	}
	if err := migrate(db); err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to SQLite at", path)
	return db
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

// migrate brings a database from whatever PRAGMA user_version it's at up to
// schemaVersion, running each numbered step in order. A brand-new database
// created by createSchema above already has every column and index the
// migrations would add, so each step must be safe to run against both an
// old database and (harmlessly) a fresh one.
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}

	if version < 2 {
		if err := migrateV2(db); err != nil {
			return err
		}
		version = 2
	}

	_, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version))
	return err
}

// migrateV2 adds created_at timestamps, removes duplicate visited-adventure
// rows recorded before dedupe was enforced, adds the unique index that
// prevents future duplicates, and backfills xp_value for adventures seeded
// before XP values were stored on the row.
func migrateV2(db *sql.DB) error {
	if err := addColumnIfMissing(db, "visited_adventures", "created_at", "DATETIME"); err != nil {
		return err
	}
	if err := addColumnIfMissing(db, "journal_entries", "created_at", "DATETIME"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		DELETE FROM visited_adventures
		WHERE id NOT IN (
			SELECT MIN(id) FROM visited_adventures GROUP BY user_id, adventure_id
		)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_visited_user_adventure
		ON visited_adventures(user_id, adventure_id)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE adventures SET xp_value = CASE type
			WHEN 'hike' THEN 150
			WHEN 'outdoor' THEN 150
			WHEN 'food' THEN 75
			WHEN 'cafe' THEN 75
			WHEN 'culture' THEN 125
			WHEN 'museum' THEN 125
			ELSE 100
		END
		WHERE xp_value IS NULL
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE adventures SET description = 'Embark on this adventure!'
		WHERE description IS NULL OR description = ''
	`); err != nil {
		return err
	}

	return nil
}

// addColumnIfMissing adds column to table if it isn't already present.
// SQLite has no "ADD COLUMN IF NOT EXISTS", so PRAGMA table_info is checked
// first; this also makes the migration safe to run against a fresh
// database where createSchema already created the column.
func addColumnIfMissing(db *sql.DB, table, column, colType string) error {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, colType))
	return err
}
