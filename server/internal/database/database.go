package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var currentVersion = 1

func Init(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	// Create schema version table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	// Check current version
	var version int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("check schema version: %w", err)
	}

	if version >= currentVersion {
		return nil
	}

	// Apply migrations
	for v := version + 1; v <= currentVersion; v++ {
		log.Printf("Applying database migration v%d", v)
		migration := getMigration(v)
		if migration == "" {
			return fmt.Errorf("unknown migration version: %d", v)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin transaction for v%d: %w", v, err)
		}

		if _, err := tx.Exec(migration); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration v%d: %w", v, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", v); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration v%d: %w", v, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration v%d: %w", v, err)
		}
	}

	return nil
}

func getMigration(version int) string {
	switch version {
	case 1:
		return `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	is_admin INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS invite_codes (
	code TEXT PRIMARY KEY,
	description TEXT DEFAULT '',
	max_uses INTEGER DEFAULT 1,
	used INTEGER DEFAULT 0,
	expires_at TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS media (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	description TEXT DEFAULT '',
	type TEXT NOT NULL CHECK(type IN ('video', 'audio')),
	path TEXT UNIQUE NOT NULL,
	duration REAL DEFAULT 0,
	resolution TEXT DEFAULT '',
	cover_path TEXT DEFAULT '',
	sprite_path TEXT DEFAULT '',
	sprite_meta TEXT DEFAULT '',
	file_size INTEGER DEFAULT 0,
	codec TEXT DEFAULT '',
	bitrate INTEGER DEFAULT 0,
	deleted INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tags (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT UNIQUE NOT NULL,
	color TEXT DEFAULT '#6C5CE7'
);

CREATE TABLE IF NOT EXISTS media_tags (
	media_id INTEGER NOT NULL,
	tag_id INTEGER NOT NULL,
	PRIMARY KEY (media_id, tag_id),
	FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE,
	FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS playlists (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	folder_path TEXT DEFAULT '',
	user_id INTEGER NOT NULL,
	play_mode TEXT DEFAULT 'sequential' CHECK(play_mode IN ('sequential', 'loop', 'random', 'single-loop')),
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS playlist_items (
	playlist_id INTEGER NOT NULL,
	media_id INTEGER NOT NULL,
	position INTEGER NOT NULL,
	PRIMARY KEY (playlist_id, media_id),
	FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
	FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS scan_folders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT UNIQUE NOT NULL,
	last_scan_at DATETIME,
	status TEXT DEFAULT 'idle' CHECK(status IN ('idle', 'scanning', 'error'))
);

CREATE TABLE IF NOT EXISTS device_codes (
	code TEXT PRIMARY KEY,
	user_id INTEGER,
	expires_at DATETIME NOT NULL,
	used INTEGER DEFAULT 0,
	FOREIGN KEY (user_id) REFERENCES users(id)
);

-- FTS5 full-text search virtual table
CREATE VIRTUAL TABLE IF NOT EXISTS media_fts USING fts5(
	title,
	description,
	tags,
	content='media',
	content_rowid='id',
	tokenize='unicode61'
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_media_type ON media(type);
CREATE INDEX IF NOT EXISTS idx_media_deleted ON media(deleted);
CREATE INDEX IF NOT EXISTS idx_media_created ON media(created_at);
CREATE INDEX IF NOT EXISTS idx_playlists_user ON playlists(user_id);
CREATE INDEX IF NOT EXISTS idx_media_tags_tag ON media_tags(tag_id);
`
	}
	return ""
}
