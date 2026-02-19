package vaults

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary SQLite database with the vault schema.
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "vault-test-*.db")
	require.NoError(t, err)
	tmpFile.Close()

	db, err := sqlx.Connect("sqlite", tmpFile.Name())
	require.NoError(t, err)

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS vaults (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS vault_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vault_id INTEGER NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
		key TEXT NOT NULL,
		value BLOB NOT NULL,
		is_secret INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(vault_id, key)
	);`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpFile.Name())
	})

	return db
}

// newTestStore creates a Store with a real SQLite DB and a test encryptor,
// bypassing the global singleton.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	key := make([]byte, keyLength)
	_, err := rand.Read(key)
	require.NoError(t, err)

	enc := &Encryptor{key: key, enabled: true}
	db := setupTestDB(t)

	return &Store{db: db, encryptor: enc}
}

func TestCreateVaultAndGetVault(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	vault, err := store.CreateVault(ctx, "myvault", "test description")
	require.NoError(t, err)
	assert.Equal(t, "myvault", vault.Name)
	assert.Equal(t, "test description", vault.Description)
	assert.NotZero(t, vault.ID)

	got, err := store.GetVault(ctx, "myvault")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, vault.ID, got.ID)
	assert.Equal(t, "myvault", got.Name)
}

func TestCreateVaultDuplicate(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "dup", "first")
	require.NoError(t, err)

	_, err = store.CreateVault(ctx, "dup", "second")
	assert.Error(t, err)
}

func TestGetVaultNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	got, err := store.GetVault(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeleteVaultThenGetVault(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "todelete", "will be deleted")
	require.NoError(t, err)

	err = store.DeleteVault(ctx, "todelete")
	require.NoError(t, err)

	got, err := store.GetVault(ctx, "todelete")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeleteVaultNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.DeleteVault(ctx, "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVaultNotFound)
}

func TestSetEntryAndGetEntry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "v1", "")
	require.NoError(t, err)

	err = store.SetEntry(ctx, "v1", "API_KEY", "secret123", true)
	require.NoError(t, err)

	value, isSecret, err := store.GetEntry(ctx, "v1", "API_KEY")
	require.NoError(t, err)
	assert.Equal(t, "secret123", value)
	assert.True(t, isSecret)
}

func TestSetEntryUpdatesExisting(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "v1", "")
	require.NoError(t, err)

	err = store.SetEntry(ctx, "v1", "KEY", "first", true)
	require.NoError(t, err)

	err = store.SetEntry(ctx, "v1", "KEY", "updated", false)
	require.NoError(t, err)

	value, isSecret, err := store.GetEntry(ctx, "v1", "KEY")
	require.NoError(t, err)
	assert.Equal(t, "updated", value)
	assert.False(t, isSecret)
}

func TestSetEntryVaultNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.SetEntry(ctx, "nonexistent", "KEY", "val", true)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVaultNotFound)
}

func TestDeleteEntryAndGetEntry(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "v1", "")
	require.NoError(t, err)

	err = store.SetEntry(ctx, "v1", "KEY", "val", true)
	require.NoError(t, err)

	err = store.DeleteEntry(ctx, "v1", "KEY")
	require.NoError(t, err)

	_, _, err = store.GetEntry(ctx, "v1", "KEY")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEntryNotFound)
}

func TestDeleteEntryNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "v1", "")
	require.NoError(t, err)

	err = store.DeleteEntry(ctx, "v1", "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEntryNotFound)
}

func TestListVaults(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "alpha", "first")
	require.NoError(t, err)
	_, err = store.CreateVault(ctx, "beta", "second")
	require.NoError(t, err)

	// Add an entry to alpha to test entry_count
	err = store.SetEntry(ctx, "alpha", "KEY1", "v1", true)
	require.NoError(t, err)

	summaries, err := store.ListVaults(ctx)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)

	// Sorted by name
	assert.Equal(t, "alpha", summaries[0].Name)
	assert.Equal(t, 1, summaries[0].EntryCount)
	assert.Equal(t, "beta", summaries[1].Name)
	assert.Equal(t, 0, summaries[1].EntryCount)
}

func TestListEntries(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "v1", "")
	require.NoError(t, err)

	err = store.SetEntry(ctx, "v1", "SECRET_KEY", "mysecret", true)
	require.NoError(t, err)
	err = store.SetEntry(ctx, "v1", "PUBLIC_KEY", "myvalue", false)
	require.NoError(t, err)

	entries, err := store.ListEntries(ctx, "v1")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Sorted by key
	assert.Equal(t, "PUBLIC_KEY", entries[0].Key)
	assert.Equal(t, "myvalue", entries[0].Value) // non-secret: plaintext
	assert.False(t, entries[0].IsSecret)

	assert.Equal(t, "SECRET_KEY", entries[1].Key)
	assert.Equal(t, "my****et", entries[1].Value) // secret: masked
	assert.True(t, entries[1].IsSecret)
}

func TestVaultExists(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	exists, err := store.VaultExists(ctx, "nope")
	require.NoError(t, err)
	assert.False(t, exists)

	_, err = store.CreateVault(ctx, "exists", "")
	require.NoError(t, err)

	exists, err = store.VaultExists(ctx, "exists")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestGetVaultByID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	vault, err := store.CreateVault(ctx, "byid", "")
	require.NoError(t, err)

	got, err := store.GetVaultByID(ctx, vault.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "byid", got.Name)

	// Non-existent ID
	got, err = store.GetVaultByID(ctx, 9999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetVaultDetail(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "detailed", "desc")
	require.NoError(t, err)
	err = store.SetEntry(ctx, "detailed", "MY_KEY", "val", false)
	require.NoError(t, err)

	detail, err := store.GetVaultDetail(ctx, "detailed")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "detailed", detail.Name)
	assert.Equal(t, "desc", detail.Description)
	assert.Len(t, detail.Entries, 1)
	assert.Equal(t, "MY_KEY", detail.Entries[0].Key)
}

func TestGetVaultData(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "data", "")
	require.NoError(t, err)
	err = store.SetEntry(ctx, "data", "A", "val_a", true)
	require.NoError(t, err)
	err = store.SetEntry(ctx, "data", "B", "val_b", false)
	require.NoError(t, err)

	data, err := store.GetVaultData(ctx, "data")
	require.NoError(t, err)
	assert.Equal(t, "val_a", data["A"])
	assert.Equal(t, "val_b", data["B"])
}

func TestGetVaultDataNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.GetVaultData(ctx, "nope")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVaultNotFound)
}

func TestValidateKeyName(t *testing.T) {
	// Valid keys
	assert.NoError(t, ValidateKeyName("MY_KEY"))
	assert.NoError(t, ValidateKeyName("key123"))
	assert.NoError(t, ValidateKeyName("_private"))
	assert.NoError(t, ValidateKeyName("a"))

	// Invalid keys
	assert.Error(t, ValidateKeyName("123start"))
	assert.Error(t, ValidateKeyName("has-dash"))
	assert.Error(t, ValidateKeyName("has space"))
	assert.Error(t, ValidateKeyName(""))

	// Reserved keys
	assert.Error(t, ValidateKeyName("PATH"))
	assert.Error(t, ValidateKeyName("HOME"))
	assert.Error(t, ValidateKeyName("VIRTUAL_ENV"))
	assert.Error(t, ValidateKeyName("PYTHONPATH"))
}

func TestStoreDisabledEncryptor(t *testing.T) {
	db := setupTestDB(t)
	store := &Store{db: db, encryptor: &Encryptor{enabled: false}}
	ctx := context.Background()

	_, err := store.CreateVault(ctx, "v", "")
	assert.ErrorIs(t, err, ErrVaultsDisabled)

	_, err = store.GetVault(ctx, "v")
	assert.ErrorIs(t, err, ErrVaultsDisabled)

	_, err = store.ListVaults(ctx)
	assert.ErrorIs(t, err, ErrVaultsDisabled)

	err = store.DeleteVault(ctx, "v")
	assert.ErrorIs(t, err, ErrVaultsDisabled)

	err = store.SetEntry(ctx, "v", "k", "v", true)
	assert.ErrorIs(t, err, ErrVaultsDisabled)

	_, _, err = store.GetEntry(ctx, "v", "k")
	assert.ErrorIs(t, err, ErrVaultsDisabled)
}
