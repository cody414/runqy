package monitoring

import (
	"encoding/json"
	"net/http"
)

// errorsResponse matches the main API error format: {"errors":["..."]}
type errorsResponse struct {
	Errors []string `json:"errors"`
}

// writeJSONError writes a JSON error response in the standard format: {"errors":["message"]}.
// This ensures consistency with the main API error format introduced in the bulletproof branch.
func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorsResponse{Errors: []string{msg}})
}
