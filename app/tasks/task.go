package tasks

import (
	"encoding/json"
	"fmt"

	models "github.com/Publikey/runqy/models"
	"github.com/hibiken/asynq"
)

// NewPredictTask creates a new prediction task
func NewPredictTask(queue string, payload models.Predict) (*asynq.Task, error) {

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return asynq.NewTask(queue, payloadBytes), nil
}

// NewGenericTask creates a new task with a raw JSON payload
func NewGenericTask(queue string, payload json.RawMessage) (*asynq.Task, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("payload cannot be empty")
	}
	return asynq.NewTask(queue, payload), nil
}
