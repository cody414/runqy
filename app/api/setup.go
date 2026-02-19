package api

import (
	"sync"

	"github.com/Publikey/runqy/config"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/Publikey/runqy/vaults"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

// Global vault store for use by worker handshake (protected by mutex)
var (
	globalVaultStore   *vaults.Store
	globalVaultStoreMu sync.RWMutex
)

// GetVaultStore returns the global vault store (for use by worker handshake)
func GetVaultStore() *vaults.Store {
	globalVaultStoreMu.RLock()
	defer globalVaultStoreMu.RUnlock()
	return globalVaultStore
}

func SetupAPI(r *gin.Engine, qwStore *queueworker.Store, qwConfigDir string, cfg *config.Config, redisOpt asynq.RedisClientOpt) {
	apiKey := cfg.APIKey

	// Setup CLI API endpoints (queue/task management)
	inspector := asynq.NewInspector(redisOpt)
	SetupCLIAPI(r, inspector, qwStore, apiKey)
	// Queue API - all routes require API key
	router_predict := r.Group("queue")
	router_predict.Use(Authorize(apiKey))
	router_predict.GET("/:uuid", GetTaskStatus)
	router_predict.POST("/add", AddTask(qwConfigDir, qwStore))
	router_predict.POST("/add-batch", AddTaskBatch(qwConfigDir, qwStore))

	// Worker registration endpoint
	router_worker := r.Group("worker")
	router_worker.Use(Authorize(apiKey))
	router_worker.POST("/register", WorkerHandshake(qwStore, cfg))

	// Workers API - all routes require API key
	router_workers := r.Group("workers")
	router_workers.Use(Authorize(apiKey))
	router_workers.GET("/config/:queue_name", GetQueueConfig(qwStore))
	router_workers.GET("", ListWorkers)
	router_workers.GET("/:worker_id", GetWorker)
	router_workers.GET("/queues", ListQueueConfigs(qwStore))
	router_workers.POST("/queues", CreateQueueConfig(qwStore))
	router_workers.DELETE("/queues/:queue_name", DeleteQueueConfig(qwStore))
	router_workers.POST("/queues/:queue_name/restore", RestoreQueueConfig(qwStore))
	router_workers.POST("/reload", ReloadQueueConfigs(qwStore, qwConfigDir))
}

// SetupVaultsAPI sets up the vaults API routes
func SetupVaultsAPI(r *gin.Engine, vaultStore *vaults.Store, apiKey string) {
	globalVaultStoreMu.Lock()
	globalVaultStore = vaultStore
	globalVaultStoreMu.Unlock()

	// Vaults API - all routes require API key authentication
	router_vaults := r.Group("/api/vaults")
	router_vaults.Use(Authorize(apiKey))

	// Vault CRUD
	router_vaults.GET("", ListVaults(vaultStore))
	router_vaults.POST("", CreateVault(vaultStore))
	router_vaults.GET("/:name", GetVault(vaultStore))
	router_vaults.DELETE("/:name", DeleteVault(vaultStore))

	// Vault entry CRUD
	router_vaults.POST("/:name/entries", SetEntry(vaultStore))
	router_vaults.GET("/:name/entries", ListEntries(vaultStore))
	router_vaults.DELETE("/:name/entries/:key", DeleteEntry(vaultStore))
}
