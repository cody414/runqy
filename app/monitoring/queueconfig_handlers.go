package monitoring

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	queueworker "github.com/Publikey/runqy/queues"
	"github.com/gorilla/mux"
)

// QueueConfigListResponse is the response for listing queue configs
type QueueConfigListResponse struct {
	Queues []QueueConfigDetailResponse `json:"queues"`
	Count  int                         `json:"count"`
}

// CreateQueueConfigRequest is the request body for creating a queue config
type CreateQueueConfigRequest struct {
	Name       string                       `json:"name"`
	Priority   int                          `json:"priority"`
	Provider   string                       `json:"provider,omitempty"`
	Deployment *queueworker.DeploymentConfig `json:"deployment,omitempty"`
}

// QueueConfigDetailResponse is the full queue config response
type QueueConfigDetailResponse struct {
	Name       string                       `json:"name"`
	Priority   int                          `json:"priority"`
	Provider   string                       `json:"provider,omitempty"`
	Deployment *queueworker.DeploymentConfig `json:"deployment,omitempty"`
	CreatedAt  int64                        `json:"created_at"`
	UpdatedAt  int64                        `json:"updated_at"`
}

func newListQueueConfigsHandlerFunc(store *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queues, err := store.ListQueues(r.Context())
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Get detail for each queue
		details := make([]QueueConfigDetailResponse, 0, len(queues))
		for _, q := range queues {
			cfg, err := store.Get(r.Context(), q)
			if err != nil || cfg == nil {
				continue
			}
			details = append(details, QueueConfigDetailResponse{
				Name:       cfg.Name,
				Priority:   cfg.Priority,
				Provider:   cfg.Provider,
				Deployment: cfg.Deployment,
				CreatedAt:  cfg.CreatedAt,
				UpdatedAt:  cfg.UpdatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueueConfigListResponse{
			Queues: details,
			Count:  len(details),
		})
	}
}

func newGetQueueConfigHandlerFunc(store *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]

		cfg, err := store.Get(r.Context(), name)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if cfg == nil {
			http.Error(w, `{"error":"queue config not found"}`, http.StatusNotFound)
			return
		}

		resp := QueueConfigDetailResponse{
			Name:       cfg.Name,
			Priority:   cfg.Priority,
			Provider:   cfg.Provider,
			Deployment: cfg.Deployment,
			CreatedAt:  cfg.CreatedAt,
			UpdatedAt:  cfg.UpdatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func newCreateQueueConfigHandlerFunc(store *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateQueueConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[QUEUE-CONFIG] Error decoding request: %v", err)
			http.Error(w, `{"error":"invalid request body: `+err.Error()+`"}`, http.StatusBadRequest)
			return
		}

		log.Printf("[QUEUE-CONFIG] Create request: name=%s, priority=%d, hasDeployment=%v", req.Name, req.Priority, req.Deployment != nil)

		if req.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}
		if req.Priority < 1 {
			http.Error(w, `{"error":"priority must be at least 1"}`, http.StatusBadRequest)
			return
		}

		// Validate deployment if provided
		if req.Deployment != nil {
			if req.Deployment.GitURL == "" {
				http.Error(w, `{"error":"git_url is required for deployment"}`, http.StatusBadRequest)
				return
			}
			if req.Deployment.StartupCmd == "" {
				http.Error(w, `{"error":"startup_cmd is required for deployment"}`, http.StatusBadRequest)
				return
			}
		}

		ctx := r.Context()

		// Check for force query parameter
		force := r.URL.Query().Get("force") == "true"

		// Check if queue already exists
		exists, err := store.Exists(ctx, req.Name)
		if err != nil {
			log.Printf("[QUEUE-CONFIG] Error checking existence for %s: %v", req.Name, err)
			http.Error(w, `{"error":"failed to check queue existence: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		if exists && !force {
			http.Error(w, `{"error":"queue '`+req.Name+`' already exists. Use force=true to update"}`, http.StatusConflict)
			return
		}

		now := time.Now().Unix()
		cfg := &queueworker.QueueConfig{
			Name:       req.Name,
			Priority:   req.Priority,
			Provider:   req.Provider,
			Deployment: req.Deployment,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		log.Printf("[QUEUE-CONFIG] Saving config: %+v", cfg)
		if err := store.Save(ctx, cfg); err != nil {
			log.Printf("[QUEUE-CONFIG] Error saving %s: %v", req.Name, err)
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Register in asynq so it appears in monitoring
		if err := store.RegisterAsynqQueues(ctx, []string{req.Name}); err != nil {
			log.Printf("Warning: failed to register queue in asynq: %v", err)
		}

		message := "Queue created successfully"
		if exists {
			message = "Queue updated successfully"
		}

		log.Printf("[QUEUE-CONFIG] %s: %s (priority=%d)", message, req.Name, req.Priority)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": message,
			"queue":   cfg,
		})
	}
}

func newUpdateQueueConfigHandlerFunc(store *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]

		var req CreateQueueConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		// Check if queue exists
		existing, err := store.Get(ctx, name)
		if err != nil {
			http.Error(w, `{"error":"failed to get queue: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if existing == nil {
			http.Error(w, `{"error":"queue not found"}`, http.StatusNotFound)
			return
		}

		// Validate deployment if provided
		if req.Deployment != nil {
			if req.Deployment.GitURL == "" {
				http.Error(w, `{"error":"git_url is required for deployment"}`, http.StatusBadRequest)
				return
			}
			if req.Deployment.StartupCmd == "" {
				http.Error(w, `{"error":"startup_cmd is required for deployment"}`, http.StatusBadRequest)
				return
			}
		}

		// Update the config
		cfg := &queueworker.QueueConfig{
			Name:       name,
			Priority:   req.Priority,
			Provider:   req.Provider,
			Deployment: req.Deployment,
			CreatedAt:  existing.CreatedAt,
			UpdatedAt:  time.Now().Unix(),
		}

		if cfg.Priority < 1 {
			cfg.Priority = existing.Priority
		}

		if err := store.Save(ctx, cfg); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[QUEUE-CONFIG] Updated: %s (priority=%d)", name, cfg.Priority)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Queue updated successfully",
			"queue":   cfg,
		})
	}
}

func newDeleteQueueConfigHandlerFunc(store *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]

		ctx := r.Context()

		// Check if queue exists
		exists, err := store.Exists(ctx, name)
		if err != nil {
			http.Error(w, `{"error":"failed to check queue existence: `+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, `{"error":"queue '`+name+`' not found"}`, http.StatusNotFound)
			return
		}

		// Delete the queue (soft-delete)
		if err := store.Delete(ctx, name); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		// Unregister from asynq
		if err := store.UnregisterAsynqQueues(ctx, []string{name}); err != nil {
			log.Printf("Warning: failed to unregister queue from asynq: %v", err)
		}

		log.Printf("[QUEUE-CONFIG] Deleted (soft): %s", name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Queue '%s' deleted successfully", name),
		})
	}
}

func newRestoreQueueConfigHandlerFunc(store *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]

		ctx := r.Context()

		// Parse the queue name
		parent, _, hasSubQueue := queueworker.ParseQueueName(name)

		if hasSubQueue {
			// Restore a specific sub-queue
			if err := store.RestoreSubQueue(ctx, name); err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
				return
			}
			// Re-register in asynq
			if err := store.RegisterAsynqQueues(ctx, []string{name}); err != nil {
				log.Printf("Warning: failed to register queue in asynq: %v", err)
			}
		} else {
			// Restore the entire parent queue
			if err := store.EnableQueue(ctx, parent); err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
				return
			}
			// Re-register all sub-queues in asynq
			queues, err := store.ListQueues(ctx)
			if err == nil {
				var matchingQueues []string
				for _, q := range queues {
					p, _, _ := queueworker.ParseQueueName(q)
					if p == parent {
						matchingQueues = append(matchingQueues, q)
					}
				}
				if len(matchingQueues) > 0 {
					if err := store.RegisterAsynqQueues(ctx, matchingQueues); err != nil {
						log.Printf("Warning: failed to register queues in asynq: %v", err)
					}
				}
			}
		}

		log.Printf("[QUEUE-CONFIG] Restored: %s", name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Queue '%s' restored successfully", name),
		})
	}
}
