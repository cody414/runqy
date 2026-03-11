package monitoring

import (
	"encoding/json"
	"errors"
	"net/http"

	queueworker "github.com/Publikey/runqy/queues"
	"github.com/gorilla/mux"

	"github.com/Publikey/runqy/third_party/asynq"
)

// ****************************************************************************
// This file defines:
//   - http.Handler(s) for queue related endpoints
// ****************************************************************************

func newListQueuesHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qnames, err := inspector.Queues()
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		snapshots := make([]*queueStateSnapshot, len(qnames))
		for i, qname := range qnames {
			qinfo, err := inspector.GetQueueInfo(qname)
			if err != nil {
				writeJSONError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			snapshots[i] = toQueueStateSnapshot(qinfo)
		}
		payload := map[string]interface{}{"queues": snapshots}
		json.NewEncoder(w).Encode(payload)
	}
}

func newGetQueueHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		qname := vars["qname"]

		payload := make(map[string]interface{})
		qinfo, err := inspector.GetQueueInfo(qname)
		if err != nil {
			// TODO: Check for queue not found error.
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payload["current"] = toQueueStateSnapshot(qinfo)

		// TODO: make this n a variable
		data, err := inspector.History(qname, 10)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var dailyStats []*dailyStats
		for _, s := range data {
			dailyStats = append(dailyStats, toDailyStats(s))
		}
		payload["history"] = dailyStats
		json.NewEncoder(w).Encode(payload)
	}
}

func newDeleteQueueHandlerFunc(inspector *asynq.Inspector, queueStore *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		qname := vars["qname"]
		// Check for force query parameter
		force := r.URL.Query().Get("force") == "true"

		// Delete from Redis (asynq queue data)
		if err := inspector.DeleteQueue(qname, force); err != nil {
			// Ignore "queue not found" in Redis - it might only exist in DB
			if !errors.Is(err, asynq.ErrQueueNotFound) {
				if errors.Is(err, asynq.ErrQueueNotEmpty) {
					writeJSONError(w, "queue is not empty - use force=true to delete anyway", http.StatusBadRequest)
					return
				}
				writeJSONError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Delete from database (queue configuration)
		if queueStore != nil {
			if err := queueStore.Delete(r.Context(), qname); err != nil {
				// Log but don't fail - Redis deletion already succeeded
				// The queue config might not exist in DB
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func newPauseQueueHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		qname := vars["qname"]
		if err := inspector.PauseQueue(qname); err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func newResumeQueueHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		qname := vars["qname"]
		if err := inspector.UnpauseQueue(qname); err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func newRestoreQueueHandlerFunc(queueStore *queueworker.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if queueStore == nil {
			writeJSONError(w, "queue store not configured", http.StatusInternalServerError)
			return
		}

		vars := mux.Vars(r)
		qname := vars["qname"]

		ctx := r.Context()

		// Check if this is a full queue name (with subqueue) or just a parent queue
		parent, _, hasSubQueue := queueworker.ParseQueueName(qname)

		if hasSubQueue {
			// Restore a specific sub-queue
			if err := queueStore.RestoreSubQueue(ctx, qname); err != nil {
				writeJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Re-register in asynq
			if err := queueStore.RegisterAsynqQueues(ctx, []string{qname}); err != nil {
				// Log but don't fail
			}
		} else {
			// Restore the entire parent queue and all its sub-queues
			if err := queueStore.EnableQueue(ctx, parent); err != nil {
				writeJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Re-register all sub-queues in asynq
			queues, err := queueStore.ListQueues(ctx)
			if err == nil {
				var matchingQueues []string
				for _, q := range queues {
					p, _, _ := queueworker.ParseQueueName(q)
					if p == parent {
						matchingQueues = append(matchingQueues, q)
					}
				}
				if len(matchingQueues) > 0 {
					queueStore.RegisterAsynqQueues(ctx, matchingQueues)
				}
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type listQueueStatsResponse struct {
	Stats map[string][]*dailyStats `json:"stats"`
}

func newListQueueStatsHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qnames, err := inspector.Queues()
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := listQueueStatsResponse{Stats: make(map[string][]*dailyStats)}
		const numdays = 90 // Get stats for the last 90 days.
		for _, qname := range qnames {
			stats, err := inspector.History(qname, numdays)
			if err != nil {
				writeJSONError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			resp.Stats[qname] = toDailyStatsList(stats)
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			writeJSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
