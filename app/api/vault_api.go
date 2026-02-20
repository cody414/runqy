package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/Publikey/runqy/models"
	"github.com/Publikey/runqy/vaults"
	"github.com/gin-gonic/gin"
)

// VaultsListResponse is the response for listing vaults
type VaultsListResponse struct {
	Vaults []vaults.VaultSummary `json:"vaults"`
	Count  int                   `json:"count"`
}

// CreateVaultRequest is the request body for creating a vault
type CreateVaultRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// SetEntryRequest is the request body for setting a vault entry
type SetEntryRequest struct {
	Key      string `json:"key" binding:"required"`
	Value    string `json:"value" binding:"required"`
	IsSecret *bool  `json:"is_secret"` // Defaults to true if not provided
}

// checkVaultsEnabled returns an error response if vaults are disabled
func checkVaultsEnabled(c *gin.Context, store *vaults.Store) bool {
	if !store.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, models.APIErrorResponse{
			Errors: []string{"vaults not enabled. Set RUNQY_VAULT_MASTER_KEY environment variable. Generate a key with: openssl rand -base64 32"},
		})
		return false
	}
	return true
}

// ListVaults returns all vaults
// @Summary List all vaults
// @Description List all vaults with entry counts
// @Tags vaults
// @Produce json
// @Success 200 {object} VaultsListResponse
// @Failure 503 {object} map[string]string
// @Router /api/vaults [get]
func ListVaults(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		summaries, err := store.ListVaults(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		c.JSON(http.StatusOK, VaultsListResponse{
			Vaults: summaries,
			Count:  len(summaries),
		})
	}
}

// CreateVault creates a new vault
// @Summary Create a new vault
// @Description Create a new vault with the given name and description
// @Tags vaults
// @Accept json
// @Produce json
// @Param request body CreateVaultRequest true "Vault creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/vaults [post]
func CreateVault(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		var req CreateVaultRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		// Check if vault already exists
		exists, err := store.VaultExists(c.Request.Context(), req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, models.APIErrorResponse{Errors: []string{"vault already exists"}})
			return
		}

		vault, err := store.CreateVault(c.Request.Context(), req.Name, req.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		log.Printf("[VAULTS] Created vault: %s", vault.Name)
		c.JSON(http.StatusCreated, gin.H{
			"message": "vault created successfully",
			"vault": gin.H{
				"name":        vault.Name,
				"description": vault.Description,
			},
		})
	}
}

// GetVault returns a vault with its entries (values masked)
// @Summary Get vault details
// @Description Get vault details with entries (secret values masked)
// @Tags vaults
// @Produce json
// @Param name path string true "Vault name"
// @Success 200 {object} vaults.VaultDetail
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/vaults/{name} [get]
func GetVault(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		name := c.Param("name")
		detail, err := store.GetVaultDetail(c.Request.Context(), name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}
		if detail == nil {
			c.JSON(http.StatusNotFound, models.APIErrorResponse{Errors: []string{"vault not found"}})
			return
		}

		c.JSON(http.StatusOK, detail)
	}
}

// DeleteVault deletes a vault and all its entries
// @Summary Delete a vault
// @Description Delete a vault and all its entries
// @Tags vaults
// @Produce json
// @Param name path string true "Vault name"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/vaults/{name} [delete]
func DeleteVault(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		name := c.Param("name")

		// Check if vault exists
		exists, err := store.VaultExists(c.Request.Context(), name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, models.APIErrorResponse{Errors: []string{"vault not found"}})
			return
		}

		if err := store.DeleteVault(c.Request.Context(), name); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		log.Printf("[VAULTS] Deleted vault: %s", name)
		c.JSON(http.StatusOK, gin.H{"message": "vault deleted successfully"})
	}
}

// SetEntry sets a key-value pair in a vault
// @Summary Set a vault entry
// @Description Set or update a key-value pair in a vault
// @Tags vaults
// @Accept json
// @Produce json
// @Param name path string true "Vault name"
// @Param request body SetEntryRequest true "Entry to set"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/vaults/{name}/entries [post]
func SetEntry(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		vaultName := c.Param("name")

		var req SetEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		// Default isSecret to true
		isSecret := true
		if req.IsSecret != nil {
			isSecret = *req.IsSecret
		}

		if err := store.SetEntry(c.Request.Context(), vaultName, req.Key, req.Value, isSecret); err != nil {
			if errors.Is(err, vaults.ErrVaultNotFound) {
				c.JSON(http.StatusNotFound, models.APIErrorResponse{Errors: []string{err.Error()}})
				return
			}
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		log.Printf("[VAULTS] Set entry '%s' in vault '%s' (secret=%v)", req.Key, vaultName, isSecret)
		c.JSON(http.StatusOK, gin.H{"message": "entry set successfully"})
	}
}

// ListEntries returns all entries for a vault (values masked)
// @Summary List vault entries
// @Description List all entries in a vault (secret values masked)
// @Tags vaults
// @Produce json
// @Param name path string true "Vault name"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/vaults/{name}/entries [get]
func ListEntries(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		vaultName := c.Param("name")

		entries, err := store.ListEntries(c.Request.Context(), vaultName)
		if err != nil {
			if errors.Is(err, vaults.ErrVaultNotFound) {
				c.JSON(http.StatusNotFound, models.APIErrorResponse{Errors: []string{err.Error()}})
				return
			}
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"entries": entries,
			"count":   len(entries),
		})
	}
}

// DeleteEntry removes an entry from a vault
// @Summary Delete a vault entry
// @Description Remove a key-value pair from a vault
// @Tags vaults
// @Produce json
// @Param name path string true "Vault name"
// @Param key path string true "Entry key"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/vaults/{name}/entries/{key} [delete]
func DeleteEntry(store *vaults.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkVaultsEnabled(c, store) {
			return
		}

		vaultName := c.Param("name")
		key := c.Param("key")

		if err := store.DeleteEntry(c.Request.Context(), vaultName, key); err != nil {
			if errors.Is(err, vaults.ErrVaultNotFound) || errors.Is(err, vaults.ErrEntryNotFound) {
				c.JSON(http.StatusNotFound, models.APIErrorResponse{Errors: []string{err.Error()}})
				return
			}
			c.JSON(http.StatusInternalServerError, models.APIErrorResponse{Errors: []string{err.Error()}})
			return
		}

		log.Printf("[VAULTS] Deleted entry '%s' from vault '%s'", key, vaultName)
		c.JSON(http.StatusOK, gin.H{"message": "entry deleted successfully"})
	}
}
