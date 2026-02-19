package vaults

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	// validKeyName matches environment-variable-safe key names
	validKeyName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

	// reservedKeyNames are names that must not be overridden via vaults
	reservedKeyNames = map[string]bool{
		"PATH":        true,
		"HOME":        true,
		"VIRTUAL_ENV": true,
		"PYTHONPATH":  true,
	}
)

// Store handles vault persistence to the database.
type Store struct {
	db        *sqlx.DB
	encryptor *Encryptor
}

// NewStore creates a new vault store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{
		db:        db,
		encryptor: GetEncryptor(),
	}
}

// IsEnabled returns true if the vaults feature is enabled (encryption configured).
func (s *Store) IsEnabled() bool {
	return s.encryptor.IsEnabled()
}

// --- Vault CRUD ---

// CreateVault creates a new vault.
func (s *Store) CreateVault(ctx context.Context, name, description string) (*Vault, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	now := time.Now()
	vault := &Vault{
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	var query string
	if s.db.DriverName() == "sqlite" {
		query = `INSERT INTO vaults (name, description, created_at, updated_at)
		         VALUES (?, ?, ?, ?)`
		result, err := s.db.ExecContext(ctx, query, vault.Name, vault.Description, vault.CreatedAt, vault.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get vault ID: %w", err)
		}
		vault.ID = id
	} else {
		query = `INSERT INTO vaults (name, description, created_at, updated_at)
		         VALUES ($1, $2, $3, $4) RETURNING id`
		err := s.db.QueryRowContext(ctx, query, vault.Name, vault.Description, vault.CreatedAt, vault.UpdatedAt).Scan(&vault.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault: %w", err)
		}
	}

	return vault, nil
}

// GetVault retrieves a vault by name.
func (s *Store) GetVault(ctx context.Context, name string) (*Vault, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	var vault Vault
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT id, name, description, created_at, updated_at FROM vaults WHERE name = ?`
	} else {
		query = `SELECT id, name, description, created_at, updated_at FROM vaults WHERE name = $1`
	}

	err := s.db.GetContext(ctx, &vault, query, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}

	return &vault, nil
}

// GetVaultByID retrieves a vault by ID.
func (s *Store) GetVaultByID(ctx context.Context, id int64) (*Vault, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	var vault Vault
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT id, name, description, created_at, updated_at FROM vaults WHERE id = ?`
	} else {
		query = `SELECT id, name, description, created_at, updated_at FROM vaults WHERE id = $1`
	}

	err := s.db.GetContext(ctx, &vault, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}

	return &vault, nil
}

// ListVaults returns all vaults with entry counts.
func (s *Store) ListVaults(ctx context.Context) ([]VaultSummary, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT v.name, v.description, COUNT(e.id) as entry_count
		         FROM vaults v
		         LEFT JOIN vault_entries e ON v.id = e.vault_id
		         GROUP BY v.id, v.name, v.description
		         ORDER BY v.name`
	} else {
		query = `SELECT v.name, v.description, COUNT(e.id) as entry_count
		         FROM vaults v
		         LEFT JOIN vault_entries e ON v.id = e.vault_id
		         GROUP BY v.id, v.name, v.description
		         ORDER BY v.name`
	}

	var summaries []VaultSummary
	err := s.db.SelectContext(ctx, &summaries, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	return summaries, nil
}

// DeleteVault deletes a vault and all its entries.
func (s *Store) DeleteVault(ctx context.Context, name string) error {
	if !s.encryptor.IsEnabled() {
		return ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, name)
	if err != nil {
		return err
	}
	if vault == nil {
		return fmt.Errorf("vault '%s' not found", name)
	}

	// Delete entries first (foreign key)
	var deleteEntries, deleteVault string
	if s.db.DriverName() == "sqlite" {
		deleteEntries = `DELETE FROM vault_entries WHERE vault_id = ?`
		deleteVault = `DELETE FROM vaults WHERE id = ?`
	} else {
		deleteEntries = `DELETE FROM vault_entries WHERE vault_id = $1`
		deleteVault = `DELETE FROM vaults WHERE id = $1`
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, deleteEntries, vault.ID); err != nil {
		return fmt.Errorf("failed to delete vault entries: %w", err)
	}

	if _, err := tx.ExecContext(ctx, deleteVault, vault.ID); err != nil {
		return fmt.Errorf("failed to delete vault: %w", err)
	}

	return tx.Commit()
}

// VaultExists checks if a vault with the given name exists.
func (s *Store) VaultExists(ctx context.Context, name string) (bool, error) {
	if !s.encryptor.IsEnabled() {
		return false, ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, name)
	if err != nil {
		return false, err
	}
	return vault != nil, nil
}

// --- Vault Entry CRUD ---

// ValidateKeyName checks that a vault key name is safe for use as an environment variable.
func ValidateKeyName(key string) error {
	if !validKeyName.MatchString(key) {
		return fmt.Errorf("invalid key name %q: must match [A-Za-z_][A-Za-z0-9_]*", key)
	}
	if reservedKeyNames[strings.ToUpper(key)] {
		return fmt.Errorf("reserved key name %q: cannot override system variables (PATH, HOME, VIRTUAL_ENV, PYTHONPATH)", key)
	}
	return nil
}

// SetEntry sets a key-value pair in a vault (creates or updates).
func (s *Store) SetEntry(ctx context.Context, vaultName, key, value string, isSecret bool) error {
	if !s.encryptor.IsEnabled() {
		return ErrVaultsDisabled
	}

	if err := ValidateKeyName(key); err != nil {
		return err
	}

	vault, err := s.GetVault(ctx, vaultName)
	if err != nil {
		return err
	}
	if vault == nil {
		return fmt.Errorf("vault '%s' not found", vaultName)
	}

	// Encrypt the value
	encrypted, err := s.encryptor.EncryptString(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	now := time.Now()

	// Check if entry exists
	existing, err := s.getEntry(ctx, vault.ID, key)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing entry
		var query string
		if s.db.DriverName() == "sqlite" {
			query = `UPDATE vault_entries SET value = ?, is_secret = ?, updated_at = ? WHERE id = ?`
		} else {
			query = `UPDATE vault_entries SET value = $1, is_secret = $2, updated_at = $3 WHERE id = $4`
		}
		_, err = s.db.ExecContext(ctx, query, encrypted, isSecret, now, existing.ID)
	} else {
		// Insert new entry
		var query string
		if s.db.DriverName() == "sqlite" {
			query = `INSERT INTO vault_entries (vault_id, key, value, is_secret, created_at, updated_at)
			         VALUES (?, ?, ?, ?, ?, ?)`
		} else {
			query = `INSERT INTO vault_entries (vault_id, key, value, is_secret, created_at, updated_at)
			         VALUES ($1, $2, $3, $4, $5, $6)`
		}
		_, err = s.db.ExecContext(ctx, query, vault.ID, key, encrypted, isSecret, now, now)
	}

	if err != nil {
		return fmt.Errorf("failed to set entry: %w", err)
	}

	// Update vault's updated_at timestamp
	var updateVault string
	if s.db.DriverName() == "sqlite" {
		updateVault = `UPDATE vaults SET updated_at = ? WHERE id = ?`
	} else {
		updateVault = `UPDATE vaults SET updated_at = $1 WHERE id = $2`
	}
	s.db.ExecContext(ctx, updateVault, now, vault.ID)

	return nil
}

// GetEntry retrieves a single entry's decrypted value.
func (s *Store) GetEntry(ctx context.Context, vaultName, key string) (string, bool, error) {
	if !s.encryptor.IsEnabled() {
		return "", false, ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, vaultName)
	if err != nil {
		return "", false, err
	}
	if vault == nil {
		return "", false, fmt.Errorf("vault '%s' not found", vaultName)
	}

	entry, err := s.getEntry(ctx, vault.ID, key)
	if err != nil {
		return "", false, err
	}
	if entry == nil {
		return "", false, fmt.Errorf("key '%s' not found in vault '%s'", key, vaultName)
	}

	value, err := s.encryptor.DecryptString(entry.Value)
	if err != nil {
		return "", false, fmt.Errorf("failed to decrypt value: %w", err)
	}

	return value, entry.IsSecret, nil
}

// getEntry retrieves a raw entry by vault ID and key.
func (s *Store) getEntry(ctx context.Context, vaultID int64, key string) (*VaultEntry, error) {
	var entry VaultEntry
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT id, vault_id, key, value, is_secret, created_at, updated_at
		         FROM vault_entries WHERE vault_id = ? AND key = ?`
	} else {
		query = `SELECT id, vault_id, key, value, is_secret, created_at, updated_at
		         FROM vault_entries WHERE vault_id = $1 AND key = $2`
	}

	err := s.db.GetContext(ctx, &entry, query, vaultID, key)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	return &entry, nil
}

// DeleteEntry removes an entry from a vault.
func (s *Store) DeleteEntry(ctx context.Context, vaultName, key string) error {
	if !s.encryptor.IsEnabled() {
		return ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, vaultName)
	if err != nil {
		return err
	}
	if vault == nil {
		return fmt.Errorf("vault '%s' not found", vaultName)
	}

	var query string
	if s.db.DriverName() == "sqlite" {
		query = `DELETE FROM vault_entries WHERE vault_id = ? AND key = ?`
	} else {
		query = `DELETE FROM vault_entries WHERE vault_id = $1 AND key = $2`
	}

	result, err := s.db.ExecContext(ctx, query, vault.ID, key)
	if err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("key '%s' not found in vault '%s'", key, vaultName)
	}

	return nil
}

// ListEntries returns all entries for a vault (values masked if secret).
func (s *Store) ListEntries(ctx context.Context, vaultName string) ([]VaultEntryView, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, vaultName)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return nil, fmt.Errorf("vault '%s' not found", vaultName)
	}

	var entries []VaultEntry
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT id, vault_id, key, value, is_secret, created_at, updated_at
		         FROM vault_entries WHERE vault_id = ? ORDER BY key`
	} else {
		query = `SELECT id, vault_id, key, value, is_secret, created_at, updated_at
		         FROM vault_entries WHERE vault_id = $1 ORDER BY key`
	}

	if err := s.db.SelectContext(ctx, &entries, query, vault.ID); err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	views := make([]VaultEntryView, len(entries))
	for i, e := range entries {
		value, _ := s.encryptor.DecryptString(e.Value)
		if e.IsSecret {
			value = MaskSecret(value)
		}
		views[i] = VaultEntryView{
			Key:       e.Key,
			Value:     value,
			IsSecret:  e.IsSecret,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
			UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
		}
	}

	return views, nil
}

// GetVaultDetail returns a vault with all its entries (values masked if secret).
func (s *Store) GetVaultDetail(ctx context.Context, name string) (*VaultDetail, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, name)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return nil, nil
	}

	entries, err := s.ListEntries(ctx, name)
	if err != nil {
		return nil, err
	}

	return &VaultDetail{
		Name:        vault.Name,
		Description: vault.Description,
		Entries:     entries,
		CreatedAt:   vault.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   vault.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetVaultData returns decrypted vault data for worker injection.
// This returns ALL values decrypted - only call for trusted use (worker bootstrap).
func (s *Store) GetVaultData(ctx context.Context, vaultName string) (VaultData, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	vault, err := s.GetVault(ctx, vaultName)
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return nil, fmt.Errorf("vault '%s' not found", vaultName)
	}

	var entries []VaultEntry
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT id, vault_id, key, value, is_secret, created_at, updated_at
		         FROM vault_entries WHERE vault_id = ?`
	} else {
		query = `SELECT id, vault_id, key, value, is_secret, created_at, updated_at
		         FROM vault_entries WHERE vault_id = $1`
	}

	if err := s.db.SelectContext(ctx, &entries, query, vault.ID); err != nil {
		return nil, fmt.Errorf("failed to get vault entries: %w", err)
	}

	data := make(VaultData, len(entries))
	for _, e := range entries {
		value, err := s.encryptor.DecryptString(e.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt entry '%s': %w", e.Key, err)
		}
		data[e.Key] = value
	}

	return data, nil
}

// GetMultipleVaultsData returns combined decrypted data from multiple vaults.
// Later vaults override earlier ones if keys conflict.
func (s *Store) GetMultipleVaultsData(ctx context.Context, vaultNames []string) (VaultData, error) {
	if !s.encryptor.IsEnabled() {
		return nil, ErrVaultsDisabled
	}

	combined := make(VaultData)
	for _, name := range vaultNames {
		data, err := s.GetVaultData(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("vault '%s': %w", name, err)
		}
		for k, v := range data {
			combined[k] = v
		}
	}

	return combined, nil
}
