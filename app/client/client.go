package client

import (
	"context"
	"encoding/json"
	"log"
	"time"

	m "github.com/Publikey/runqy/models"
	t "github.com/Publikey/runqy/tasks"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// EnqueuePredictTask enqueues a task to the specified queue
func EnqueuePredictTask(client *asynq.Client, rdb *redis.Client, queue string, timeout int64, payload m.Predict) (*asynq.TaskInfo, error) {
	payload.EnqueuedAt = time.Now().Unix()
	payload.Status = "pending"

	task, err := t.NewPredictTask(queue, payload)
	if err != nil {
		log.Fatalf("Failed to create %d prediction task: %v", payload.PredictId, err)
	}

	opts := []asynq.Option{
		asynq.Timeout(time.Duration(timeout) * time.Second),
		asynq.Queue(queue),
		asynq.MaxRetry(3),
		asynq.Retention(24 * time.Hour),
	}

	taskInfo, err := client.Enqueue(task, opts...)
	if err != nil {
		return nil, err
	}

	// Register queue for asynqmon visibility
	rdb.SAdd(context.Background(), "asynq:queues", queue)

	return taskInfo, err
}

// EnqueueGenericTask enqueues a task with a raw JSON payload to the specified queue
// EnqueueGenericTask enqueues a task. Payload may be json.RawMessage or any typed object
// which will be marshaled to JSON here.
func EnqueueGenericTask(client *asynq.Client, rdb *redis.Client, queue string, timeout int64, payload interface{}) (*asynq.TaskInfo, error) {
	var payloadBytes []byte
	switch p := payload.(type) {
	case json.RawMessage:
		payloadBytes = p
	default:
		b, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		payloadBytes = b
	}

	task, err := t.NewGenericTask(queue, payloadBytes)
	if err != nil {
		return nil, err
	}

	opts := []asynq.Option{
		asynq.Timeout(time.Duration(timeout) * time.Second),
		asynq.Queue(queue),
		asynq.MaxRetry(3),
		asynq.Retention(24 * time.Hour),
	}

	taskInfo, err := client.Enqueue(task, opts...)
	if err != nil {
		return nil, err
	}

	// Register queue for asynqmon visibility
	rdb.SAdd(context.Background(), "asynq:queues", queue)

	return taskInfo, err
}
