package client

import (
	"context"
	"encoding/json"
	"time"

	queueworker "github.com/Publikey/runqy/queues"
	t "github.com/Publikey/runqy/tasks"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// EnqueueGenericTask enqueues a task with a raw JSON payload to the specified queue
// EnqueueGenericTask enqueues a task. Payload may be json.RawMessage or any typed object
// which will be marshaled to JSON here.
func EnqueueGenericTask(client *asynq.Client, rdb *redis.Client, queue string, timeout int64, payload interface{}) (*asynq.TaskInfo, error) {
	// Normalize queue name: if no sub-queue specified, append .default
	queue = queueworker.NormalizeQueueName(queue)

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

	// Store queue name in task hash for reverse lookup (GET /queue/{task_id})
	rdb.HSet(context.Background(), "asynq:t:"+taskInfo.ID, "queue", queue)

	// Register queue for asynqmon visibility
	rdb.SAdd(context.Background(), "asynq:queues", queue)

	return taskInfo, err
}
