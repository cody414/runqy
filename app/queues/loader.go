package queueworker

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadAll loads all YAML files from the specified directory
func LoadAll(dir string) ([]*QueueWorkersYAML, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("queueworkers directory not found: %s", dir)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Find all YAML files
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob yaml files: %w", err)
	}

	// Also check for .yml files
	ymlFiles, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob yml files: %w", err)
	}
	files = append(files, ymlFiles...)

	if len(files) == 0 {
		return nil, nil // No files found, not an error
	}

	var configs []*QueueWorkersYAML
	for _, file := range files {
		cfg, err := LoadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		if cfg != nil {
			configs = append(configs, cfg)
		}
	}
	return configs, nil
}

// LoadFile loads and validates a single YAML file
func LoadFile(path string) (*QueueWorkersYAML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Skip empty files
	if len(data) == 0 {
		return nil, nil
	}

	var cfg QueueWorkersYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}

	// Skip files that don't have the expected structure
	if cfg.Queues == nil || len(cfg.Queues) == 0 {
		return nil, nil
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks the configuration is valid
func (c *QueueWorkersYAML) Validate() error {
	for queueName, q := range c.Queues {
		if queueName == "" {
			return fmt.Errorf("queue name cannot be empty")
		}

		// If no sub_queues, must have priority
		if len(q.SubQueues) == 0 && q.Priority == 0 {
			return fmt.Errorf("queue '%s': either priority or sub_queues is required", queueName)
		}

		// Validate sub_queues if present
		for i, sq := range q.SubQueues {
			if sq.Name == "" {
				return fmt.Errorf("queue '%s': sub_queues[%d].name is required", queueName, i)
			}
			if sq.Priority == 0 {
				return fmt.Errorf("queue '%s': sub_queues[%d].priority is required", queueName, i)
			}
		}

		// External workers (no provider or provider="worker") require deployment
		if q.Provider == "" || q.Provider == "worker" {
			if q.Deployment == nil {
				return fmt.Errorf("queue '%s': deployment is required for external workers", queueName)
			}
			if q.Deployment.GitURL == "" {
				return fmt.Errorf("queue '%s': deployment.git_url is required", queueName)
			}
			if q.Deployment.StartupCmd == "" {
				return fmt.Errorf("queue '%s': deployment.startup_cmd is required", queueName)
			}
		}
	}
	return nil
}

// ToQueueConfigs converts a YAML queue config to runtime QueueConfig(s)
// If sub_queues are defined, creates multiple configs like "inference_high", "inference_low"
// If no sub_queues, creates a single config with name "queueName_default"
func (q *QueueYAML) ToQueueConfigs(baseName string) []*QueueConfig {
	var configs []*QueueConfig

	// Create deployment config if present
	var deployment *DeploymentConfig
	if q.Deployment != nil {
		deployment = &DeploymentConfig{
			GitURL:             q.Deployment.GitURL,
			Branch:             q.Deployment.Branch,
			CodePath:           q.Deployment.CodePath,
			StartupCmd:         q.Deployment.StartupCmd,
			Mode:               q.Deployment.Mode,
			StartupTimeoutSecs: q.Deployment.StartupTimeoutSecs,
			RedisStorage:       q.Deployment.RedisStorage,
			Vaults:             q.Deployment.Vaults,
			GitToken:           q.Deployment.GitToken,
		}
	}

	if len(q.SubQueues) > 0 {
		// Create a config for each sub-queue
		for _, sq := range q.SubQueues {
			cfg := &QueueConfig{
				Name:       fmt.Sprintf("%s_%s", baseName, sq.Name),
				Priority:   sq.Priority,
				Provider:   q.Provider,
				Deployment: deployment,
			}
			configs = append(configs, cfg)
		}
	} else {
		// No sub-queues, create single config with _default suffix
		cfg := &QueueConfig{
			Name:       fmt.Sprintf("%s_default", baseName),
			Priority:   q.Priority,
			Provider:   q.Provider,
			Deployment: deployment,
		}
		configs = append(configs, cfg)
	}

	return configs
}
