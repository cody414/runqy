package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ****************************************************************************
// This file defines:
//   - http.Handler(s) for external workers (runqy-worker) endpoints
//   - Reads from asynq:workers:* keys (HASH type) set by runqy-worker
// ****************************************************************************

// workerData represents an external worker from asynq:workers:* keys
type workerData struct {
	WorkerID    string `json:"worker_id"`
	StartedAt   int64  `json:"started_at"`
	LastBeat    int64  `json:"last_beat"`
	Concurrency int    `json:"concurrency"`
	Queues      string `json:"queues"`
	Status      string `json:"status"`
	IsStale     bool   `json:"is_stale"` // True if no heartbeat for 30+ seconds
}

type listWorkersResponse struct {
	Workers []*workerData `json:"workers"`
}

// newListWorkersHandlerFunc returns a handler that lists external workers.
// External workers (like runqy-worker) register themselves at asynq:workers:{id} keys.
func newListWorkersHandlerFunc(rc redis.UniversalClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		// Scan for all worker keys matching pattern asynq:workers:*
		var workerKeys []string
		iter := rc.Scan(ctx, 0, "asynq:workers:*", 100).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			// Skip if it's not a worker data key (must have at least 2 colons)
			if strings.Count(key, ":") >= 2 {
				workerKeys = append(workerKeys, key)
			}
		}
		if err := iter.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		workers := make([]*workerData, 0, len(workerKeys))
		now := time.Now().Unix()

		for _, workerKey := range workerKeys {
			// Check if it's a HASH (worker data) - skip other types
			keyType, err := rc.Type(ctx, workerKey).Result()
			if err != nil || keyType != "hash" {
				continue
			}

			data, err := rc.HGetAll(ctx, workerKey).Result()
			if err != nil || len(data) == 0 {
				continue
			}

			// Extract worker ID from key (asynq:workers:{worker_id})
			workerID := strings.TrimPrefix(workerKey, "asynq:workers:")

			worker := &workerData{
				WorkerID: workerID,
				Status:   data["status"],
				Queues:   data["queues"],
			}

			if startedAt, err := strconv.ParseInt(data["started_at"], 10, 64); err == nil {
				worker.StartedAt = startedAt
			}
			if lastBeat, err := strconv.ParseInt(data["last_beat"], 10, 64); err == nil {
				worker.LastBeat = lastBeat
				// Consider stale if no heartbeat for 30 seconds
				worker.IsStale = (now - lastBeat) > 30
			}
			if concurrency, err := strconv.Atoi(data["concurrency"]); err == nil {
				worker.Concurrency = concurrency
			}

			workers = append(workers, worker)
		}

		resp := listWorkersResponse{Workers: workers}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
