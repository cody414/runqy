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

// QueueRow represents a row in the queues table
type QueueRow struct {
	ID           int            `db:"id"`
	Name         sql.NullString `db:"name"`
	Provider     sql.NullString `db:"provider"`
	Deployment   sql.NullString `db:"deployment"`     // JSON stored as TEXT for SQLite compatibility
	InputSchema  sql.NullString `db:"input_schema"`   // JSON stored as TEXT
	OutputSchema sql.NullString `db:"output_schema"`  // JSON stored as TEXT
	Description  sql.NullString `db:"description"`
	Enabled      sql.NullBool   `db:"enabled"`
	CreatedAt    sql.NullTime   `db:"created_at"`
	UpdatedAt    sql.NullTime   `db:"updated_at"`
}

// SubQueueRow represents a row in the sub_queues table
type SubQueueRow struct {
	ID        int            `db:"id"`
	QueueID   int            `db:"queue_id"`
	Name      sql.NullString `db:"name"`
	Priority  sql.NullInt64  `db:"priority"`
	CreatedAt sql.NullTime   `db:"created_at"`
	UpdatedAt sql.NullTime   `db:"updated_at"`
}

// Store handles database operations for queue configurations (PostgreSQL or SQLite)
// Redis is still used for asynq-related operations
type Store struct {
	db  *sqlx.DB
	rdb *redis.Client
}

// NewStore creates a new queue worker store
func NewStore(db *sqlx.DB, rdb *redis.Client) *Store {
	return &Store{db: db, rdb: rdb}
}

// SaveQueue stores or updates a parent queue in the database
func (s *Store) SaveQueue(ctx context.Context, queue *Queue) (int, error) {
	// Marshal deployment if present
	var deploymentJSON sql.NullString
	if queue.Deployment != nil {
		b, err := json.Marshal(queue.Deployment)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal deployment: %w", err)
		}
		deploymentJSON = sql.NullString{String: string(b), Valid: true}
	}

	// Marshal input_schema if present
	var inputSchemaJSON sql.NullString
	if len(queue.InputSchema) > 0 {
		b, err := json.Marshal(queue.InputSchema)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal input_schema: %w", err)
		}
		inputSchemaJSON = sql.NullString{String: string(b), Valid: true}
	}

	// Marshal output_schema if present
	var outputSchemaJSON sql.NullString
	if len(queue.OutputSchema) > 0 {
		b, err := json.Marshal(queue.OutputSchema)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal output_schema: %w", err)
		}
		outputSchemaJSON = sql.NullString{String: string(b), Valid: true}
	}

	// Use different queries for PostgreSQL and SQLite
	driverName := s.db.DriverName()

	if driverName == "sqlite" {
		// SQLite: use INSERT OR REPLACE and get ID separately
		query := `
			INSERT INTO queues (name, provider, deployment, input_schema, output_schema, description, enabled)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (name) DO UPDATE SET
				provider = excluded.provider,
				deployment = excluded.deployment,
				input_schema = excluded.input_schema,
				output_schema = excluded.output_schema,
				description = excluded.description,
				enabled = excluded.enabled,
				updated_at = CURRENT_TIMESTAMP
		`
		_, err := s.db.ExecContext(ctx, query,
			queue.Name,
			queue.Provider,
			deploymentJSON,
			inputSchemaJSON,
			outputSchemaJSON,
			queue.Description,
			queue.Enabled,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to save queue: %w", err)
		}

		// Get the ID
		var id int
		err = s.db.GetContext(ctx, &id, `SELECT id FROM queues WHERE name = ?`, queue.Name)
		if err != nil {
			return 0, fmt.Errorf("failed to get queue id: %w", err)
		}
		return id, nil
	}

	// PostgreSQL: use RETURNING
	query := s.db.Rebind(`
		INSERT INTO queues (name, provider, deployment, input_schema, output_schema, description, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (name) DO UPDATE SET
			provider = excluded.provider,
			deployment = excluded.deployment,
			input_schema = excluded.input_schema,
			output_schema = excluded.output_schema,
			description = excluded.description,
			enabled = excluded.enabled,
			updated_at = NOW()
		RETURNING id
	`)

	var id int
	err := s.db.GetContext(ctx, &id, query,
		queue.Name,
		queue.Provider,
		deploymentJSON,
		inputSchemaJSON,
		outputSchemaJSON,
		queue.Description,
		queue.Enabled,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save queue: %w", err)
	}

	return id, nil
}

// SaveSubQueue stores or updates a sub-queue in the database
func (s *Store) SaveSubQueue(ctx context.Context, queueID int, subQueue *SubQueue) error {
	driverName := s.db.DriverName()

	var query string
	if driverName == "sqlite" {
		query = `
			INSERT INTO sub_queues (queue_id, name, priority)
			VALUES (?, ?, ?)
			ON CONFLICT (queue_id, name) DO UPDATE SET
				priority = excluded.priority,
				updated_at = CURRENT_TIMESTAMP
		`
	} else {
		query = s.db.Rebind(`
			INSERT INTO sub_queues (queue_id, name, priority)
			VALUES (?, ?, ?)
			ON CONFLICT (queue_id, name) DO UPDATE SET
				priority = excluded.priority,
				updated_at = NOW()
		`)
	}

	_, err := s.db.ExecContext(ctx, query,
		queueID,
		subQueue.Name,
		subQueue.Priority,
	)
	if err != nil {
		return fmt.Errorf("failed to save sub-queue: %w", err)
	}

	return nil
}

// Save stores or updates a queue configuration in the database (backward compatibility)
// Deprecated: Use SaveQueue and SaveSubQueue instead
func (s *Store) Save(ctx context.Context, cfg *QueueConfig) error {
	// Parse the queue name to get parent and sub-queue parts
	parent, subQueueName, hasSubQueue := ParseQueueName(cfg.Name)
	if !hasSubQueue {
		subQueueName = DefaultSubQueueName
	}

	// Create or update the parent queue
	queue := &Queue{
		Name:        parent,
		Provider:    cfg.Provider,
		Deployment:  cfg.Deployment,
		Enabled:     true,
		Description: "",
	}

	queueID, err := s.SaveQueue(ctx, queue)
	if err != nil {
		return err
	}

	// Create or update the sub-queue
	subQueue := &SubQueue{
		QueueID:  queueID,
		Name:     subQueueName,
		Priority: cfg.Priority,
	}

	return s.SaveSubQueue(ctx, queueID, subQueue)
}

// GetQueue retrieves a parent queue by name
func (s *Store) GetQueue(ctx context.Context, queueName string) (*Queue, error) {
	query := s.db.Rebind(`
		SELECT id, name, provider, deployment, input_schema, output_schema, description, enabled, created_at, updated_at
		FROM queues
		WHERE name = ?
	`)

	var row QueueRow
	err := s.db.GetContext(ctx, &row, query, queueName)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}

	return s.rowToQueue(&row), nil
}

// GetQueueByID retrieves a parent queue by ID
func (s *Store) GetQueueByID(ctx context.Context, id int) (*Queue, error) {
	query := s.db.Rebind(`
		SELECT id, name, provider, deployment, input_schema, output_schema, description, enabled, created_at, updated_at
		FROM queues
		WHERE id = ?
	`)

	var row QueueRow
	err := s.db.GetContext(ctx, &row, query, id)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}

	return s.rowToQueue(&row), nil
}

// rowToQueue converts a QueueRow to a Queue struct
func (s *Store) rowToQueue(row *QueueRow) *Queue {
	queue := &Queue{
		ID:      row.ID,
		Enabled: true, // Default
	}

	if row.Name.Valid {
		queue.Name = row.Name.String
	}
	if row.Provider.Valid {
		queue.Provider = row.Provider.String
	}
	if row.Description.Valid {
		queue.Description = row.Description.String
	}
	if row.Enabled.Valid {
		queue.Enabled = row.Enabled.Bool
	}
	if row.CreatedAt.Valid {
		queue.CreatedAt = row.CreatedAt.Time.Unix()
	}
	if row.UpdatedAt.Valid {
		queue.UpdatedAt = row.UpdatedAt.Time.Unix()
	}

	// Unmarshal deployment if present
	if row.Deployment.Valid && row.Deployment.String != "" && row.Deployment.String != "null" {
		var deployment DeploymentConfig
		if err := json.Unmarshal([]byte(row.Deployment.String), &deployment); err == nil {
			queue.Deployment = &deployment
		}
	}

	// Unmarshal input_schema if present
	if row.InputSchema.Valid && row.InputSchema.String != "" && row.InputSchema.String != "null" {
		var inputSchema []FieldSchema
		if err := json.Unmarshal([]byte(row.InputSchema.String), &inputSchema); err == nil {
			queue.InputSchema = inputSchema
		}
	}

	// Unmarshal output_schema if present
	if row.OutputSchema.Valid && row.OutputSchema.String != "" && row.OutputSchema.String != "null" {
		var outputSchema []FieldSchema
		if err := json.Unmarshal([]byte(row.OutputSchema.String), &outputSchema); err == nil {
			queue.OutputSchema = outputSchema
		}
	}

	return queue
}

// ListSubQueues retrieves all sub-queues for a parent queue
func (s *Store) ListSubQueues(ctx context.Context, queueID int) ([]SubQueue, error) {
	query := s.db.Rebind(`
		SELECT id, queue_id, name, priority, created_at, updated_at
		FROM sub_queues
		WHERE queue_id = ?
		ORDER BY priority DESC
	`)

	var rows []SubQueueRow
	err := s.db.SelectContext(ctx, &rows, query, queueID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-queues: %w", err)
	}

	subQueues := make([]SubQueue, 0, len(rows))
	for _, row := range rows {
		sq := SubQueue{
			ID:       row.ID,
			QueueID:  row.QueueID,
			Priority: 1, // Default
		}
		if row.Name.Valid {
			sq.Name = row.Name.String
		}
		if row.Priority.Valid {
			sq.Priority = int(row.Priority.Int64)
		}
		if row.CreatedAt.Valid {
			sq.CreatedAt = row.CreatedAt.Time.Unix()
		}
		if row.UpdatedAt.Valid {
			sq.UpdatedAt = row.UpdatedAt.Time.Unix()
		}
		subQueues = append(subQueues, sq)
	}

	return subQueues, nil
}

// GetQueueWithSubQueues retrieves a parent queue with all its sub-queues
func (s *Store) GetQueueWithSubQueues(ctx context.Context, queueName string) (*QueueWithSubQueues, error) {
	queue, err := s.GetQueue(ctx, queueName)
	if err != nil {
		return nil, err
	}
	if queue == nil {
		return nil, nil
	}

	subQueues, err := s.ListSubQueues(ctx, queue.ID)
	if err != nil {
		return nil, err
	}

	return &QueueWithSubQueues{
		Queue:     *queue,
		SubQueues: subQueues,
	}, nil
}

// Get retrieves a queue configuration from the database (backward compatibility)
// Returns a QueueConfig for a full queue name (e.g., "inference.high")
func (s *Store) Get(ctx context.Context, queueName string) (*QueueConfig, error) {
	// Parse the full queue name to get parent and sub-queue parts
	parent, subQueueName, hasSubQueue := ParseQueueName(queueName)
	if !hasSubQueue {
		subQueueName = DefaultSubQueueName
	}

	// Get the parent queue
	queue, err := s.GetQueue(ctx, parent)
	if err != nil {
		return nil, err
	}
	if queue == nil {
		return nil, nil
	}

	// Get sub-queues for this parent
	subQueues, err := s.ListSubQueues(ctx, queue.ID)
	if err != nil {
		return nil, err
	}

	// Find the matching sub-queue
	var matchingSubQueue *SubQueue
	for i := range subQueues {
		if subQueues[i].Name == subQueueName {
			matchingSubQueue = &subQueues[i]
			break
		}
	}

	if matchingSubQueue == nil {
		return nil, nil // Sub-queue not found
	}

	// Build the QueueConfig
	return &QueueConfig{
		Name:       queueName,
		Priority:   matchingSubQueue.Priority,
		Provider:   queue.Provider,
		Deployment: queue.Deployment,
		CreatedAt:  queue.CreatedAt,
		UpdatedAt:  queue.UpdatedAt,
	}, nil
}

// ListParentQueues returns all parent queue names
func (s *Store) ListParentQueues(ctx context.Context) ([]string, error) {
	query := `SELECT name FROM queues ORDER BY name`

	type nameRow struct {
		Name string `db:"name"`
	}
	var rows []nameRow
	err := s.db.SelectContext(ctx, &rows, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list parent queues: %w", err)
	}

	queues := make([]string, 0, len(rows))
	for _, r := range rows {
		queues = append(queues, r.Name)
	}

	return queues, nil
}

// ListAllQueuesWithSubQueues returns all parent queues with their sub-queues
func (s *Store) ListAllQueuesWithSubQueues(ctx context.Context) ([]QueueWithSubQueues, error) {
	parentQueues, err := s.ListParentQueues(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]QueueWithSubQueues, 0, len(parentQueues))
	for _, name := range parentQueues {
		qws, err := s.GetQueueWithSubQueues(ctx, name)
		if err != nil {
			return nil, err
		}
		if qws != nil {
			result = append(result, *qws)
		}
	}

	return result, nil
}

// ListQueues returns all registered full queue names (parent.subqueue format)
// This is for backward compatibility - returns names like "inference.high", "inference.low"
func (s *Store) ListQueues(ctx context.Context) ([]string, error) {
	query := `
		SELECT q.name as parent_name, sq.name as sub_name
		FROM queues q
		JOIN sub_queues sq ON sq.queue_id = q.id
		ORDER BY q.name, sq.priority DESC
	`

	type joinRow struct {
		ParentName string `db:"parent_name"`
		SubName    string `db:"sub_name"`
	}
	var rows []joinRow
	err := s.db.SelectContext(ctx, &rows, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list queues: %w", err)
	}

	queues := make([]string, 0, len(rows))
	for _, r := range rows {
		fullName := BuildFullQueueName(r.ParentName, r.SubName)
		queues = append(queues, fullName)
	}

	return queues, nil
}

// DeleteQueue removes a parent queue and all its sub-queues (via cascade)
func (s *Store) DeleteQueue(ctx context.Context, queueName string) error {
	query := s.db.Rebind(`DELETE FROM queues WHERE name = ?`)
	_, err := s.db.ExecContext(ctx, query, queueName)
	if err != nil {
		return fmt.Errorf("failed to delete queue: %w", err)
	}
	return nil
}

// DeleteSubQueue removes a specific sub-queue
func (s *Store) DeleteSubQueue(ctx context.Context, queueID int, subQueueName string) error {
	query := s.db.Rebind(`DELETE FROM sub_queues WHERE queue_id = ? AND name = ?`)
	_, err := s.db.ExecContext(ctx, query, queueID, subQueueName)
	if err != nil {
		return fmt.Errorf("failed to delete sub-queue: %w", err)
	}
	return nil
}

// Delete removes a queue configuration from the database (backward compatibility)
// If queueName is a full name (e.g., "inference.high"), deletes that sub-queue
// If it's a parent name (e.g., "inference"), deletes the parent and all sub-queues
func (s *Store) Delete(ctx context.Context, queueName string) error {
	parent, subQueueName, hasSubQueue := ParseQueueName(queueName)

	if !hasSubQueue {
		// Delete the entire parent queue (cascades to sub-queues)
		return s.DeleteQueue(ctx, parent)
	}

	// Delete just the sub-queue
	queue, err := s.GetQueue(ctx, parent)
	if err != nil {
		return err
	}
	if queue == nil {
		return nil // Parent not found, nothing to delete
	}

	return s.DeleteSubQueue(ctx, queue.ID, subQueueName)
}

// ListByPrefix returns all queue configurations that start with the given prefix
// prefix should be like "inference." to get all sub-queues of the "inference" parent
func (s *Store) ListByPrefix(ctx context.Context, prefix string) ([]*QueueConfig, error) {
	// Extract parent queue name from prefix (remove trailing separator)
	parentName := prefix
	if len(prefix) > 0 && prefix[len(prefix)-1] == '.' {
		parentName = prefix[:len(prefix)-1]
	}

	// Get the parent queue with its sub-queues
	qws, err := s.GetQueueWithSubQueues(ctx, parentName)
	if err != nil {
		return nil, err
	}
	if qws == nil {
		return []*QueueConfig{}, nil
	}

	// Convert to QueueConfig slice
	configs := make([]*QueueConfig, 0, len(qws.SubQueues))
	for _, sq := range qws.SubQueues {
		cfg := &QueueConfig{
			Name:       BuildFullQueueName(qws.Name, sq.Name),
			Priority:   sq.Priority,
			Provider:   qws.Provider,
			Deployment: qws.Deployment,
			CreatedAt:  qws.CreatedAt,
			UpdatedAt:  qws.UpdatedAt,
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

// QueueExists checks if a parent queue exists
func (s *Store) QueueExists(ctx context.Context, queueName string) (bool, error) {
	query := s.db.Rebind(`SELECT EXISTS(SELECT 1 FROM queues WHERE name = ?)`)

	var exists bool
	err := s.db.GetContext(ctx, &exists, query, queueName)
	if err != nil {
		return false, fmt.Errorf("failed to check queue existence: %w", err)
	}
	return exists, nil
}

// SubQueueExists checks if a sub-queue exists
func (s *Store) SubQueueExists(ctx context.Context, queueID int, subQueueName string) (bool, error) {
	query := s.db.Rebind(`SELECT EXISTS(SELECT 1 FROM sub_queues WHERE queue_id = ? AND name = ?)`)

	var exists bool
	err := s.db.GetContext(ctx, &exists, query, queueID, subQueueName)
	if err != nil {
		return false, fmt.Errorf("failed to check sub-queue existence: %w", err)
	}
	return exists, nil
}

// Exists checks if a queue configuration exists (backward compatibility)
// Works for both full names (e.g., "inference.high") and parent names (e.g., "inference")
func (s *Store) Exists(ctx context.Context, queueName string) (bool, error) {
	parent, subQueueName, hasSubQueue := ParseQueueName(queueName)

	// Check if parent queue exists
	queue, err := s.GetQueue(ctx, parent)
	if err != nil {
		return false, err
	}
	if queue == nil {
		return false, nil
	}

	if !hasSubQueue {
		// Just checking for parent queue existence
		return true, nil
	}

	// Check if specific sub-queue exists
	return s.SubQueueExists(ctx, queue.ID, subQueueName)
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

// UnregisterAsynqQueues removes queue names from asynq's queue registry
func (s *Store) UnregisterAsynqQueues(ctx context.Context, queueNames []string) error {
	if len(queueNames) == 0 {
		return nil
	}
	args := make([]interface{}, len(queueNames))
	for i, name := range queueNames {
		args[i] = name
	}
	return s.rdb.SRem(ctx, asynqQueuesKey, args...).Err()
}

// SyncConfigsToAsynq ensures all configs in the database are registered in asynq
func (s *Store) SyncConfigsToAsynq(ctx context.Context) error {
	queues, err := s.ListQueues(ctx)
	if err != nil {
		return err
	}
	if len(queues) > 0 {
		return s.RegisterAsynqQueues(ctx, queues)
	}
	return nil
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
