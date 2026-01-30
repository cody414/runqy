package api

import (
	"github.com/Publikey/runqy/config"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/Publikey/runqy/vaults"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

// Global vault store for use by worker handshake
var globalVaultStore *vaults.Store

// GetVaultStore returns the global vault store (for use by worker handshake)
func GetVaultStore() *vaults.Store {
	return globalVaultStore
}

func SetupAPI(r *gin.Engine, qwStore *queueworker.Store, qwConfigDir string, cfg *config.Config, redisOpt asynq.RedisClientOpt) {
	// Setup CLI API endpoints (queue/task management)
	inspector := asynq.NewInspector(redisOpt)
	SetupCLIAPI(r, inspector, qwStore)
	// api GET queue
	router_predict := r.Group("queue")
	router_predict.GET("/:uuid", GetTaskStatus)
	// api POST queue
	router_predict.Use(Authorize())
	router_predict.POST("/add", AddTask(qwConfigDir))

	// Workers API - reads from asynq:workers keys (matching runqy-worker)
	router_workers := r.Group("workers")
	router_workers.GET("", ListWorkers)
	router_workers.GET("/:worker_id", GetWorker)

	// Worker registration endpoint - workers are trusted, no API key validation
	router_worker := r.Group("worker")
	router_worker.POST("/register", WorkerHandshake(qwStore, cfg))

	// Public queue config endpoint
	router_workers.GET("/config/:queue_name", GetQueueConfig(qwStore))

	// Admin-only endpoints - require global RUNQY_API_KEY
	router_workers.Use(Authorize())
	router_workers.GET("/queues", ListQueueConfigs(qwStore))
	router_workers.POST("/queues", CreateQueueConfig(qwStore))
	router_workers.DELETE("/queues/:queue_name", DeleteQueueConfig(qwStore))
	router_workers.POST("/reload", ReloadQueueConfigs(qwStore, qwConfigDir))
}

// SetupVaultsAPI sets up the vaults API routes
func SetupVaultsAPI(r *gin.Engine, vaultStore *vaults.Store) {
	globalVaultStore = vaultStore

	// Vaults API - all routes require API key authentication
	router_vaults := r.Group("/api/vaults")
	router_vaults.Use(Authorize())

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
