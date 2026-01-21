package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Publikey/runqy/config"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/gin-gonic/gin"
)

// HandshakeRequest is the worker handshake request body
type HandshakeRequest struct {
	Queue    string `json:"queue"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Version  string `json:"version"`
}

// WorkerRedisConfig contains Redis connection info for the worker
type WorkerRedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	TLS      bool   `json:"use_tls"`
	DB       int    `json:"db"`
}

// WorkerQueueConfig contains queue info for the worker
type WorkerQueueConfig struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

// SubQueueConfig contains sub-queue info for the worker
type SubQueueConfig struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

// WorkerDeploymentConfig contains deployment info for the worker
type WorkerDeploymentConfig struct {
	GitURL             string            `json:"git_url"`
	Branch             string            `json:"branch"`
	CodePath           string            `json:"code_path,omitempty"`
	StartupCmd         string            `json:"startup_cmd"`
	Mode               string            `json:"mode,omitempty"`
	EnvVars            map[string]string `json:"env_vars"`
	StartupTimeoutSecs int               `json:"startup_timeout_secs"`
}

// WorkerConfigResponse is the full response for worker registration
type WorkerConfigResponse struct {
	Redis      WorkerRedisConfig      `json:"redis"`
	Queue      WorkerQueueConfig      `json:"queue"`
	SubQueues  []SubQueueConfig       `json:"sub_queues"`
	Deployment WorkerDeploymentConfig `json:"deployment"`
}

// HandshakeErrorResponse is returned on handshake errors
type HandshakeErrorResponse struct {
	Error string `json:"error"`
}

// resolveEnvVars resolves environment variable references in env_vars map.
// Values prefixed with "env://" are replaced with the actual environment variable value.
// For example, "env://SECRET_KEY" becomes the value of os.Getenv("SECRET_KEY").
func resolveEnvVars(envVars map[string]string) map[string]string {
	if envVars == nil {
		return nil
	}

	resolved := make(map[string]string, len(envVars))
	for key, value := range envVars {
		if envName, found := strings.CutPrefix(value, "env://"); found {
			resolved[key] = os.Getenv(envName)
		} else {
			resolved[key] = value
		}
	}
	return resolved
}

// QueuesListResponse is the response for listing all queues
type QueuesListResponse struct {
	Queues []queueworker.QueueSummary `json:"queues"`
	Count  int                        `json:"count"`
}

// ReloadResponse is the response for reloading configurations
type ReloadResponse struct {
	Reloaded  []string `json:"reloaded"`
	Errors    []string `json:"errors"`
	Timestamp int64    `json:"timestamp"`
}

// WorkerHandshake handles worker registration and config retrieval
// Workers are trusted - they know their queue name and get the config directly
// @Summary Worker handshake
// @Description Register worker and retrieve queue configuration
// @Tags workers
// @Accept json
// @Produce json
// @Param request body HandshakeRequest true "Handshake request"
// @Success 200 {object} WorkerConfigResponse
// @Failure 400 {object} HandshakeErrorResponse
// @Failure 404 {object} HandshakeErrorResponse
// @Router /workers/handshake [post]
// @Router /worker/register [post]
func WorkerHandshake(store *queueworker.Store, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req HandshakeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, HandshakeErrorResponse{
				Error: "invalid request: " + err.Error(),
			})
			return
		}

		if req.Queue == "" {
			c.JSON(http.StatusBadRequest, HandshakeErrorResponse{
				Error: "queue name is required",
			})
			return
		}

		queueName := req.Queue

		// Get all sub-queues for this queue (matching "queueName_*" pattern)
		subQueueConfigs, err := store.ListByPrefix(c.Request.Context(), queueName+"_")
		if err != nil {
			c.JSON(http.StatusInternalServerError, HandshakeErrorResponse{
				Error: "failed to retrieve sub-queue configurations",
			})
			return
		}

		// Build sub-queues list
		var subQueues []SubQueueConfig
		var queueCfg *queueworker.QueueConfig

		if len(subQueueConfigs) > 0 {
			// Use explicitly defined sub-queues
			for _, sq := range subQueueConfigs {
				subQueues = append(subQueues, SubQueueConfig{
					Name:     sq.Name,
					Priority: sq.Priority,
				})
			}
			// Use the first sub-queue config for deployment info
			queueCfg = subQueueConfigs[0]
		} else {
			// No sub-queues found, try to get the exact queue name
			queueCfg, err = store.Get(c.Request.Context(), queueName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, HandshakeErrorResponse{
					Error: "failed to retrieve configuration",
				})
				return
			}

			if queueCfg == nil {
				c.JSON(http.StatusNotFound, HandshakeErrorResponse{
					Error: fmt.Sprintf("queue '%s' not found", queueName),
				})
				return
			}

			// Default behavior: auto-create single sub-queue named "{queue}_default"
			subQueues = []SubQueueConfig{{
				Name:     queueName + "_default",
				Priority: queueCfg.Priority,
			}}
		}

		// Build deployment config with resolved environment variables
		var deployment WorkerDeploymentConfig
		if queueCfg.Deployment != nil {
			deployment = WorkerDeploymentConfig{
				GitURL:             queueCfg.Deployment.GitURL,
				Branch:             queueCfg.Deployment.Branch,
				CodePath:           queueCfg.Deployment.CodePath,
				StartupCmd:         queueCfg.Deployment.StartupCmd,
				Mode:               queueCfg.Deployment.Mode,
				EnvVars:            resolveEnvVars(queueCfg.Deployment.EnvVars),
				StartupTimeoutSecs: queueCfg.Deployment.StartupTimeoutSecs,
			}
		}

		// Return the full worker config response
		c.JSON(http.StatusOK, WorkerConfigResponse{
			Redis: WorkerRedisConfig{
				Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
				Password: cfg.RedisPassword,
				TLS:      cfg.RedisTLS,
			},
			Queue: WorkerQueueConfig{
				Name:     queueName,
				Priority: queueCfg.Priority,
			},
			SubQueues:  subQueues,
			Deployment: deployment,
		})
	}
}

// GetQueueConfig returns configuration for a specific queue
// @Summary Get queue configuration
// @Description Retrieve queue configuration
// @Tags workers
// @Produce json
// @Param queue_name path string true "Queue name"
// @Success 200 {object} queueworker.QueueConfig
// @Failure 404 {object} map[string]string
// @Router /workers/config/{queue_name} [get]
func GetQueueConfig(store *queueworker.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		queueName := c.Param("queue_name")

		cfg, err := store.Get(c.Request.Context(), queueName)
		if err != nil || cfg == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "queue not found"})
			return
		}

		c.JSON(http.StatusOK, cfg)
	}
}

// ListQueueConfigs returns all registered queues
// @Summary List all queue configurations
// @Description List all registered queue configurations
// @Tags workers
// @Produce json
// @Success 200 {object} QueuesListResponse
// @Failure 500 {object} map[string]string
// @Router /workers/queues [get]
func ListQueueConfigs(store *queueworker.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		queues, err := store.ListQueues(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Get summary for each queue
		summaries := make([]queueworker.QueueSummary, 0, len(queues))
		for _, q := range queues {
			cfg, err := store.Get(c.Request.Context(), q)
			if err != nil || cfg == nil {
				continue
			}
			summaries = append(summaries, queueworker.QueueSummary{
				Name:     cfg.Name,
				Priority: cfg.Priority,
				Provider: cfg.Provider,
			})
		}

		c.JSON(http.StatusOK, QueuesListResponse{
			Queues: summaries,
			Count:  len(summaries),
		})
	}
}

// ReloadQueueConfigs reloads YAML configurations (admin only - uses global API key)
// @Summary Reload queue configurations from YAML files
// @Description Reload all queue configurations from YAML files (admin only)
// @Tags workers
// @Produce json
// @Success 200 {object} ReloadResponse
// @Router /workers/reload [post]
func ReloadQueueConfigs(store *queueworker.Store, configDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reloaded, errors := reloadFromYAML(c.Request.Context(), store, configDir)
		c.JSON(http.StatusOK, ReloadResponse{
			Reloaded:  reloaded,
			Errors:    errors,
			Timestamp: time.Now().Unix(),
		})
	}
}

// ReloadFromYAMLContext is an exported version for use by file watcher
func ReloadFromYAMLContext(ctx context.Context, store *queueworker.Store, dir string) ([]string, []string) {
	return reloadFromYAML(ctx, store, dir)
}

// reloadFromYAML loads configurations from YAML files and saves to DB
func reloadFromYAML(ctx context.Context, store *queueworker.Store, dir string) ([]string, []string) {
	var reloaded []string
	var errors []string

	yamls, err := queueworker.LoadAll(dir)
	if err != nil {
		errors = append(errors, "failed to load YAML files: "+err.Error())
		return reloaded, errors
	}

	for _, y := range yamls {
		for queueName, queueCfg := range y.Queues {
			// Convert to runtime configs (handles sub_queues)
			configs := queueCfg.ToQueueConfigs(queueName)

			for _, cfg := range configs {
				// Save to DB
				if err := store.Save(ctx, cfg); err != nil {
					errors = append(errors, cfg.Name+": failed to save: "+err.Error())
					continue
				}

				reloaded = append(reloaded, cfg.Name)
				log.Printf("[QUEUEWORKER] Loaded: %s (provider=%s, priority=%d)", cfg.Name, cfg.Provider, cfg.Priority)
			}
		}
	}

	// Register queues in asynq so they appear in asynqmon
	if len(reloaded) > 0 {
		if err := store.RegisterAsynqQueues(ctx, reloaded); err != nil {
			log.Printf("[QUEUEWORKER] Warning: failed to register asynq queues: %v", err)
		} else {
			log.Printf("[QUEUEWORKER] Registered queues in asynq: %v", reloaded)
		}
	}

	return reloaded, errors
}

// LoadQueueWorkersAtStartup is called during application startup to load YAML configs
// Returns the list of provider types found in the configs for provider registration
func LoadQueueWorkersAtStartup(store *queueworker.Store, configDir string) ([]string, error) {
	ctx := context.Background()
	reloaded, providerTypes, errors := reloadFromYAMLWithProviders(ctx, store, configDir)

	if len(errors) > 0 {
		for _, e := range errors {
			log.Printf("[QUEUEWORKER] Warning: %s", e)
		}
	}

	if len(reloaded) > 0 {
		log.Printf("[QUEUEWORKER] Loaded %d queue configurations: %v", len(reloaded), reloaded)

		// Register queues in asynq so they appear in asynqmon
		if err := store.RegisterAsynqQueues(ctx, reloaded); err != nil {
			log.Printf("[QUEUEWORKER] Warning: failed to register asynq queues: %v", err)
		} else {
			log.Printf("[QUEUEWORKER] Registered queues in asynq: %v", reloaded)
		}
	} else {
		log.Printf("[QUEUEWORKER] No queue configurations loaded from %s", configDir)
	}

	return providerTypes, nil
}

// reloadFromYAMLWithProviders loads configs and returns provider types
func reloadFromYAMLWithProviders(ctx context.Context, store *queueworker.Store, dir string) ([]string, []string, []string) {
	var reloaded []string
	var providerTypes []string
	var errors []string

	yamls, err := queueworker.LoadAll(dir)
	if err != nil {
		errors = append(errors, "failed to load YAML files: "+err.Error())
		return reloaded, providerTypes, errors
	}

	for _, y := range yamls {
		for queueName, queueCfg := range y.Queues {
			// Collect provider type
			if queueCfg.Provider != "" && queueCfg.Provider != "worker" {
				providerTypes = append(providerTypes, queueCfg.Provider)
			}

			// Convert to runtime configs (handles sub_queues)
			configs := queueCfg.ToQueueConfigs(queueName)

			for _, cfg := range configs {
				// Save to DB
				if err := store.Save(ctx, cfg); err != nil {
					errors = append(errors, cfg.Name+": failed to save: "+err.Error())
					continue
				}

				reloaded = append(reloaded, cfg.Name)
				log.Printf("[QUEUEWORKER] Loaded: %s (provider=%s, priority=%d)", cfg.Name, cfg.Provider, cfg.Priority)
			}
		}
	}

	return reloaded, providerTypes, errors
}
