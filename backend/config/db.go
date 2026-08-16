package config

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const schema = `
CREATE TABLE IF NOT EXISTS adventures (
	id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name        TEXT NOT NULL,
	type        TEXT,
	region      TEXT NOT NULL,
	scenery     TEXT,
	effort      TEXT,
	duration    TEXT,
	description TEXT,
	xp_value    INTEGER,
	lat         DOUBLE PRECISION,
	lng         DOUBLE PRECISION
);

CREATE TABLE IF NOT EXISTS users (
	id                       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id                  TEXT NOT NULL UNIQUE,
	total_xp                 INTEGER NOT NULL DEFAULT 0,
	email                    TEXT UNIQUE,
	password_hash            TEXT,
	reroll_tokens            INTEGER NOT NULL DEFAULT 5,
	reroll_reset_at          TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '1 day'),
	last_quest_completed_at  DATE
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS journal_entries (
	id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id      TEXT NOT NULL,
	adventure_id BIGINT NOT NULL REFERENCES adventures(id),
	text         TEXT NOT NULL,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS visited_adventures (
	id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id      TEXT NOT NULL,
	adventure_id BIGINT NOT NULL REFERENCES adventures(id),
	lat          DOUBLE PRECISION NOT NULL,
	lng          DOUBLE PRECISION NOT NULL,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_visited_user_adventure
	ON visited_adventures(user_id, adventure_id);

CREATE TABLE IF NOT EXISTS user_achievements (
	user_id        TEXT NOT NULL,
	achievement_id TEXT NOT NULL,
	unlocked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (user_id, achievement_id)
);

CREATE TABLE IF NOT EXISTS trail_points (
	id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	user_id     TEXT NOT NULL,
	lat         DOUBLE PRECISION NOT NULL,
	lng         DOUBLE PRECISION NOT NULL,
	recorded_at TIMESTAMPTZ NOT NULL,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trail_points_user ON trail_points(user_id, recorded_at);

CREATE TABLE IF NOT EXISTS schema_migrations (
	version    INTEGER PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

// migration is a versioned, one-way schema change applied after the base
// schema above. Add new entries here (in increasing version order) whenever
// the schema needs to change for an already-deployed database; each up func
// runs inside its own transaction.
type migration struct {
	version int
	name    string
	up      func(tx *sql.Tx) error
}

var migrations = []migration{
	{
		version: 1,
		name:    "add_reroll_tokens",
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE users
					ADD COLUMN IF NOT EXISTS reroll_tokens INTEGER NOT NULL DEFAULT 5,
					ADD COLUMN IF NOT EXISTS reroll_reset_at TIMESTAMPTZ NOT NULL DEFAULT now();
			`)
			return err
		},
	},
	{
		version: 2,
		name:    "add_user_achievements",
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS user_achievements (
					user_id        TEXT NOT NULL,
					achievement_id TEXT NOT NULL,
					unlocked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
					PRIMARY KEY (user_id, achievement_id)
				);
			`)
			return err
		},
	},
	{
		version: 3,
		name:    "add_daily_quest_tracking",
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE users
					ADD COLUMN IF NOT EXISTS last_quest_completed_at DATE;
			`)
			return err
		},
	},
	{
		version: 4,
		name:    "fix_reroll_reset_at_default",
		up: func(tx *sql.Tx) error {
			// reroll_reset_at previously defaulted to now(), so a freshly
			// created user row was born already "expired" per the
			// `reroll_reset_at <= now()` check in consumeReroll/getRerollStatus
			// — the very next read reset reroll_tokens back to the daily
			// allowance, silently discarding any bonus already credited.
			_, err := tx.Exec(`
				ALTER TABLE users
					ALTER COLUMN reroll_reset_at SET DEFAULT (now() + interval '1 day');
			`)
			return err
		},
	},
	{
		version: 5,
		name:    "add_trail_points",
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE IF NOT EXISTS trail_points (
					id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
					user_id     TEXT NOT NULL,
					lat         DOUBLE PRECISION NOT NULL,
					lng         DOUBLE PRECISION NOT NULL,
					recorded_at TIMESTAMPTZ NOT NULL,
					created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
				);
				CREATE INDEX IF NOT EXISTS idx_trail_points_user ON trail_points(user_id, recorded_at);
			`)
			return err
		},
	},
}

// InitDB opens the Postgres database at the given connection string, ensures
// the base schema exists, and runs any outstanding migrations.
func InitDB(connString string) *sql.DB {
	db, err := sql.Open("pgx", connString)
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
	log.Println("Connected to Postgres")
	return db
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

// migrate brings a database from whatever it recorded in schema_migrations up
// to the latest version in migrations, running each pending step in order
// inside its own transaction.
func migrate(db *sql.DB) error {
	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := m.up(tx); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, m.version); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
