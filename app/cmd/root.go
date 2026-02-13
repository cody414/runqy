package cmd

import (
	"fmt"
	"os"

	"github.com/Publikey/runqy/config"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	// Global config loaded from environment
	cfg *config.Config

	// Remote server mode flags
	serverURL string
	apiKey    string

	// Redis URI flag
	redisURI string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "runqy",
	Short:   "Distributed task queue system with server-driven bootstrap",
	Version: Version,
	Long: `runqy is a distributed task queue system built on asynq.

It provides a central server for worker registration, queue management,
REST API, and monitoring dashboard. Workers receive all configuration
(Redis credentials, code deployment specs, task routing) from the server at startup.

Commands:
  serve       Start the HTTP server (default)
  queue       Queue management commands
  task        Task management commands
  worker      Worker management commands
  config      Configuration management commands`,
	// Default to serve if no subcommand is provided
	Run: func(cmd *cobra.Command, args []string) {
		serveCmd.Run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set custom version template to include git commit and build date
	rootCmd.SetVersionTemplate(fmt.Sprintf("runqy version %s (commit: %s, built: %s)\n", Version, GitCommit, BuildDate))

	// Global flags
	rootCmd.PersistentFlags().StringVar(&redisURI, "redis-uri", "", "Redis URI (e.g., redis://:password@host:port)")

	// Remote server mode flags
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "", "Remote server URL (e.g., https://runqy.example.com:3000)")
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication (or set RUNQY_API_KEY env var)")
}

// IsRemoteMode returns true if CLI should use remote server API
func IsRemoteMode() bool {
	return serverURL != ""
}

// GetServerURL returns the remote server URL
func GetServerURL() string {
	return serverURL
}

// GetAPIKey returns the API key (from flag or environment)
func GetAPIKey() string {
	if apiKey != "" {
		return apiKey
	}
	return cfg.APIKey
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load .env file if present (for local development)
	// Looks for .env.secret in parent dir, then .env in current dir
	// Use Overload to ensure .env.secret values take priority over existing env vars
	if err := godotenv.Overload("../.env.secret"); err != nil {
		_ = godotenv.Overload() // Try .env as fallback
	}

	// Load configuration from environment
	cfg = config.Load()

	// Override config from CLI flags (CLI > env > defaults)
	if apiKey != "" {
		cfg.APIKey = apiKey
	}

	if redisURI != "" {
		if err := cfg.ParseRedisURI(redisURI); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid --redis-uri: %v\n", err)
		}
	}

	// Load saved credentials if no --server flag provided
	// Priority: CLI flags > env vars > saved credentials
	if serverURL == "" {
		// Check environment variable first
		if envServer := os.Getenv("RUNQY_SERVER"); envServer != "" {
			serverURL = envServer
		} else {
			// Load from saved credentials
			if creds, err := GetCurrentCredentials(); err == nil && creds != nil {
				serverURL = creds.URL
				// Only use saved API key if not already set via flag or env
				if apiKey == "" && os.Getenv("RUNQY_API_KEY") == "" {
					apiKey = creds.APIKey
					cfg.APIKey = creds.APIKey
				}
			}
		}
	}

	// Warn when running CLI commands without a server configured (local mode)
	if serverURL == "" && len(os.Args) > 1 {
		cmd := os.Args[1]
		if cmd != "serve" && cmd != "login" && cmd != "version" && cmd != "help" {
			fmt.Fprintln(os.Stderr, "Warning: No server configured. Using local database. Run 'runqy login' for remote mode.")
		}
	}
}

// GetConfig returns the global configuration
func GetConfig() *config.Config {
	return cfg
}
