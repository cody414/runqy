package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	HTTPPort string

	// API Authentication
	APIKey string

	// Redis (existing env vars, centralized here)
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisTLS      bool

	// Azure OpenAI
	AzureAPIKey     string
	AzureEndpoint   string
	AzureAPIVersion string
	AzureDeployment string

	// Google Gemini
	GoogleAPIKey string

	// Virtual Worker settings (for API providers like azure, google)
	VirtualWorkerConcurrency int

	// Webhook settings
	WebhookTimeout time.Duration
	WebhookRetries int

	// Queue Workers
	QueueWorkersDir string

	// PostgreSQL
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSL      string

	// GitHub Repository Config
	ConfigRepoURL       string
	ConfigRepoBranch    string
	ConfigRepoPath      string
	ConfigCloneDir      string
	GitHubPAT           string // Personal Access Token for HTTPS auth
	ConfigWatchInterval int
}

// Load creates a Config from environment variables
func Load() *Config {
	return &Config{
		// Server
		HTTPPort: getEnv("PORT", "3000"),

		// API Authentication
		APIKey: os.Getenv("ASYNQ_API_KEY"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6060"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisTLS:      os.Getenv("REDIS_TLS") == "true",

		// Azure OpenAI
		AzureAPIKey:     os.Getenv("AZURE_OPENAI_API_KEY"),
		AzureEndpoint:   os.Getenv("AZURE_OPENAI_ENDPOINT"),
		AzureAPIVersion: getEnv("AZURE_OPENAI_API_VERSION", "2024-02-01"),

		// Google Gemini
		GoogleAPIKey: os.Getenv("GOOGLE_API_KEY"),

		// Virtual Workers (for API providers)
		VirtualWorkerConcurrency: getEnvInt("VIRTUAL_WORKER_CONCURRENCY", 10),

		// Webhook
		WebhookTimeout: time.Duration(getEnvInt("WEBHOOK_TIMEOUT_SECONDS", 30)) * time.Second,
		WebhookRetries: getEnvInt("WEBHOOK_RETRIES", 3),

		// Queue Workers
		QueueWorkersDir: getEnv("QUEUE_WORKERS_DIR", "../queueworkers"),

		// PostgreSQL
		PostgresHost:     getEnv("DATABASE_HOST", "localhost"),
		PostgresPort:     getEnv("DATABASE_PORT", "5432"),
		PostgresUser:     getEnv("DATABASE_USER", "postgres"),
		PostgresPassword: os.Getenv("DATABASE_PASSWORD"),
		PostgresDB:       getEnv("DATABASE_DBNAME", "sdxl_queuing_dev"),
		PostgresSSL:      getEnv("DATABASE_SSL", "disable"),

		// GitHub Repository Config
		ConfigRepoURL:       os.Getenv("CONFIG_REPO_URL"),
		ConfigRepoBranch:    getEnv("CONFIG_REPO_BRANCH", "main"),
		ConfigRepoPath:      getEnv("CONFIG_REPO_PATH", "deployment"),
		ConfigCloneDir:      getEnv("CONFIG_CLONE_DIR", "downloads"),
		GitHubPAT:           os.Getenv("GITHUB_PAT"),
		ConfigWatchInterval: getEnvInt("CONFIG_WATCH_INTERVAL", 60),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
