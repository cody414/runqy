package queueworker

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Publikey/runqy/config"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitLoader handles loading YAML configs from a Git repository
type GitLoader struct {
	repoURL   string
	branch    string
	path      string
	pat       string // Personal Access Token for HTTPS auth
	localPath string
	repo      *git.Repository
	auth      transport.AuthMethod
}

// NewGitLoader creates a new GitLoader from config
func NewGitLoader(cfg *config.Config) (*GitLoader, error) {
	if cfg.ConfigRepoURL == "" {
		return nil, fmt.Errorf("CONFIG_REPO_URL is required")
	}

	// Create clone directory (defaults to "downloads" in current working directory)
	cloneDir := cfg.ConfigCloneDir
	if cloneDir == "" {
		cloneDir = "downloads"
	}

	// Make absolute path if relative
	if !filepath.IsAbs(cloneDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		cloneDir = filepath.Join(cwd, cloneDir)
	}

	// Create the downloads directory if it doesn't exist
	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create clone directory %s: %w", cloneDir, err)
	}

	// Create a unique subdirectory for this clone
	tmpDir, err := os.MkdirTemp(cloneDir, "queueworker-configs-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory in %s: %w", cloneDir, err)
	}

	loader := &GitLoader{
		repoURL:   cfg.ConfigRepoURL,
		branch:    cfg.ConfigRepoBranch,
		path:      cfg.ConfigRepoPath,
		pat:       cfg.GitHubPAT,
		localPath: tmpDir,
	}

	// Setup PAT authentication for HTTPS
	if err := loader.setupAuth(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}

	return loader, nil
}

func (g *GitLoader) setupAuth() error {
	if g.pat != "" {
		log.Printf("[GIT-LOADER] Using PAT authentication")
		g.auth = &http.BasicAuth{
			Username: "x-access-token", // GitHub accepts any non-empty username with PAT
			Password: g.pat,
		}
	} else {
		log.Printf("[GIT-LOADER] No PAT provided, attempting unauthenticated access (public repos only)")
	}
	return nil
}

// Clone clones the repository to the local temp directory
func (g *GitLoader) Clone() error {
	log.Printf("[GIT-LOADER] Cloning %s (branch: %s)", g.repoURL, g.branch)

	cloneOpts := &git.CloneOptions{
		URL:           g.repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(g.branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          g.auth,
	}

	repo, err := git.PlainClone(g.localPath, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	g.repo = repo
	log.Printf("[GIT-LOADER] Cloned to %s", g.localPath)
	return nil
}

// Pull fetches and pulls the latest changes
// Returns true if there were new changes
func (g *GitLoader) Pull() (bool, error) {
	if g.repo == nil {
		return false, fmt.Errorf("repository not cloned")
	}

	// Get current HEAD
	headBefore, err := g.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get worktree
	worktree, err := g.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Pull latest changes
	err = worktree.Pull(&git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(g.branch),
		SingleBranch:  true,
		Auth:          g.auth,
	})

	if err == git.NoErrAlreadyUpToDate {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to pull: %w", err)
	}

	// Check if HEAD changed
	headAfter, err := g.repo.Head()
	if err != nil {
		return false, fmt.Errorf("failed to get HEAD after pull: %w", err)
	}

	changed := headBefore.Hash() != headAfter.Hash()
	if changed {
		log.Printf("[GIT-LOADER] Updated from %s to %s", headBefore.Hash().String()[:7], headAfter.Hash().String()[:7])
	}

	return changed, nil
}

// GetConfigPath returns the path to YAML config files
func (g *GitLoader) GetConfigPath() string {
	if g.path != "" {
		return filepath.Join(g.localPath, g.path)
	}
	return g.localPath
}

// GetLocalPath returns the local clone directory
func (g *GitLoader) GetLocalPath() string {
	return g.localPath
}

// LoadConfigs loads all YAML configs from the cloned repo
func (g *GitLoader) LoadConfigs() ([]*QueueWorkersYAML, error) {
	configPath := g.GetConfigPath()
	return LoadAll(configPath)
}

// Cleanup removes the temporary clone directory
func (g *GitLoader) Cleanup() {
	if g.localPath != "" {
		log.Printf("[GIT-LOADER] Cleaning up %s", g.localPath)
		os.RemoveAll(g.localPath)
	}
}
