package models

import (
	"fmt"

	"github.com/Publikey/runqy/config"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// BuildDB creates a database connection based on configuration.
// If UseSQLite is true, it creates a SQLite connection.
// Otherwise, it creates a PostgreSQL connection.
func BuildDB(cfg *config.Config) (*sqlx.DB, error) {
	if cfg.UseSQLite {
		return buildSQLiteDB(cfg.SQLiteDBPath)
	}
	return BuildPostgresDB(cfg)
}

// buildSQLiteDB creates a SQLite database connection using sqlx
func buildSQLiteDB(dbPath string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite: %w", err)
	}

	// Enable foreign keys (disabled by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return db, nil
}
