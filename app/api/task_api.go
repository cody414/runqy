package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	_client "github.com/Publikey/runqy/client"
	"github.com/Publikey/runqy/models"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/Publikey/runqy/utilities"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// GetTaskStatus godoc
//
//	@Summary		Get the information about the task in the queue and it's response if any.
//	@Description	Retrieve the body of the response or the status of the request if has not been processed already.
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			uuid	path		string	true	"The uuid of the task returned from the POST query"
//	@Success		200		{object}	models.ResponseGet
//	@Failure		400		{object}	models.APIErrorResponse
//	@Failure		404		{object}	models.APIErrorResponse
//	@Router			/queue/{uuid} [get]
func GetTaskStatus(c *gin.Context) {
	uuid := c.Param("uuid")

	redisAddr, err := models.BuildRedisConns()
	if err != nil {
		log.Fatalf("[FATAL] Redis build connection failed: %v", err)
	}

	// Look up queue from task hash: asynq:t:{task_id}
	taskKey := fmt.Sprintf("asynq:t:%s", uuid)
	queue, err := redisAddr.RDB.HGet(c, taskKey, "queue").Result()
	if err == redis.Nil {
		c.JSON(http.StatusNotFound, models.APIErrorResponse{Errors: []string{"task not found"}})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
		return
	}

	inspector := asynq.NewInspector(redisAddr.AsynqOpt)
	defer inspector.Close()

	resp, err := waitForResult(context.Background(), inspector, queue, uuid)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
		return
	}

	taskDoc := models.GetTaskInfoDoc{
		ID:            resp.ID,
		Type:          resp.Type,
		Payload:       utilities.DecodeBase64OrReturnRaw(resp.Payload),
		State:         resp.State.String(),
		Queue:         resp.Queue,
		MaxRetry:      resp.MaxRetry,
		Retried:       resp.Retried,
		LastErr:       resp.LastErr,
		LastFailedAt:  resp.LastFailedAt,
		Deadline:      resp.Deadline,
		Group:         resp.Group,
		NextProcessAt: resp.NextProcessAt,
		IsOrphaned:    resp.IsOrphaned,
		CompletedAt:   resp.CompletedAt,
		Result:        utilities.DecodeBase64OrReturnRaw(resp.Result),
	}
	response := models.ResponseGet{
		Info: taskDoc,
	}
	c.JSON(http.StatusOK, response)
}

// AddTask godoc
//
//	@Summary		Send a new task to the queuer
//	@Description	Send a generic task request with queue, timeout, and a flexible JSON payload.
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Param			task	body		models.GenericTask	true	"Task with queue, timeout, and payload"
//	@Success		200		{object}	models.GenericResponsePost
//	@Failure		400		{object}	models.APIErrorResponse
//	@Router			/queue/add [post]
//
// AddTask returns a handler that validates the incoming task `data` against
// the queue worker YAML schemas found in `qwConfigDir` before enqueuing.
func AddTask(qwConfigDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
        // Parse as raw JSON to support both flat and nested formats
        var rawBody map[string]json.RawMessage
        if err := c.ShouldBindBodyWithJSON(&rawBody); err != nil {
            c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
            return
        }

        // Extract queue (required)
        var queue string
        if queueRaw, ok := rawBody["queue"]; ok {
            if err := json.Unmarshal(queueRaw, &queue); err != nil {
                c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"invalid queue field: " + err.Error()}})
                return
            }
        }
        if queue == "" {
            c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"queue is required"}})
            return
        }

        // Normalize queue name: if no sub-queue specified, append .default
        queue = queueworker.NormalizeQueueName(queue)

        // Extract timeout (required)
        var timeout int64
        if timeoutRaw, ok := rawBody["timeout"]; ok {
            if err := json.Unmarshal(timeoutRaw, &timeout); err != nil {
                c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"invalid timeout field: " + err.Error()}})
                return
            }
        }

        // Determine the data payload
        var dataBytes json.RawMessage
        if dataRaw, ok := rawBody["data"]; ok {
            // Nested format: use "data" field directly
            dataBytes = dataRaw
        } else {
            // Flat format: collect all fields except queue and timeout
            flatData := make(map[string]json.RawMessage)
            for k, v := range rawBody {
                if k != "queue" && k != "timeout" {
                    flatData[k] = v
                }
            }
            var err error
            dataBytes, err = json.Marshal(flatData)
            if err != nil {
                c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"failed to build data payload: " + err.Error()}})
                return
            }
        }

        // Build query struct for compatibility with rest of the code
        query := models.GenericTask{
            Queue:   queue,
            Timeout: timeout,
            Data:    dataBytes,
        }

        // Prepare payloadToSend (TypedPayload by default)
        var payloadToSend interface{}

		// Validate payload against YAML schema for this queue if available
		yamls, err := queueworker.LoadAll(qwConfigDir)
		if err == nil && len(yamls) > 0 {
			var matched *queueworker.QueueYAML
			// Try to find the queue by matching runtime config names
			for _, y := range yamls {
				for baseName, qcfg := range y.Queues {
					configs := qcfg.ToQueueConfigs(baseName)
					for _, cfg := range configs {
						if cfg.Name == query.Queue {
							matched = &qcfg
							break
						}
					}
					if matched != nil {
						break
					}
				}
				if matched != nil {
					break
				}
			}

			// If not found, try trimming a trailing .default
			if matched == nil && strings.HasSuffix(query.Queue, ".default") {
				base := strings.TrimSuffix(query.Queue, ".default")
				for _, y := range yamls {
					if qcfg, ok := y.Queues[base]; ok {
						matched = &qcfg
						break
					}
				}
			}

			if matched != nil && len(matched.Input) > 0 {
				var dataMap map[string]interface{}
				if err := json.Unmarshal(query.Data, &dataMap); err != nil {
					c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{"invalid data payload: " + err.Error()}})
					return
				}

				// Validate required fields and types, and build filtered payload
				filtered := make(map[string]interface{})
				for _, field := range matched.Input {
					val, ok := dataMap[field.Name]
					if !ok {
						c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{fmt.Sprintf("%s is required", field.Name)}})
						return
					}

					if !checkAllowedType(val, field.Type) {
						c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{fmt.Sprintf("%s has invalid type", field.Name)}})
						return
					}

					// If the field expects an int but JSON gave float64, convert to int64
					if contains(field.Type, "int") {
						if f, ok := val.(float64); ok {
							// use int64 for integers
							filtered[field.Name] = int64(f)
							continue
						}
					}
					filtered[field.Name] = val
				}

				// Build typed payload (flat map) to send
				payloadToSend = models.TypedPayload(filtered)
			}
			// if matched == nil, fallthrough and parse into TypedPayload below
		}

		// If payloadToSend is still nil, create a TypedPayload from the raw data
		if payloadToSend == nil {
			var dataMap map[string]interface{}
			if err := json.Unmarshal(query.Data, &dataMap); err != nil {
				// If unmarshalling fails, fallback to sending raw bytes
				payloadToSend = json.RawMessage(query.Data)
			} else {
				payloadToSend = models.TypedPayload(dataMap)
			}
		}

		asynqClient := c.Keys["client"].(*asynq.Client)
		rdb := c.Keys["rdb"].(*redis.Client)

		info, err := _client.EnqueueGenericTask(asynqClient, rdb, query.Queue, query.Timeout, payloadToSend)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}
		// Manual mapping from *asynq.TaskInfo to TaskInfoDoc
		taskDoc := models.AddTaskInfoDoc{
			ID:            info.ID,
			Type:          info.Type,
			Payload:       info.Payload,
			State:         info.State.String(),
			Queue:         info.Queue,
			MaxRetry:      info.MaxRetry,
			Retried:       info.Retried,
			LastErr:       info.LastErr,
			LastFailedAt:  info.LastFailedAt,
			Deadline:      info.Deadline,
			Group:         info.Group,
			NextProcessAt: info.NextProcessAt,
			IsOrphaned:    info.IsOrphaned,
			CompletedAt:   info.CompletedAt,
			Result:        info.Result,
		}
		response := models.GenericResponsePost{
			Info: taskDoc,
			Data: query.Data,
		}
		c.JSON(http.StatusOK, response)
	}

}

func waitForResult(ctx context.Context, i *asynq.Inspector, queue, taskID string) (*asynq.TaskInfo, error) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			taskInfo, err := i.GetTaskInfo(queue, taskID)
			if err != nil {
				return nil, err
			}
			if taskInfo.LastErr != "" {
				return taskInfo, nil
			}
			return taskInfo, nil
		case <-ctx.Done():
			return nil, fmt.Errorf("context closed")
		}
	}
}

// checkAllowedType checks if the provided value matches any of the allowed types
func checkAllowedType(v interface{}, allowed []string) bool {
	switch val := v.(type) {
	case string:
		return contains(allowed, "string")
	case bool:
		return contains(allowed, "bool")
	case float64:
		// JSON numbers are float64; determine if integer-valued
		if math.Trunc(val) == val {
			return contains(allowed, "int") || contains(allowed, "float")
		}
		return contains(allowed, "float")
	case []interface{}:
		return contains(allowed, "array")
	case map[string]interface{}:
		return contains(allowed, "object")
	default:
		return false
	}
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}
