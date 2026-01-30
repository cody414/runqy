package monitoring

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Publikey/runqy/vaults"
	"github.com/gorilla/mux"
)

// VaultsListResponse is the response for listing vaults
type VaultsListResponse struct {
	Vaults []vaults.VaultSummary `json:"vaults"`
	Count  int                   `json:"count"`
}

// CreateVaultRequest is the request body for creating a vault
type CreateVaultRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SetEntryRequest is the request body for setting a vault entry
type SetEntryRequest struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret *bool  `json:"is_secret"`
}

func newListVaultsHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		summaries, err := store.ListVaults(r.Context())
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		resp := VaultsListResponse{
			Vaults: summaries,
			Count:  len(summaries),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func newCreateVaultHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		var req CreateVaultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}

		// Check if vault already exists
		exists, err := store.VaultExists(r.Context(), req.Name)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if exists {
			http.Error(w, `{"error":"vault already exists"}`, http.StatusConflict)
			return
		}

		vault, err := store.CreateVault(r.Context(), req.Name, req.Description)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[VAULTS] Created vault: %s", vault.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "vault created successfully",
			"vault": map[string]string{
				"name":        vault.Name,
				"description": vault.Description,
			},
		})
	}
}

func newGetVaultHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		name := vars["name"]

		detail, err := store.GetVaultDetail(r.Context(), name)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if detail == nil {
			http.Error(w, `{"error":"vault not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(detail)
	}
}

func newDeleteVaultHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		name := vars["name"]

		// Check if vault exists
		exists, err := store.VaultExists(r.Context(), name)
		if err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, `{"error":"vault not found"}`, http.StatusNotFound)
			return
		}

		if err := store.DeleteVault(r.Context(), name); err != nil {
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[VAULTS] Deleted vault: %s", name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "vault deleted successfully"})
	}
}

func newSetEntryHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		vaultName := vars["name"]

		var req SetEntryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		if req.Key == "" {
			http.Error(w, `{"error":"key is required"}`, http.StatusBadRequest)
			return
		}
		if req.Value == "" {
			http.Error(w, `{"error":"value is required"}`, http.StatusBadRequest)
			return
		}

		// Default isSecret to true
		isSecret := true
		if req.IsSecret != nil {
			isSecret = *req.IsSecret
		}

		if err := store.SetEntry(r.Context(), vaultName, req.Key, req.Value, isSecret); err != nil {
			if err.Error() == "vault '"+vaultName+"' not found" {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[VAULTS] Set entry '%s' in vault '%s' (secret=%v)", req.Key, vaultName, isSecret)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "entry set successfully"})
	}
}

func newListEntriesHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		vaultName := vars["name"]

		entries, err := store.ListEntries(r.Context(), vaultName)
		if err != nil {
			if err.Error() == "vault '"+vaultName+"' not found" {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"entries": entries,
			"count":   len(entries),
		})
	}
}

func newDeleteEntryHandlerFunc(store *vaults.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !store.IsEnabled() {
			http.Error(w, `{"error":"vaults not configured: RUNQY_VAULT_MASTER_KEY not set"}`, http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		vaultName := vars["name"]
		key := vars["key"]

		if err := store.DeleteEntry(r.Context(), vaultName, key); err != nil {
			errMsg := err.Error()
			if errMsg == "vault '"+vaultName+"' not found" ||
				errMsg == "key '"+key+"' not found in vault '"+vaultName+"'" {
				http.Error(w, `{"error":"`+errMsg+`"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"`+errMsg+`"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("[VAULTS] Deleted entry '%s' from vault '%s'", key, vaultName)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "entry deleted successfully"})
	}
}
