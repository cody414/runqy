package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Publikey/runqy/models"
	queueworker "github.com/Publikey/runqy/queues"
	t "github.com/Publikey/runqy/tasks"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// BatchTaskRequest represents a batch of tasks to enqueue
type BatchTaskRequest struct {
	Queue   string            `json:"queue" binding:"required"`
	Timeout int64             `json:"timeout"` // Default timeout for all jobs
	Jobs    []json.RawMessage `json:"jobs" binding:"required"`
}

// BatchTaskResponse contains the results of batch enqueue
type BatchTaskResponse struct {
	Enqueued int      `json:"enqueued"`
	Failed   int      `json:"failed"`
	TaskIDs  []string `json:"task_ids"`
	Errors   []string `json:"errors,omitempty"`
}

// AddTaskBatch godoc
//
//	@Summary		Send multiple tasks to the queue in a single request
//	@Description	Batch enqueue for high-throughput job submission. Uses Redis pipelining for optimal performance.
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			batch	body		BatchTaskRequest	true	"Batch of tasks with queue and jobs array"
//	@Success		200		{object}	BatchTaskResponse
//	@Failure		400		{object}	models.APIErrorResponse
//	@Router			/queue/add-batch [post]
//
// AddTaskBatch returns a handler that enqueues multiple tasks using Redis pipelining
func AddTaskBatch(qwConfigDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		if len(req.Jobs) == 0 {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"jobs array cannot be empty"}})
			return
		}

		// Normalize queue name
		queue := queueworker.NormalizeQueueName(req.Queue)

		// Get timeout (default 30s)
		timeout := req.Timeout
		if timeout == 0 {
			timeout = 30
		}

		// Get asynq client and redis client
		asynqClient := c.Keys["client"].(*asynq.Client)
		rdb := c.Keys["rdb"].(*redis.Client)

		// Prepare batch
		response := BatchTaskResponse{
			TaskIDs: make([]string, 0, len(req.Jobs)),
			Errors:  make([]string, 0),
		}

		// Use Redis pipeline for batch operations
		ctx := context.Background()
		pipe := rdb.Pipeline()

		// Collect task infos for the response
		type taskEntry struct {
			taskID string
			queue  string
		}
		var taskEntries []taskEntry

		for i, jobData := range req.Jobs {
			// Create task
			task, err := t.NewGenericTask(queue, jobData)
			if err != nil {
				response.Failed++
				response.Errors = append(response.Errors, err.Error())
				continue
			}

			// Enqueue task
			opts := []asynq.Option{
				asynq.Timeout(time.Duration(timeout) * time.Second),
				asynq.Queue(queue),
				asynq.MaxRetry(3),
				asynq.Retention(24 * time.Hour),
			}

			taskInfo, err := asynqClient.Enqueue(task, opts...)
			if err != nil {
				response.Failed++
				response.Errors = append(response.Errors, err.Error())
				continue
			}

			// Queue metadata update in pipeline (non-blocking)
			pipe.HSet(ctx, "asynq:t:"+taskInfo.ID, "queue", queue)
			taskEntries = append(taskEntries, taskEntry{taskID: taskInfo.ID, queue: queue})

			response.TaskIDs = append(response.TaskIDs, taskInfo.ID)
			response.Enqueued++

			// Flush pipeline every 100 jobs to prevent memory buildup
			if (i+1)%100 == 0 {
				pipe.Exec(ctx)
				pipe = rdb.Pipeline()
			}
		}

		// Execute remaining pipeline commands
		if len(taskEntries) > 0 {
			pipe.SAdd(ctx, "asynq:queues", queue)
			pipe.Exec(ctx)
		}

		c.JSON(http.StatusOK, response)
	}
}

// AddTaskBatchDirect is an optimized version that bypasses asynq client
// and writes directly to Redis using pipelining for maximum throughput.
// Use this when you need absolute maximum performance and can accept
// slightly reduced feature set (no per-job options).
func AddTaskBatchDirect(qwConfigDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		if len(req.Jobs) == 0 {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"jobs array cannot be empty"}})
			return
		}

		// Normalize queue name
		queue := queueworker.NormalizeQueueName(req.Queue)

		// Get redis client
		rdb := c.Keys["rdb"].(*redis.Client)
		ctx := context.Background()

		response := BatchTaskResponse{
			TaskIDs: make([]string, 0, len(req.Jobs)),
			Errors:  make([]string, 0),
		}

		// Prepare all tasks first
		var taskMsgs []interface{}
		pendingKey := "asynq:{" + queue + "}:pending"

		pipe := rdb.Pipeline()

		for _, jobData := range req.Jobs {
			task, err := t.NewGenericTask(queue, jobData)
			if err != nil {
				response.Failed++
				response.Errors = append(response.Errors, err.Error())
				continue
			}

			// Serialize task for Redis
			taskMsg, err := serializeAsynqTask(task, queue)
			if err != nil {
				response.Failed++
				response.Errors = append(response.Errors, err.Error())
				continue
			}

			taskMsgs = append(taskMsgs, taskMsg)
			response.TaskIDs = append(response.TaskIDs, task.ResultWriter().TaskID())
			response.Enqueued++

			// Store metadata
			pipe.HSet(ctx, "asynq:t:"+task.ResultWriter().TaskID(), "queue", queue)
		}

		// Bulk LPUSH all tasks
		if len(taskMsgs) > 0 {
			rdb.LPush(ctx, pendingKey, taskMsgs...)
			pipe.SAdd(ctx, "asynq:queues", queue)
			pipe.Exec(ctx)
		}

		c.JSON(http.StatusOK, response)
	}
}

// serializeAsynqTask converts an asynq.Task to the wire format used by asynq
// This is a simplified version - for full compatibility, use asynq.Client
func serializeAsynqTask(task *asynq.Task, queue string) ([]byte, error) {
	// asynq uses a specific protobuf/msgpack format
	// For now, we use JSON as a placeholder
	// TODO: Use proper asynq internal format
	msg := map[string]interface{}{
		"type":     task.Type(),
		"payload":  task.Payload(),
		"queue":    queue,
		"retry":    3,
		"timeout":  30,
		"deadline": time.Now().Add(time.Hour).Unix(),
	}
	return json.Marshal(msg)
}
