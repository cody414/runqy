package models

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// Schema SQL for queue_workers_config table
const schemaSQL = `
CREATE TABLE IF NOT EXISTS queue_workers_config (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    priority INTEGER,
    deployment JSONB
);

CREATE INDEX IF NOT EXISTS idx_queue_workers_config_name ON queue_workers_config(name);
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
		return nil
	}

	log.Println("[SCHEMA] Creating table 'queue_workers_config'...")
	if err := createSchema(db); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("[SCHEMA] Table 'queue_workers_config' created successfully")
	return nil
}

// tableExists checks if a table exists in the public schema
func tableExists(db *sqlx.DB, tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	err := db.Get(&exists, query, tableName)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// createSchema executes the schema creation SQL
func createSchema(db *sqlx.DB) error {
	_, err := db.Exec(schemaSQL)
	return err
}
