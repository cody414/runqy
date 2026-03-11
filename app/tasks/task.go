package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/Publikey/runqy/third_party/asynq"
)

// NewGenericTask creates a new task with a raw JSON payload
func NewGenericTask(queue string, payload json.RawMessage) (*asynq.Task, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("payload cannot be empty")
	}
	return asynq.NewTask(queue, payload), nil
}
