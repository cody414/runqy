package watcher

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ReloadFunc is the function signature for config reload operations
type ReloadFunc func(ctx context.Context) (reloaded []string, errors []string)

// ConfigWatcher watches for YAML file changes and triggers reloads
type ConfigWatcher struct {
	watcher    *fsnotify.Watcher
	configDir  string
	reloadFunc ReloadFunc

	debounce time.Duration
	mu       sync.Mutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewConfigWatcher creates a new file watcher for the specified directory
func NewConfigWatcher(configDir string, reloadFunc ReloadFunc) (*ConfigWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &ConfigWatcher{
		watcher:    w,
		configDir:  configDir,
		reloadFunc: reloadFunc,
		debounce:   500 * time.Millisecond,
		stopCh:     make(chan struct{}),
	}, nil
}

// Start begins watching the config directory for changes
func (cw *ConfigWatcher) Start() error {
	if err := cw.watcher.Add(cw.configDir); err != nil {
		return err
	}

	log.Printf("[WATCHER] Watching for changes in: %s", cw.configDir)

	cw.wg.Add(1)
	go cw.run()

	return nil
}

// Stop gracefully stops the watcher
func (cw *ConfigWatcher) Stop() {
	close(cw.stopCh)
	cw.watcher.Close()
	cw.wg.Wait()
	log.Println("[WATCHER] Stopped")
}

func (cw *ConfigWatcher) run() {
	defer cw.wg.Done()

	var debounceTimer *time.Timer

	for {
		select {
		case <-cw.stopCh:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Only process YAML file changes
			if !cw.isYAMLFile(event.Name) {
				continue
			}

			// Only handle Create, Write, Remove operations
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
				continue
			}

			log.Printf("[WATCHER] Detected change: %s (%s)", filepath.Base(event.Name), event.Op)

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(cw.debounce, cw.triggerReload)

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[WATCHER] Error: %v", err)
		}
	}
}

func (cw *ConfigWatcher) isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

func (cw *ConfigWatcher) triggerReload() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	ctx := context.Background()
	reloaded, errors := cw.reloadFunc(ctx)

	if len(errors) > 0 {
		for _, e := range errors {
			log.Printf("[WATCHER] Reload error: %s", e)
		}
	}

	if len(reloaded) > 0 {
		log.Printf("[WATCHER] Reloaded configurations: %v", reloaded)
	} else if len(errors) == 0 {
		log.Println("[WATCHER] No configurations changed")
	}
}

// GitPullFunc is the function signature for git pull operations
type GitPullFunc func() (changed bool, err error)

// GitWatcher polls a git repository for changes
type GitWatcher struct {
	pullFunc   GitPullFunc
	reloadFunc ReloadFunc
	interval   time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewGitWatcher creates a new git polling watcher
func NewGitWatcher(pullFunc GitPullFunc, reloadFunc ReloadFunc, interval time.Duration) *GitWatcher {
	return &GitWatcher{
		pullFunc:   pullFunc,
		reloadFunc: reloadFunc,
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

// Start begins polling the git repository for changes
func (gw *GitWatcher) Start() error {
	log.Printf("[GIT-WATCHER] Polling for changes every %v", gw.interval)

	gw.wg.Add(1)
	go gw.run()

	return nil
}

// Stop gracefully stops the git watcher
func (gw *GitWatcher) Stop() {
	close(gw.stopCh)
	gw.wg.Wait()
	log.Println("[GIT-WATCHER] Stopped")
}

func (gw *GitWatcher) run() {
	defer gw.wg.Done()

	ticker := time.NewTicker(gw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-gw.stopCh:
			return
		case <-ticker.C:
			gw.checkForChanges()
		}
	}
}

func (gw *GitWatcher) checkForChanges() {
	changed, err := gw.pullFunc()
	if err != nil {
		log.Printf("[GIT-WATCHER] Pull error: %v", err)
		return
	}

	if !changed {
		return
	}

	log.Println("[GIT-WATCHER] Changes detected, reloading configurations...")

	ctx := context.Background()
	reloaded, errors := gw.reloadFunc(ctx)

	if len(errors) > 0 {
		for _, e := range errors {
			log.Printf("[GIT-WATCHER] Reload error: %s", e)
		}
	}

	if len(reloaded) > 0 {
		log.Printf("[GIT-WATCHER] Reloaded configurations: %v", reloaded)
	}
}
