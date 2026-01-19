package queueworker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// QueueWorkerRow represents a row in the queue_workers_config table
type QueueWorkerRow struct {
	ID         int             `db:"id"`
	Name       sql.NullString  `db:"name"`
	Priority   sql.NullInt64   `db:"priority"`
	Deployment json.RawMessage `db:"deployment"`
}

// Store handles PostgreSQL operations for queue configurations
// Redis is still used for asynq-related operations
type Store struct {
	db  *sqlx.DB
	rdb *redis.Client
}

// NewStore creates a new queue worker store
func NewStore(db *sqlx.DB, rdb *redis.Client) *Store {
	return &Store{db: db, rdb: rdb}
}

// Save stores or updates a queue configuration in PostgreSQL
func (s *Store) Save(ctx context.Context, cfg *QueueConfig) error {
	// Marshal deployment if present
	var deploymentJSON sql.NullString
	var err error
	if cfg.Deployment != nil {
		b, err := json.Marshal(cfg.Deployment)
		if err != nil {
			return fmt.Errorf("failed to marshal deployment: %w", err)
		}
		deploymentJSON = sql.NullString{String: string(b), Valid: true}
	} else {
		// The deployment can be nil if it's only a key API for external services (think Google or OpenAI etc.)
		deploymentJSON = sql.NullString{Valid: false}
	}
	query := `
		INSERT INTO queue_workers_config (name, priority, deployment)
		VALUES ($1, $2, $3)
		ON CONFLICT (name) DO UPDATE SET
			priority = EXCLUDED.priority,
			deployment = EXCLUDED.deployment
	`

	_, err = s.db.ExecContext(ctx, query,
		cfg.Name,
		cfg.Priority,
		deploymentJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// Get retrieves a queue configuration from PostgreSQL
func (s *Store) Get(ctx context.Context, queueName string) (*QueueConfig, error) {
	query := `
		SELECT id, name, priority, deployment
		FROM queue_workers_config
		WHERE name = $1
	`

	var row QueueWorkerRow
	err := s.db.GetContext(ctx, &row, query, queueName)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	cfg := &QueueConfig{
		Name: row.Name.String,
	}

	if row.Priority.Valid {
		cfg.Priority = int(row.Priority.Int64)
	}

	// Unmarshal deployment if present
	if len(row.Deployment) > 0 && string(row.Deployment) != "null" {
		var deployment DeploymentConfig
		if err := json.Unmarshal(row.Deployment, &deployment); err == nil {
			cfg.Deployment = &deployment
		}
	}

	return cfg, nil
}

// ListQueues returns all registered queue names
func (s *Store) ListQueues(ctx context.Context) ([]string, error) {
	query := `SELECT name FROM queue_workers_config ORDER BY name`

	var names []sql.NullString
	err := s.db.SelectContext(ctx, &names, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	queues := make([]string, 0, len(names))
	for _, n := range names {
		if n.Valid {
			queues = append(queues, n.String)
		}
	}

	return queues, nil
}

// Delete removes a queue configuration from PostgreSQL
func (s *Store) Delete(ctx context.Context, queueName string) error {
	query := `DELETE FROM queue_workers_config WHERE name = $1`
	_, err := s.db.ExecContext(ctx, query, queueName)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// ListByPrefix returns all queue configurations that start with the given prefix
func (s *Store) ListByPrefix(ctx context.Context, prefix string) ([]*QueueConfig, error) {
	query := `
		SELECT id, name, priority, deployment
		FROM queue_workers_config
		WHERE name LIKE $1
		ORDER BY priority DESC
	`

	var rows []QueueWorkerRow
	err := s.db.SelectContext(ctx, &rows, query, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to list queues by prefix: %w", err)
	}

	configs := make([]*QueueConfig, 0, len(rows))
	for _, row := range rows {
		cfg := &QueueConfig{
			Name: row.Name.String,
		}

		if row.Priority.Valid {
			cfg.Priority = int(row.Priority.Int64)
		}

		// Unmarshal deployment if present
		if len(row.Deployment) > 0 && string(row.Deployment) != "null" {
			var deployment DeploymentConfig
			if err := json.Unmarshal(row.Deployment, &deployment); err == nil {
				cfg.Deployment = &deployment
			}
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// Exists checks if a queue configuration exists
func (s *Store) Exists(ctx context.Context, queueName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM queue_workers_config WHERE name = $1)`

	var exists bool
	err := s.db.GetContext(ctx, &exists, query, queueName)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

// --- Redis-only operations (for asynq compatibility) ---

const (
	workerKeyPattern = "asynq:workers:*"
	staleThreshold   = 30 // seconds
	asynqQueuesKey   = "asynq:queues"
)

// RegisterAsynqQueues adds queue names to asynq's queue registry so they appear in asynqmon
func (s *Store) RegisterAsynqQueues(ctx context.Context, queueNames []string) error {
	if len(queueNames) == 0 {
		return nil
	}
	args := make([]interface{}, len(queueNames))
	for i, name := range queueNames {
		args[i] = name
	}
	return s.rdb.SAdd(ctx, asynqQueuesKey, args...).Err()
}

// CleanupStaleWorkers removes worker entries that haven't sent a heartbeat recently
func (s *Store) CleanupStaleWorkers(ctx context.Context) (int, error) {
	// Scan for all worker keys
	var workerKeys []string
	iter := s.rdb.Scan(ctx, 0, workerKeyPattern, 100).Iterator()
	for iter.Next(ctx) {
		workerKeys = append(workerKeys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("failed to scan workers: %w", err)
	}

	now := time.Now().Unix()
	deleted := 0

	for _, key := range workerKeys {
		// Check if it's a HASH type
		keyType, err := s.rdb.Type(ctx, key).Result()
		if err != nil || keyType != "hash" {
			continue
		}

		// Get last_beat
		lastBeatStr, err := s.rdb.HGet(ctx, key, "last_beat").Result()
		if err != nil {
			// No last_beat field, delete it
			s.rdb.Del(ctx, key)
			deleted++
			continue
		}

		lastBeat, err := strconv.ParseInt(lastBeatStr, 10, 64)
		if err != nil {
			continue
		}

		// If stale, delete it
		if now-lastBeat > staleThreshold {
			s.rdb.Del(ctx, key)
			deleted++
		}
	}

	return deleted, nil
}
