package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Publikey/runqy/models"
	"github.com/Publikey/runqy/vaults"
	"github.com/spf13/cobra"
)

var (
	vaultDescription string
	vaultNoSecret    bool
	vaultForce       bool
)

// vaultCmd represents the vault command
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Vault management commands",
	Long: `Vault management commands for creating, viewing, and managing vaults.

Vaults are named collections of key-value pairs that are encrypted at rest
and injected into workers as environment variables at bootstrap time.

Examples:
  runqy vault list
  runqy vault create api-keys -d "API keys for external services"
  runqy vault show api-keys
  runqy vault set api-keys OPENAI_API_KEY sk-xxx
  runqy vault get api-keys OPENAI_API_KEY
  runqy vault unset api-keys OPENAI_API_KEY
  runqy vault entries api-keys
  runqy vault delete api-keys --force

Remote mode:
  runqy --server https://runqy.example.com -k API_KEY vault list`,
}

// vaultListCmd lists all vaults
var vaultListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vaults",
	Long:  "List all vaults with their entry counts.",
	RunE:  runVaultList,
}

// vaultCreateCmd creates a new vault
var vaultCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new vault",
	Long: `Create a new vault with the given name.

The vault name should be lowercase letters, numbers, and hyphens only.`,
	Args: cobra.ExactArgs(1),
	RunE: runVaultCreate,
}

// vaultShowCmd shows vault details
var vaultShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show vault details",
	Long:  "Show vault details including all entries (secret values are masked).",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultShow,
}

// vaultDeleteCmd deletes a vault
var vaultDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a vault",
	Long: `Delete a vault and all its entries.

Use --force to skip confirmation.`,
	Args: cobra.ExactArgs(1),
	RunE: runVaultDelete,
}

// vaultSetCmd sets a vault entry
var vaultSetCmd = &cobra.Command{
	Use:   "set <vault> <key> <value>",
	Short: "Set a vault entry",
	Long: `Set or update a key-value pair in a vault.

By default, values are marked as secrets and will be masked in API responses.
Use --no-secret to store the value as non-secret (visible in API responses).`,
	Args: cobra.ExactArgs(3),
	RunE: runVaultSet,
}

// vaultGetCmd gets a vault entry
var vaultGetCmd = &cobra.Command{
	Use:   "get <vault> <key>",
	Short: "Get a vault entry value",
	Long:  "Get the decrypted value of a vault entry.",
	Args:  cobra.ExactArgs(2),
	RunE:  runVaultGet,
}

// vaultUnsetCmd removes a vault entry
var vaultUnsetCmd = &cobra.Command{
	Use:   "unset <vault> <key>",
	Short: "Remove a vault entry",
	Long:  "Remove a key-value pair from a vault.",
	Args:  cobra.ExactArgs(2),
	RunE:  runVaultUnset,
}

// vaultEntriesCmd lists vault entries
var vaultEntriesCmd = &cobra.Command{
	Use:   "entries <vault>",
	Short: "List vault entries",
	Long:  "List all entries in a vault (secret values are masked).",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultEntries,
}

func init() {
	rootCmd.AddCommand(vaultCmd)
	vaultCmd.AddCommand(vaultListCmd)
	vaultCmd.AddCommand(vaultCreateCmd)
	vaultCmd.AddCommand(vaultShowCmd)
	vaultCmd.AddCommand(vaultDeleteCmd)
	vaultCmd.AddCommand(vaultSetCmd)
	vaultCmd.AddCommand(vaultGetCmd)
	vaultCmd.AddCommand(vaultUnsetCmd)
	vaultCmd.AddCommand(vaultEntriesCmd)

	// Flags
	vaultCreateCmd.Flags().StringVarP(&vaultDescription, "description", "d", "", "Vault description")
	vaultSetCmd.Flags().BoolVar(&vaultNoSecret, "no-secret", false, "Store value as non-secret (visible in API)")
	vaultDeleteCmd.Flags().BoolVar(&vaultForce, "force", false, "Skip confirmation")
}

func getVaultStore() (*vaults.Store, error) {
	cfg := GetConfig()
	db, err := models.BuildDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return vaults.NewStore(db), nil
}

func runVaultList(cmd *cobra.Command, args []string) error {
	// Remote mode: use API client
	if IsRemoteMode() {
		return runVaultListRemote()
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	summaries, err := store.ListVaults(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list vaults: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Println("No vaults found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tENTRIES")

	for _, v := range summaries {
		desc := v.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%d\n", v.Name, desc, v.EntryCount)
	}

	w.Flush()
	return nil
}

func runVaultListRemote() error {
	client := NewAPIClient()

	summaries, err := client.ListVaultsAPI()
	if err != nil {
		return err
	}

	if len(summaries) == 0 {
		fmt.Println("No vaults found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tENTRIES")

	for _, v := range summaries {
		desc := v.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%d\n", v.Name, desc, v.EntryCount)
	}

	w.Flush()
	return nil
}

func runVaultCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Remote mode: use API client
	if IsRemoteMode() {
		client := NewAPIClient()
		if err := client.CreateVaultAPI(name, vaultDescription); err != nil {
			return err
		}
		fmt.Printf("Vault '%s' created successfully.\n", name)
		return nil
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	// Check if vault already exists
	exists, err := store.VaultExists(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to check vault existence: %w", err)
	}
	if exists {
		return fmt.Errorf("vault '%s' already exists", name)
	}

	_, err = store.CreateVault(context.Background(), name, vaultDescription)
	if err != nil {
		return fmt.Errorf("failed to create vault: %w", err)
	}

	fmt.Printf("Vault '%s' created successfully.\n", name)
	return nil
}

func runVaultShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Remote mode: use API client
	if IsRemoteMode() {
		return runVaultShowRemote(name)
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	detail, err := store.GetVaultDetail(context.Background(), name)
	if err != nil {
		return fmt.Errorf("failed to get vault: %w", err)
	}
	if detail == nil {
		return fmt.Errorf("vault '%s' not found", name)
	}

	fmt.Printf("Vault: %s\n", detail.Name)
	if detail.Description != "" {
		fmt.Printf("Description: %s\n", detail.Description)
	}
	fmt.Printf("Created: %s\n", detail.CreatedAt)
	fmt.Printf("Updated: %s\n", detail.UpdatedAt)
	fmt.Println()

	if len(detail.Entries) == 0 {
		fmt.Println("No entries.")
		return nil
	}

	fmt.Printf("Entries (%d):\n", len(detail.Entries))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE\tSECRET")

	for _, e := range detail.Entries {
		secret := "yes"
		if !e.IsSecret {
			secret = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Key, e.Value, secret)
	}

	w.Flush()
	return nil
}

func runVaultShowRemote(name string) error {
	client := NewAPIClient()

	detail, err := client.GetVaultAPI(name)
	if err != nil {
		return err
	}

	fmt.Printf("Vault: %s\n", detail.Name)
	if detail.Description != "" {
		fmt.Printf("Description: %s\n", detail.Description)
	}
	fmt.Printf("Created: %s\n", detail.CreatedAt)
	fmt.Printf("Updated: %s\n", detail.UpdatedAt)
	fmt.Println()

	if len(detail.Entries) == 0 {
		fmt.Println("No entries.")
		return nil
	}

	fmt.Printf("Entries (%d):\n", len(detail.Entries))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE\tSECRET")

	for _, e := range detail.Entries {
		secret := "yes"
		if !e.IsSecret {
			secret = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Key, e.Value, secret)
	}

	w.Flush()
	return nil
}

func runVaultDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !vaultForce {
		fmt.Printf("Are you sure you want to delete vault '%s' and all its entries? (y/N): ", name)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remote mode: use API client
	if IsRemoteMode() {
		client := NewAPIClient()
		if err := client.DeleteVaultAPI(name); err != nil {
			return err
		}
		fmt.Printf("Vault '%s' deleted successfully.\n", name)
		return nil
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	if err := store.DeleteVault(context.Background(), name); err != nil {
		return fmt.Errorf("failed to delete vault: %w", err)
	}

	fmt.Printf("Vault '%s' deleted successfully.\n", name)
	return nil
}

func runVaultSet(cmd *cobra.Command, args []string) error {
	vaultName := args[0]
	key := args[1]
	value := args[2]

	isSecret := !vaultNoSecret

	// Remote mode: use API client
	if IsRemoteMode() {
		client := NewAPIClient()
		if err := client.SetEntryAPI(vaultName, key, value, &isSecret); err != nil {
			return err
		}
		fmt.Printf("Entry '%s' set in vault '%s'.\n", key, vaultName)
		return nil
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	if err := store.SetEntry(context.Background(), vaultName, key, value, isSecret); err != nil {
		return fmt.Errorf("failed to set entry: %w", err)
	}

	fmt.Printf("Entry '%s' set in vault '%s'.\n", key, vaultName)
	return nil
}

func runVaultGet(cmd *cobra.Command, args []string) error {
	vaultName := args[0]
	key := args[1]

	// Remote mode: not supported (would expose secrets)
	if IsRemoteMode() {
		return fmt.Errorf("vault get is not supported in remote mode for security reasons")
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	value, _, err := store.GetEntry(context.Background(), vaultName, key)
	if err != nil {
		return fmt.Errorf("failed to get entry: %w", err)
	}

	fmt.Println(value)
	return nil
}

func runVaultUnset(cmd *cobra.Command, args []string) error {
	vaultName := args[0]
	key := args[1]

	// Remote mode: use API client
	if IsRemoteMode() {
		client := NewAPIClient()
		if err := client.DeleteEntryAPI(vaultName, key); err != nil {
			return err
		}
		fmt.Printf("Entry '%s' removed from vault '%s'.\n", key, vaultName)
		return nil
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	if err := store.DeleteEntry(context.Background(), vaultName, key); err != nil {
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	fmt.Printf("Entry '%s' removed from vault '%s'.\n", key, vaultName)
	return nil
}

func runVaultEntries(cmd *cobra.Command, args []string) error {
	vaultName := args[0]

	// Remote mode: use API client
	if IsRemoteMode() {
		return runVaultEntriesRemote(vaultName)
	}

	// Local mode: direct database access
	store, err := getVaultStore()
	if err != nil {
		return err
	}

	if !store.IsEnabled() {
		return fmt.Errorf("vaults not configured: RUNQY_VAULT_MASTER_KEY not set")
	}

	entries, err := store.ListEntries(context.Background(), vaultName)
	if err != nil {
		return fmt.Errorf("failed to list entries: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No entries.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE\tSECRET\tUPDATED")

	for _, e := range entries {
		secret := "yes"
		if !e.IsSecret {
			secret = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Key, e.Value, secret, e.UpdatedAt)
	}

	w.Flush()
	return nil
}

func runVaultEntriesRemote(vaultName string) error {
	client := NewAPIClient()

	entries, err := client.GetEntriesAPI(vaultName)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No entries.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE\tSECRET\tUPDATED")

	for _, e := range entries {
		secret := "yes"
		if !e.IsSecret {
			secret = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Key, e.Value, secret, e.UpdatedAt)
	}

	w.Flush()
	return nil
}
