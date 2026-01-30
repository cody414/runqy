package queueworker

import (
	"fmt"
	"strings"
)

// Queue naming constants
const (
	// SubQueueSeparator is the character used to separate parent queue from sub-queue
	// e.g., "inference.high" where "inference" is parent and "high" is sub-queue
	SubQueueSeparator = "."

	// DefaultSubQueueName is the default sub-queue name when none is specified
	DefaultSubQueueName = "default"
)

// ValidateQueueName checks that a queue name does not contain the separator character.
// This applies to both parent queue names and sub-queue names.
func ValidateQueueName(name string) error {
	if strings.Contains(name, SubQueueSeparator) {
		return fmt.Errorf("queue name '%s' cannot contain '%s' (reserved as separator)", name, SubQueueSeparator)
	}
	return nil
}

// BuildFullQueueName combines a parent queue name and sub-queue name with the separator.
// e.g., BuildFullQueueName("inference", "high") returns "inference.high"
func BuildFullQueueName(parent, subQueue string) string {
	return parent + SubQueueSeparator + subQueue
}

// ParseQueueName splits a full queue name into parent and sub-queue parts.
// Returns (parent, subQueue, hasSubQueue).
// e.g., ParseQueueName("inference.high") returns ("inference", "high", true)
// e.g., ParseQueueName("simple") returns ("simple", "", false)
func ParseQueueName(fullName string) (parent, subQueue string, hasSubQueue bool) {
	if idx := strings.Index(fullName, SubQueueSeparator); idx > 0 {
		return fullName[:idx], fullName[idx+1:], true
	}
	return fullName, "", false
}

// HasSubQueue checks if a queue name includes a sub-queue part.
// e.g., "inference.high" returns true, "inference" returns false
func HasSubQueue(name string) bool {
	return strings.Contains(name, SubQueueSeparator)
}

// NormalizeQueueName ensures a queue name has a sub-queue part.
// If the name doesn't contain a separator, appends ".default".
// e.g., "inference" becomes "inference.default", "inference.high" stays unchanged
func NormalizeQueueName(name string) string {
	if !HasSubQueue(name) {
		return BuildFullQueueName(name, DefaultSubQueueName)
	}
	return name
}

// QueueWorkersYAML is the root config structure
type QueueWorkersYAML struct {
	Queues map[string]QueueYAML `yaml:"queues"`
}

// QueueYAML represents a single queue configuration in YAML
// Supports two formats:
// 1. With sub_queues: creates multiple queues like "inference.high", "inference.low"
// 2. Without sub_queues: uses priority field directly, creates "name.default"
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
	GitURL             string   `yaml:"git_url"`
	Branch             string   `yaml:"branch"`
	CodePath           string   `yaml:"code_path,omitempty"` // Path within the repo to the code
	StartupCmd         string   `yaml:"startup_cmd"`
	Mode               string   `yaml:"mode,omitempty"` // "long_running" or "one_shot"
	StartupTimeoutSecs int      `yaml:"startup_timeout_secs"`
	RedisStorage       *bool    `yaml:"redis_storage,omitempty"`
	Vaults             []string `yaml:"vaults,omitempty"`    // List of vault names to inject as env vars
	GitToken           string   `yaml:"git_token,omitempty"` // Vault reference for git auth: "vault://vault-name/key"
}

// Queue represents a parent queue with deployment configuration (stored in 'queues' table)
type Queue struct {
	ID           int               `json:"id" db:"id"`
	Name         string            `json:"name" db:"name"`
	Provider     string            `json:"provider,omitempty" db:"provider"`
	Deployment   *DeploymentConfig `json:"deployment,omitempty"`
	InputSchema  []FieldSchema     `json:"input_schema,omitempty"`
	OutputSchema []FieldSchema     `json:"output_schema,omitempty"`
	Description  string            `json:"description,omitempty" db:"description"`
	Enabled      bool              `json:"enabled" db:"enabled"`
	CreatedAt    int64             `json:"created_at" db:"created_at"`
	UpdatedAt    int64             `json:"updated_at" db:"updated_at"`
}

// SubQueue represents a sub-queue with priority (stored in 'sub_queues' table, references parent Queue)
type SubQueue struct {
	ID        int    `json:"id" db:"id"`
	QueueID   int    `json:"queue_id" db:"queue_id"`
	Name      string `json:"name" db:"name"` // Just "high", "low", "default"
	Priority  int    `json:"priority" db:"priority"`
	Enabled   bool   `json:"enabled" db:"enabled"`
	CreatedAt int64  `json:"created_at" db:"created_at"`
	UpdatedAt int64  `json:"updated_at" db:"updated_at"`
}

// QueueWithSubQueues is the combined view for API responses
type QueueWithSubQueues struct {
	Queue
	SubQueues []SubQueue `json:"sub_queues"`
}

// QueueConfig is the runtime representation stored in DB and returned via API
// Deprecated: Use Queue and SubQueue types instead. This type is kept for backward compatibility.
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
	GitURL             string   `json:"git_url"`
	Branch             string   `json:"branch"`
	CodePath           string   `json:"code_path,omitempty"`
	StartupCmd         string   `json:"startup_cmd"`
	Mode               string   `json:"mode,omitempty"`
	StartupTimeoutSecs int      `json:"startup_timeout_secs"`
	RedisStorage       *bool    `json:"redis_storage,omitempty"`
	Vaults             []string `json:"vaults,omitempty"`    // List of vault names to inject as env vars
	GitToken           string   `json:"git_token,omitempty"` // Vault reference for git auth: "vault://vault-name/key"
}

// QueueSummary is a lightweight version for listing queues
type QueueSummary struct {
	Name     string `json:"name"`
	Priority int    `json:"priority"`
	Provider string `json:"provider,omitempty"`
}
