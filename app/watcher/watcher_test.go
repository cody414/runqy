package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfigWatcher_DetectsYAMLChanges(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Track reload calls
	var reloadCount atomic.Int32
	mockReload := func(ctx context.Context) ([]string, []string) {
		reloadCount.Add(1)
		return []string{"test-queue"}, nil
	}

	// Create watcher
	cw, err := NewConfigWatcher(tmpDir, mockReload)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := cw.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer cw.Stop()

	// Create a YAML file
	yamlFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(yamlFile, []byte("queues: {}"), 0644); err != nil {
		t.Fatalf("Failed to write yaml file: %v", err)
	}

	// Wait for debounce (500ms) + processing time
	time.Sleep(1 * time.Second)

	count := reloadCount.Load()
	if count == 0 {
		t.Error("Expected reload to be triggered on file create, got 0 calls")
	}
	t.Logf("Reload triggered %d time(s) after file create", count)

	// Modify the file
	reloadCount.Store(0)
	if err := os.WriteFile(yamlFile, []byte("queues:\n  test: {}"), 0644); err != nil {
		t.Fatalf("Failed to modify yaml file: %v", err)
	}

	time.Sleep(1 * time.Second)

	count = reloadCount.Load()
	if count == 0 {
		t.Error("Expected reload to be triggered on file modify, got 0 calls")
	}
	t.Logf("Reload triggered %d time(s) after file modify", count)
}

func TestConfigWatcher_IgnoresNonYAMLFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	var reloadCount atomic.Int32
	mockReload := func(ctx context.Context) ([]string, []string) {
		reloadCount.Add(1)
		return nil, nil
	}

	cw, err := NewConfigWatcher(tmpDir, mockReload)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := cw.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}
	defer cw.Stop()

	// Create a non-YAML file
	txtFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(txtFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to write txt file: %v", err)
	}

	time.Sleep(1 * time.Second)

	if reloadCount.Load() != 0 {
		t.Errorf("Expected no reload for .txt file, got %d calls", reloadCount.Load())
	}
	t.Log("Correctly ignored non-YAML file")
}
