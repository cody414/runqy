package api

import (
	"github.com/Publikey/runqy/config"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/gin-gonic/gin"
)

func SetupAPI(r *gin.Engine, qwStore *queueworker.Store, qwConfigDir string, cfg *config.Config) {
	// api GET queue
	router_predict := r.Group("queue")
	router_predict.GET("/:uuid/:priority", GetPredictStatus)
	// api POST queue
	router_predict.Use(Authorize())
	router_predict.POST("/add", NewPredict(qwConfigDir))

	// Workers API - reads from asynq:workers keys (matching runqy-worker)
	router_workers := r.Group("workers")
	router_workers.GET("", ListWorkers)
	router_workers.GET("/:worker_id", GetWorker)

	// Worker registration endpoint - workers are trusted, no API key validation
	router_worker := r.Group("worker")
	router_worker.POST("/register", WorkerHandshake(qwStore, cfg))

	// Public queue config endpoint
	router_workers.GET("/config/:queue_name", GetQueueConfig(qwStore))

	// Admin-only endpoints - require global ASYNQ_API_KEY
	router_workers.Use(Authorize())
	router_workers.GET("/queues", ListQueueConfigs(qwStore))
	router_workers.POST("/reload", ReloadQueueConfigs(qwStore, qwConfigDir))
}
