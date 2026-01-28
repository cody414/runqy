package vaults

import "time"

// Vault represents a named collection of secret key-value pairs.
type Vault struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// VaultEntry represents a single key-value pair within a vault.
// Values are stored encrypted in the database.
type VaultEntry struct {
	ID        int64     `db:"id" json:"id"`
	VaultID   int64     `db:"vault_id" json:"vault_id"`
	Key       string    `db:"key" json:"key"`
	Value     []byte    `db:"value" json:"-"`          // Encrypted value, never exposed in JSON
	IsSecret  bool      `db:"is_secret" json:"is_secret"` // If true, mask value in API responses
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// VaultData is a map of key-value pairs used for worker injection.
// Keys map to decrypted string values.
type VaultData map[string]string

// VaultSummary is a lightweight view of a vault for listing.
type VaultSummary struct {
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	EntryCount  int    `db:"entry_count" json:"entry_count"`
}

// VaultEntryView is used for API responses - masks secret values.
type VaultEntryView struct {
	Key       string `json:"key"`
	Value     string `json:"value"` // Masked if IsSecret is true
	IsSecret  bool   `json:"is_secret"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// VaultDetail is a full vault view with entries (values masked).
type VaultDetail struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Entries     []VaultEntryView `json:"entries"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}
