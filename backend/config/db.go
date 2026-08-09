package config

import (
	"database/sql"
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
	text         TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS visited_adventures (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id      TEXT NOT NULL,
	adventure_id TEXT NOT NULL,
	lat          REAL NOT NULL,
	lng          REAL NOT NULL
);
`

// InitDB opens the SQLite database at the given path (or ":memory:" for an
// in-memory database) and ensures the schema exists.
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
	log.Println("Connected to SQLite at", path)
	return db
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
