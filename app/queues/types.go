package queueworker

// QueueWorkersYAML is the root config structure
type QueueWorkersYAML struct {
	Queues map[string]QueueYAML `yaml:"queues"`
}

// QueueYAML represents a single queue configuration in YAML
// Supports two formats:
// 1. With sub_queues: creates multiple queues like "inference_high", "inference_low"
// 2. Without sub_queues: uses priority field directly, creates "name_default"
type QueueYAML struct {
	Priority   int             `yaml:"priority,omitempty"`   // Used when no sub_queues defined
	Provider   string          `yaml:"provider,omitempty"`   // "azure", "google", or empty for external workers
	Deployment *DeploymentYAML `yaml:"deployment,omitempty"` // Required for external workers
	SubQueues  []SubQueueYAML  `yaml:"sub_queues,omitempty"` // Optional list of sub-queues

	// Input and Output describe the schema of the task payload for this queue
	Input  []FieldSchema `yaml:"input,omitempty"`
	Output []FieldSchema `yaml:"output,omitempty"`
}

// FieldSchema describes a single named field and the allowed types
type FieldSchema struct {
	Name string   `yaml:"name"`
	Type []string `yaml:"type"`
}

// SubQueueYAML represents a sub-queue with its own name and priority
type SubQueueYAML struct {
	Name     string `yaml:"name"`
	Priority int    `yaml:"priority"`
}

// DeploymentYAML holds deployment configuration for external workers
type DeploymentYAML struct {
	GitURL             string            `yaml:"git_url"`
	Branch             string            `yaml:"branch"`
	CodePath           string            `yaml:"code_path,omitempty"` // Path within the repo to the code
	StartupCmd         string            `yaml:"startup_cmd"`
	Mode               string            `yaml:"mode,omitempty"` // "long_running" or "one_shot"
	EnvVars            map[string]string `yaml:"env_vars"`
	StartupTimeoutSecs int               `yaml:"startup_timeout_secs"`
	RedisStorage       *bool             `yaml:"redis_storage,omitempty"`
}

// QueueConfig is the runtime representation stored in DB and returned via API
type QueueConfig struct {
	Name       string            `json:"name"`
	Priority   int               `json:"priority"`
	Provider   string            `json:"provider,omitempty"`
	Deployment *DeploymentConfig `json:"deployment,omitempty"`
	CreatedAt  int64             `json:"created_at"`
	UpdatedAt  int64             `json:"updated_at"`
}

// DeploymentConfig is the runtime deployment configuration
type DeploymentConfig struct {
	GitURL             string            `json:"git_url"`
	Branch             string            `json:"branch"`
	CodePath           string            `json:"code_path,omitempty"`
	StartupCmd         string            `json:"startup_cmd"`
	Mode               string            `json:"mode,omitempty"`
	EnvVars            map[string]string `json:"env_vars"`
	StartupTimeoutSecs int               `json:"startup_timeout_secs"`
	RedisStorage       *bool             `json:"redis_storage,omitempty"`
}

// QueueSummary is a lightweight version for listing queues
type QueueSummary struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Provider string `json:"provider,omitempty"`
}
