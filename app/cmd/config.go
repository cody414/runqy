package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Publikey/runqy/models"
	queueworker "github.com/Publikey/runqy/queues"
	"github.com/spf13/cobra"
)

var (
	configListDir string
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long: `Configuration management commands for listing, reloading, and validating queue configs.

Queue configurations are stored in PostgreSQL and loaded from YAML files.

Examples:
  runqy config list
  runqy config reload --dir ./deployment
  runqy config validate --dir ./deployment

Remote mode:
  runqy --server https://runqy.example.com -k API_KEY config list
  runqy --server https://runqy.example.com -k API_KEY config reload`,
}

// configListCmd lists all queue configurations from DB
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all queue configurations from database",
	Long: `List all queue configurations stored in PostgreSQL.

Shows queue name, priority, provider, and deployment info.`,
	RunE: runConfigList,
}

// configReloadCmd reloads configs from YAML files
var configReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload configurations from YAML files",
	Long: `Reload all queue configurations from YAML files into the database.

This is the same operation as calling POST /workers/reload API.

Note: In remote mode, this triggers reload on the server (--dir is ignored).`,
	RunE: runConfigReload,
}

// configValidateCmd validates YAML files without loading
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate YAML configuration files",
	Long: `Validate all YAML files in the specified directory without loading them into the database.

Useful for checking configuration syntax before deployment.

Note: This command is local-only and cannot be used in remote mode.`,
	RunE: runConfigValidate,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configReloadCmd)
	configCmd.AddCommand(configValidateCmd)

	// Reload/Validate flags
	configReloadCmd.Flags().StringVar(&configListDir, "dir", "", "Config directory (defaults to QUEUE_WORKERS_DIR)")
	configValidateCmd.Flags().StringVar(&configListDir, "dir", "", "Config directory (defaults to QUEUE_WORKERS_DIR)")
}

func runConfigList(cmd *cobra.Command, args []string) error {
	// Remote mode: use API client
	if IsRemoteMode() {
		return runConfigListRemote()
	}

	// Local mode: direct DB access
	cfg := GetConfig()

	// Build PostgreSQL connection
	pgDB, err := models.BuildPostgresDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer pgDB.Close()

	redisAddr, err := models.BuildRedisConns()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer redisAddr.RDB.Close()

	store := queueworker.NewStore(pgDB, redisAddr.RDB)
	ctx := context.Background()

	queues, err := store.ListQueues(ctx)
	if err != nil {
		return fmt.Errorf("failed to list queues: %w", err)
	}

	if len(queues) == 0 {
		fmt.Println("No queue configurations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPRIORITY\tPROVIDER\tMODE\tGIT_URL")

	for _, queueName := range queues {
		queueCfg, err := store.Get(ctx, queueName)
		if err != nil || queueCfg == nil {
			fmt.Fprintf(w, "%s\t-\t-\t-\t-\n", queueName)
			continue
		}

		provider := queueCfg.Provider
		if provider == "" {
			provider = "worker"
		}

		mode := "-"
		gitURL := "-"
		if queueCfg.Deployment != nil {
			mode = queueCfg.Deployment.Mode
			if mode == "" {
				mode = "long_running"
			}
			gitURL = truncate(queueCfg.Deployment.GitURL, 50)
		}

		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
			queueCfg.Name,
			queueCfg.Priority,
			provider,
			mode,
			gitURL,
		)
	}

	w.Flush()
	return nil
}

func runConfigListRemote() error {
	apiClient := NewAPIClient()

	configs, err := apiClient.ListConfigs()
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		fmt.Println("No queue configurations found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPRIORITY\tPROVIDER")

	for _, cfg := range configs {
		provider := cfg.Provider
		if provider == "" {
			provider = "worker"
		}

		fmt.Fprintf(w, "%s\t%d\t%s\n",
			cfg.Name,
			cfg.Priority,
			provider,
		)
	}

	w.Flush()
	return nil
}

func runConfigReload(cmd *cobra.Command, args []string) error {
	// Remote mode: use API client
	if IsRemoteMode() {
		return runConfigReloadRemote()
	}

	// Local mode: direct DB access
	cfg := GetConfig()

	// Override config dir if specified
	dir := cfg.QueueWorkersDir
	if configListDir != "" {
		dir = configListDir
	}

	// Build PostgreSQL connection
	pgDB, err := models.BuildPostgresDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer pgDB.Close()

	redisAddr, err := models.BuildRedisConns()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer redisAddr.RDB.Close()

	store := queueworker.NewStore(pgDB, redisAddr.RDB)
	ctx := context.Background()

	// Load YAML files
	yamls, err := queueworker.LoadAll(dir)
	if err != nil {
		return fmt.Errorf("failed to load YAML files from %s: %w", dir, err)
	}

	if len(yamls) == 0 {
		fmt.Println("No YAML configuration files found.")
		return nil
	}

	var reloaded []string
	var errors []string

	for _, y := range yamls {
		for queueName, queueCfg := range y.Queues {
			configs := queueCfg.ToQueueConfigs(queueName)

			for _, qc := range configs {
				if err := store.Save(ctx, qc); err != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", qc.Name, err))
					continue
				}
				reloaded = append(reloaded, qc.Name)
			}
		}
	}

	// Register queues in asynq
	if len(reloaded) > 0 {
		if err := store.RegisterAsynqQueues(ctx, reloaded); err != nil {
			fmt.Printf("Warning: failed to register asynq queues: %v\n", err)
		}
	}

	fmt.Printf("Reloaded %d queue configuration(s):\n", len(reloaded))
	for _, name := range reloaded {
		fmt.Printf("  - %s\n", name)
	}

	if len(errors) > 0 {
		fmt.Println()
		fmt.Printf("Errors (%d):\n", len(errors))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

func runConfigReloadRemote() error {
	apiClient := NewAPIClient()

	resp, err := apiClient.ReloadConfigs()
	if err != nil {
		return err
	}

	fmt.Printf("Reloaded %d queue configuration(s):\n", len(resp.Reloaded))
	for _, name := range resp.Reloaded {
		fmt.Printf("  - %s\n", name)
	}

	if len(resp.Errors) > 0 {
		fmt.Println()
		fmt.Printf("Errors (%d):\n", len(resp.Errors))
		for _, e := range resp.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Validate is local-only
	if IsRemoteMode() {
		return fmt.Errorf("config validate is local-only and cannot be used in remote mode")
	}

	cfg := GetConfig()

	// Override config dir if specified
	dir := cfg.QueueWorkersDir
	if configListDir != "" {
		dir = configListDir
	}

	// Load and validate YAML files
	yamls, err := queueworker.LoadAll(dir)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if len(yamls) == 0 {
		fmt.Println("No YAML configuration files found.")
		return nil
	}

	fmt.Printf("Validated %d YAML file(s) from %s\n", len(yamls), dir)

	totalQueues := 0
	for _, y := range yamls {
		for queueName, queueCfg := range y.Queues {
			configs := queueCfg.ToQueueConfigs(queueName)
			for _, qc := range configs {
				fmt.Printf("  - %s (priority=%d)\n", qc.Name, qc.Priority)
				totalQueues++
			}
		}
	}

	fmt.Printf("\nTotal: %d queue configuration(s)\n", totalQueues)
	fmt.Println("Validation successful!")

	return nil
}
