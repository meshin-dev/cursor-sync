package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"cursor-sync/internal/auth"
	"cursor-sync/internal/github"
	"cursor-sync/internal/logger"
)

// Repository represents a Git repository
type Repository struct {
	repo       *git.Repository
	remoteName string
	branch     string
	localPath  string
	auth       *auth.GitHubAuth
	owner      string
	repoName   string
}

// New creates a new Git repository instance
func New(localPath, remoteName, branch, repoURL string) (*Repository, error) {
	// Initialize GitHub authentication
	githubAuth, err := auth.NewGitHubAuth()
	if err != nil {
		return nil, fmt.Errorf("GitHub authentication failed: %w", err)
	}

	// Parse repository owner and name from URL
	owner, repoName, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	return &Repository{
		localPath:  localPath,
		remoteName: remoteName,
		branch:     branch,
		auth:       githubAuth,
		owner:      owner,
		repoName:   repoName,
	}, nil
}

// Clone clones a remote repository using GitHub token authentication
func (r *Repository) Clone(remoteURL string) error {
	logger.Info("Cloning repository from %s to %s", remoteURL, r.localPath)

	// Remove existing directory if it exists
	if _, err := os.Stat(r.localPath); err == nil {
		if err := os.RemoveAll(r.localPath); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Create parent directory
	if err := os.MkdirAll(r.localPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Use token authentication
	auth := &http.BasicAuth{
		Username: "token", // GitHub uses 'token' as username for PAT auth
		Password: r.auth.GetToken(),
	}

	// Try to clone repository with authentication
	repo, err := git.PlainClone(r.localPath, false, &git.CloneOptions{
		URL:           remoteURL,
		Auth:          auth,
		ReferenceName: plumbing.NewBranchReferenceName(r.branch),
		SingleBranch:  true,
		Depth:         1,
	})

	if err != nil {
		// Check if error is due to empty repository (common with new GitHub repos)
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "remote repository is empty") ||
			strings.Contains(errStr, "reference not found") ||
			strings.Contains(errStr, "couldn't find remote ref") {

			logger.Info("🔄 Remote repository is empty - initializing with first commit...")
			return r.initializeEmptyRepository(remoteURL, auth)
		}

		// Check if repository doesn't exist - try to create it automatically
		if strings.Contains(strings.ToLower(err.Error()), "repository not found") ||
			strings.Contains(strings.ToLower(err.Error()), "404") {

			logger.Info("🚀 Repository not found - attempting to create it automatically...")
			return r.createAndCloneRepository(remoteURL, auth)
		}

		return fmt.Errorf("failed to clone repository: %w", err)
	}

	r.repo = repo
	logger.Info("Repository cloned successfully")

	return nil
}

// initializeEmptyRepository initializes a new local repository and pushes initial content to empty remote
func (r *Repository) initializeEmptyRepository(remoteURL string, auth *http.BasicAuth) error {
	logger.Info("🚀 Initializing empty repository with initial commit...")

	// Initialize local git repository
	repo, err := git.PlainInit(r.localPath, false)
	if err != nil {
		return fmt.Errorf("failed to initialize local repository: %w", err)
	}
	r.repo = repo

	// Create initial README.md file
	readmePath := filepath.Join(r.localPath, "README.md")
	readmeContent := fmt.Sprintf(`# Cursor Settings Sync

This repository contains synchronized Cursor IDE settings.

- **Repository**: %s
- **Initialized**: %s
- **Purpose**: Automatic Cursor IDE settings synchronization via cursor-sync

## Files

- `+"`settings.json`"+` - Main Cursor IDE settings
- `+"`keybindings.json`"+` - Custom keyboard shortcuts  
- `+"`snippets/`"+` - Code snippets
- `+"`tasks.json`"+` - VS Code tasks configuration
- `+"`launch.json`"+` - Debug launch configurations
- And more...

> **Note**: This repository is managed automatically by cursor-sync. 
> Manual changes may be overwritten during synchronization.

## Security

🔒 **This repository should be PRIVATE** to protect your sensitive settings and configurations.
`, remoteURL, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	// Add README.md to repository
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := worktree.Add("README.md"); err != nil {
		return fmt.Errorf("failed to add README.md: %w", err)
	}

	// Create initial commit
	commit, err := worktree.Commit("🎉 Initialize cursor-sync settings repository", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "cursor-sync",
			Email: "cursor-sync@localhost",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create initial commit: %w", err)
	}

	logger.Info("✅ Created initial commit: %s", commit.String()[:8])

	// Add remote origin
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: r.remoteName,
		URLs: []string{remoteURL},
	})
	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Push to remote repository (creates main branch on GitHub)
	logger.Info("📤 Pushing initial commit to remote repository...")
	err = repo.Push(&git.PushOptions{
		RemoteName: r.remoteName,
		Auth:       auth,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", r.branch, r.branch))},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push initial commit: %w", err)
	}

	logger.Info("🎉 Empty repository initialized successfully with main branch!")
	logger.Info("📍 Repository is now ready for cursor-sync operations")

	return nil
}

// createAndCloneRepository creates a new repository on GitHub and then clones it
func (r *Repository) createAndCloneRepository(remoteURL string, auth *http.BasicAuth) error {
	logger.Info("🔧 Creating new repository on GitHub...")

	// Create GitHub API client
	githubAPI, err := github.New()
	if err != nil {
		return fmt.Errorf("failed to create GitHub API client: %w", err)
	}

	// Parse owner and repo name from URL
	owner, repoName, err := parseGitHubURL(remoteURL)
	if err != nil {
		return fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Check if repository already exists (in case it was created by another process)
	exists, err := githubAPI.RepositoryExists(owner, repoName)
	if err != nil {
		logger.Warn("Failed to check repository existence: %v", err)
	} else if exists {
		logger.Info("✅ Repository already exists, proceeding with clone...")
		return r.retryCloneWithBackoff(remoteURL, auth)
	}

	// Create repository description
	description := fmt.Sprintf("Cursor IDE settings sync repository - managed by cursor-sync")

	// Create the repository
	repo, err := githubAPI.CreateRepository(owner, repoName, description)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	logger.Info("✅ Repository created successfully: %s", repo.HTMLURL)
	logger.Info("🔒 Repository is PRIVATE for security")

	// Wait for repository to be ready (GitHub sometimes takes a few seconds)
	if err := githubAPI.WaitForRepositoryReady(owner, repoName, 10*time.Second); err != nil {
		logger.Warn("Repository not ready after waiting: %v", err)
		logger.Info("🔄 Proceeding anyway - will retry clone with backoff...")
	}

	// Retry cloning with exponential backoff
	return r.retryCloneWithBackoff(remoteURL, auth)
}

// retryCloneWithBackoff retries cloning with exponential backoff
func (r *Repository) retryCloneWithBackoff(remoteURL string, auth *http.BasicAuth) error {
	maxRetries := 5
	baseDelay := 2 * time.Second
	maxDelay := 10 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info("🔄 Attempt %d/%d: Trying to clone repository...", attempt, maxRetries)

		// Try to clone
		repo, err := git.PlainClone(r.localPath, false, &git.CloneOptions{
			URL:           remoteURL,
			Auth:          auth,
			ReferenceName: plumbing.NewBranchReferenceName(r.branch),
			SingleBranch:  true,
			Depth:         1,
		})

		if err == nil {
			r.repo = repo
			logger.Info("✅ Repository cloned successfully on attempt %d", attempt)
			return nil
		}

		// Check if it's an empty repository error
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "remote repository is empty") ||
			strings.Contains(errStr, "reference not found") ||
			strings.Contains(errStr, "couldn't find remote ref") {

			logger.Info("🔄 Repository is empty - initializing with first commit...")
			return r.initializeEmptyRepository(remoteURL, auth)
		}

		// If this is the last attempt, return the error
		if attempt == maxRetries {
			return fmt.Errorf("failed to clone repository after %d attempts: %w", maxRetries, err)
		}

		// Calculate delay with exponential backoff
		delay := time.Duration(attempt) * baseDelay
		if delay > maxDelay {
			delay = maxDelay
		}

		logger.Info("⏳ Repository not ready yet, waiting %v before retry...", delay)
		time.Sleep(delay)
	}

	return fmt.Errorf("failed to clone repository after %d attempts", maxRetries)
}

// parseGitHubURL parses a GitHub repository URL and extracts owner and repo name
func parseGitHubURL(repoURL string) (owner, repo string, err error) {
	// This function should be same as in privacy package
	// Handle various GitHub URL formats:
	// https://github.com/owner/repo.git
	// https://github.com/owner/repo
	// git@github.com:owner/repo.git

	url := strings.TrimSpace(repoURL)
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimSuffix(url, ".git")

	// Replace : with / for SSH format
	url = strings.Replace(url, ":", "/", 1)

	// Remove github.com prefix
	if strings.HasPrefix(url, "github.com/") {
		url = strings.TrimPrefix(url, "github.com/")
	}

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}

	return parts[0], parts[1], nil
}

// Open opens an existing repository
func (r *Repository) Open() error {
	repo, err := git.PlainOpen(r.localPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	r.repo = repo
	return nil
}

// Pull pulls changes from the remote repository using GitHub token
func (r *Repository) Pull() error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	logger.Debug("Pulling changes from remote")

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Use token authentication for pull
	auth := &http.BasicAuth{
		Username: "token",
		Password: r.auth.GetToken(),
	}

	err = worktree.Pull(&git.PullOptions{
		RemoteName:    r.remoteName,
		ReferenceName: plumbing.NewBranchReferenceName(r.branch),
		Auth:          auth,
		Depth:         1, // Shallow pull - only fetch latest commit
	})

	if err == git.NoErrAlreadyUpToDate {
		logger.Debug("Repository already up to date")
		return nil
	}

	// Handle specific Git errors more gracefully
	if err != nil {
		errStr := err.Error()

		// Check for common conflict scenarios
		if strings.Contains(errStr, "non-fast-forward") ||
			strings.Contains(errStr, "rejected") ||
			strings.Contains(errStr, "cannot lock ref") {
			logger.Debug("Pull conflict detected: %v", err)
			return fmt.Errorf("pull conflict: %w", err)
		}

		// Check for network or authentication issues
		if strings.Contains(errStr, "authentication") ||
			strings.Contains(errStr, "network") ||
			strings.Contains(errStr, "timeout") {
			logger.Debug("Network/authentication issue during pull: %v", err)
			return fmt.Errorf("network/authentication error: %w", err)
		}

		return fmt.Errorf("failed to pull changes: %w", err)
	}

	logger.Info("Pulled changes from remote")
	return nil
}

// PullWithConflictResolution performs a pull with robust conflict resolution
func (r *Repository) PullWithConflictResolution(strategy string) error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	logger.Debug("Pulling changes from remote with conflict resolution")

	// First, try normal pull
	if err := r.Pull(); err == nil {
		return nil // Success
	}

	// If normal pull failed, try conflict resolution based on strategy
	logger.Info("Normal pull failed, attempting conflict resolution with strategy: %s", strategy)

	switch strategy {
	case "newer":
		return r.pullWithNewerStrategy()
	case "local":
		return r.pullWithLocalStrategy()
	case "remote":
		return r.pullWithRemoteStrategy()
	default:
		return fmt.Errorf("unknown conflict resolution strategy: %s", strategy)
	}
}

// pullWithNewerStrategy resolves conflicts by comparing timestamps
func (r *Repository) pullWithNewerStrategy() error {
	localTime, err := r.GetLastCommitTime()
	if err != nil {
		logger.Warn("Failed to get local commit time, using remote strategy: %v", err)
		return r.pullWithRemoteStrategy()
	}

	remoteTime, err := r.GetRemoteLastCommitTime()
	if err != nil {
		logger.Warn("Failed to get remote commit time, using local strategy: %v", err)
		return r.pullWithLocalStrategy()
	}

	if localTime.After(remoteTime) {
		logger.Info("Local changes are newer, keeping local version")
		return r.pullWithLocalStrategy()
	} else {
		logger.Info("Remote changes are newer, keeping remote version")
		return r.pullWithRemoteStrategy()
	}
}

// pullWithLocalStrategy keeps local changes and discards remote conflicts
func (r *Repository) pullWithLocalStrategy() error {
	logger.Info("Using local strategy - keeping local changes")

	// Reset to local HEAD to discard any partial merge state
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Reset to HEAD to clean up any merge state
	if err := worktree.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: plumbing.ZeroHash, // Reset to HEAD
	}); err != nil {
		return fmt.Errorf("failed to reset worktree: %w", err)
	}

	logger.Info("Successfully kept local changes")
	return nil
}

// pullWithRemoteStrategy discards local changes and accepts remote
func (r *Repository) pullWithRemoteStrategy() error {
	logger.Info("Using remote strategy - accepting remote changes")

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Use token authentication
	auth := &http.BasicAuth{
		Username: "token",
		Password: r.auth.GetToken(),
	}

	// Force pull to overwrite local changes
	err = worktree.Pull(&git.PullOptions{
		RemoteName:    r.remoteName,
		ReferenceName: plumbing.NewBranchReferenceName(r.branch),
		Auth:          auth,
		Force:         true, // Force overwrite local changes
		Depth:         1,    // Shallow pull
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to force pull remote changes: %w", err)
	}

	logger.Info("Successfully accepted remote changes")
	return nil
}

// Push pushes changes to the remote repository using GitHub token
func (r *Repository) Push() error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	logger.Debug("Pushing changes to remote")

	// Use token authentication for push
	auth := &http.BasicAuth{
		Username: "token",
		Password: r.auth.GetToken(),
	}

	err := r.repo.Push(&git.PushOptions{
		RemoteName: r.remoteName,
		Auth:       auth,
	})

	if err == git.NoErrAlreadyUpToDate {
		logger.Debug("Remote already up to date")
		return nil
	}

	// Handle specific Git errors more gracefully
	if err != nil {
		errStr := err.Error()

		// Check for common conflict scenarios
		if strings.Contains(errStr, "non-fast-forward") ||
			strings.Contains(errStr, "rejected") ||
			strings.Contains(errStr, "cannot lock ref") ||
			strings.Contains(errStr, "object not found") {
			logger.Debug("Push conflict detected: %v", err)
			return fmt.Errorf("push conflict: %w", err)
		}

		// Check for network or authentication issues
		if strings.Contains(errStr, "authentication") ||
			strings.Contains(errStr, "network") ||
			strings.Contains(errStr, "timeout") {
			logger.Debug("Network/authentication issue during push: %v", err)
			return fmt.Errorf("network/authentication error: %w", err)
		}

		return fmt.Errorf("failed to push changes: %w", err)
	}

	logger.Info("Pushed changes to remote")
	return nil
}

// Add adds files to the staging area
func (r *Repository) Add(pattern string) error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	_, err = worktree.Add(pattern)
	if err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	return nil
}

// Commit commits staged changes
func (r *Repository) Commit(message, authorName, authorEmail string) error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	commit, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	logger.Debug("Created commit: %s", commit.String())
	return nil
}

// HasChanges checks if there are uncommitted changes
func (r *Repository) HasChanges() (bool, error) {
	if r.repo == nil {
		return false, fmt.Errorf("repository not initialized")
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return false, fmt.Errorf("failed to get status: %w", err)
	}

	return !status.IsClean(), nil
}

// GetLastCommitTime returns the timestamp of the last commit
func (r *Repository) GetLastCommitTime() (time.Time, error) {
	if r.repo == nil {
		return time.Time{}, fmt.Errorf("repository not initialized")
	}

	ref, err := r.repo.Head()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.Author.When, nil
}

// GetRemoteLastCommitTime returns the timestamp of the last commit on the remote branch using GitHub API
func (r *Repository) GetRemoteLastCommitTime() (time.Time, error) {
	ctx := context.Background()
	client := r.auth.GetClient()

	// Get the latest commit from the branch using GitHub API
	branch, _, err := client.Repositories.GetBranch(ctx, r.owner, r.repoName, r.branch, 3)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get branch info from GitHub API: %w", err)
	}

	if branch.Commit == nil || branch.Commit.Commit == nil || branch.Commit.Commit.Author == nil {
		return time.Time{}, fmt.Errorf("invalid commit information from GitHub API")
	}

	return branch.Commit.Commit.Author.GetDate().Time, nil
}

// ResolveConflicts resolves merge conflicts based on strategy
func (r *Repository) ResolveConflicts(strategy string) error {
	if r.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	logger.Info("Resolving conflicts using strategy: %s", strategy)

	switch strategy {
	case "newer":
		return r.resolveByTimestamp()
	case "local":
		return r.resolveWithLocal()
	case "remote":
		return r.resolveWithRemote()
	default:
		return fmt.Errorf("unknown conflict resolution strategy: %s", strategy)
	}
}

func (r *Repository) resolveByTimestamp() error {
	localTime, err := r.GetLastCommitTime()
	if err != nil {
		return fmt.Errorf("failed to get local commit time: %w", err)
	}

	remoteTime, err := r.GetRemoteLastCommitTime()
	if err != nil {
		return fmt.Errorf("failed to get remote commit time: %w", err)
	}

	if localTime.After(remoteTime) {
		logger.Info("Local changes are newer, keeping local version")
		return r.resolveWithLocal()
	} else {
		logger.Info("Remote changes are newer, keeping remote version")
		return r.resolveWithRemote()
	}
}

func (r *Repository) resolveWithLocal() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	for file, stat := range status {
		if stat.Staging == git.UpdatedButUnmerged || stat.Worktree == git.UpdatedButUnmerged {
			// Keep local version
			_, err = worktree.Remove(file)
			if err != nil && !strings.Contains(err.Error(), "file does not exist") {
				return fmt.Errorf("failed to remove conflicted file: %w", err)
			}
		}
	}

	return nil
}

func (r *Repository) resolveWithRemote() error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Pull again to accept remote changes
	auth := &http.BasicAuth{
		Username: "token",
		Password: r.auth.GetToken(),
	}

	err = worktree.Pull(&git.PullOptions{
		RemoteName:    r.remoteName,
		ReferenceName: plumbing.NewBranchReferenceName(r.branch),
		Auth:          auth,
		Force:         true,
		Depth:         1, // Shallow pull - only fetch latest commit
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull remote changes: %w", err)
	}

	return nil
}
