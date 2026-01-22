package models

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// PostgreSQL schema
const postgresSchemaSQL = `
CREATE TABLE IF NOT EXISTS queue_workers_config (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    priority INTEGER,
    deployment JSONB
);

CREATE INDEX IF NOT EXISTS idx_queue_workers_config_name ON queue_workers_config(name);
`

// SQLite schema (uses TEXT instead of JSONB, INTEGER PRIMARY KEY AUTOINCREMENT instead of SERIAL)
const sqliteSchemaSQL = `
CREATE TABLE IF NOT EXISTS queue_workers_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    priority INTEGER,
    deployment TEXT
);

CREATE INDEX IF NOT EXISTS idx_queue_workers_config_name ON queue_workers_config(name);
`

// EnsureSchema checks if required tables exist and creates them if they don't.
// This is called at application startup to ensure the database is ready.
func EnsureSchema(db *sqlx.DB) error {
	exists, err := tableExists(db)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists {
		log.Println("[SCHEMA] Table 'queue_workers_config' already exists")
		return nil
	}

	log.Println("[SCHEMA] Creating table 'queue_workers_config'...")
	if err := createSchema(db); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("[SCHEMA] Table 'queue_workers_config' created successfully")
	return nil
}

// tableExists checks if a table exists (works for both PostgreSQL and SQLite)
func tableExists(db *sqlx.DB) (bool, error) {
	driverName := db.DriverName()

	var exists bool
	var query string

	if driverName == "sqlite" {
		// SQLite uses sqlite_master
		query = `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`
	} else {
		// PostgreSQL uses information_schema
		query = `
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = $1
			)
		`
	}

	err := db.Get(&exists, query, "queue_workers_config")
	if err != nil {
		return false, err
	}
	return exists, nil
}

// createSchema executes the schema creation SQL based on the database driver
func createSchema(db *sqlx.DB) error {
	driverName := db.DriverName()

	var schemaSQL string
	if driverName == "sqlite" {
		schemaSQL = sqliteSchemaSQL
	} else {
		schemaSQL = postgresSchemaSQL
	}

	_, err := db.Exec(schemaSQL)
	return err
}
