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

// PostgreSQL schema for vaults
const postgresVaultsSchemaSQL = `
CREATE TABLE IF NOT EXISTS vaults (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vaults_name ON vaults(name);

CREATE TABLE IF NOT EXISTS vault_entries (
    id SERIAL PRIMARY KEY,
    vault_id INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value BYTEA NOT NULL,
    is_secret BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(vault_id, key)
);

CREATE INDEX IF NOT EXISTS idx_vault_entries_vault_id ON vault_entries(vault_id);
CREATE INDEX IF NOT EXISTS idx_vault_entries_key ON vault_entries(vault_id, key);
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

// SQLite schema for vaults
const sqliteVaultsSchemaSQL = `
CREATE TABLE IF NOT EXISTS vaults (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vaults_name ON vaults(name);

CREATE TABLE IF NOT EXISTS vault_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vault_id INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value BLOB NOT NULL,
    is_secret INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(vault_id, key)
);

CREATE INDEX IF NOT EXISTS idx_vault_entries_vault_id ON vault_entries(vault_id);
CREATE INDEX IF NOT EXISTS idx_vault_entries_key ON vault_entries(vault_id, key);
`

// EnsureSchema checks if required tables exist and creates them if they don't.
// This is called at application startup to ensure the database is ready.
func EnsureSchema(db *sqlx.DB) error {
	exists, err := tableExists(db, "queue_workers_config")
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists {
		log.Println("[SCHEMA] Table 'queue_workers_config' already exists")
	} else {
		log.Println("[SCHEMA] Creating table 'queue_workers_config'...")
		if err := createSchema(db); err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		log.Println("[SCHEMA] Table 'queue_workers_config' created successfully")
	}

	// Check and create vault tables
	vaultsExist, err := tableExists(db, "vaults")
	if err != nil {
		return fmt.Errorf("failed to check vaults table existence: %w", err)
	}

	if vaultsExist {
		log.Println("[SCHEMA] Vaults tables already exist")
	} else {
		log.Println("[SCHEMA] Creating vaults tables...")
		if err := createVaultsSchema(db); err != nil {
			return fmt.Errorf("failed to create vaults schema: %w", err)
		}
		log.Println("[SCHEMA] Vaults tables created successfully")
	}

	return nil
}

// tableExists checks if a table exists (works for both PostgreSQL and SQLite)
func tableExists(db *sqlx.DB, tableName string) (bool, error) {
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

	err := db.Get(&exists, query, tableName)
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

// createVaultsSchema creates the vaults tables based on the database driver
func createVaultsSchema(db *sqlx.DB) error {
	driverName := db.DriverName()

	var schemaSQL string
	if driverName == "sqlite" {
		schemaSQL = sqliteVaultsSchemaSQL
	} else {
		schemaSQL = postgresVaultsSchemaSQL
	}

	_, err := db.Exec(schemaSQL)
	return err
}
