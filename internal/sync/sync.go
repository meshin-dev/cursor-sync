package sync

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"cursor-sync/internal/config"
	"cursor-sync/internal/git"
	"cursor-sync/internal/logger"
	"cursor-sync/internal/privacy"
)

// HashResult represents the result of a hash calculation
type HashResult struct {
	FilePath string
	Hash     string
	Error    error
}

// Syncer handles synchronization between local and remote repositories
type Syncer struct {
	config    *config.Config
	repo      *git.Repository
	lastSync  time.Time
	forcePush bool
	forcePull bool
	// Hash calculation throttling and parallel processing
	hashCache      map[string]string // filepath -> hash
	hashCacheMutex sync.RWMutex
	hashThrottle   time.Duration
	lastHashTime   time.Time
	// Parallel hash calculation
	hashWorkers    int
	hashJobChan    chan string
	hashResultChan chan HashResult
	hashWg         sync.WaitGroup
	hashStopChan   chan struct{}
}

// New creates a new syncer
func New(cfg *config.Config) (*Syncer, error) {
	repo, err := git.New(cfg.Repository.LocalPath, "origin", cfg.Repository.Branch, cfg.Repository.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create git repository: %w", err)
	}

	// Determine number of workers based on CPU cores
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2 // Minimum 2 workers
	}
	if numWorkers > 8 {
		numWorkers = 8 // Maximum 8 workers to prevent overwhelming
	}

	syncer := &Syncer{
		config:         cfg,
		repo:           repo,
		hashCache:      make(map[string]string),
		hashThrottle:   cfg.Sync.HashThrottleDelay,
		hashWorkers:    numWorkers,
		hashJobChan:    make(chan string, numWorkers*2),
		hashResultChan: make(chan HashResult, numWorkers*2),
		hashStopChan:   make(chan struct{}),
	}

	// Start hash calculation workers
	syncer.startHashWorkers()

	return syncer, nil
}

// Initialize initializes the sync repository
func (s *Syncer) Initialize() error {
	logger.Info("Initializing sync repository...")

	// SECURITY CHECK: Verify repository is private before any operations
	if err := s.checkRepositoryPrivacy(); err != nil {
		return fmt.Errorf("repository privacy check failed: %w", err)
	}

	// Check if repository already exists
	if _, err := os.Stat(filepath.Join(s.config.Repository.LocalPath, ".git")); err == nil {
		logger.Debug("Repository already exists, opening...")
		if err := s.repo.Open(); err != nil {
			return err
		}

		// CRITICAL LOGIC: Check if this is a fresh Cursor installation (no .custom.sync marker)
		// If no marker exists, it means local settings have NEVER been synced before
		// In this case, we IGNORE all local files and OVERWRITE them from remote
		if !s.hasCustomSyncMarker() {
			logger.Info("🚨 No custom sync marker found - this indicates local settings have NEVER been synced")
			logger.Info("📥 Performing complete overwrite from remote (ignoring all local files)")

			// Perform initial sync from remote, overwriting all local files
			if err := s.syncFromRemote(); err != nil {
				return err
			}

			// Create the marker file to indicate sync has been performed
			logger.Info("✅ Creating sync marker to indicate local settings are now synced")
			return s.createCustomSyncMarker()
		}

		logger.Debug("Custom sync marker found - local settings have been synced before")
		return nil
	}

	// Clone repository (first time setup)
	logger.Info("Repository doesn't exist locally - cloning from remote")
	if err := s.repo.Clone(s.config.Repository.URL); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// For fresh installation, copy local settings TO repository first
	logger.Info("📤 Performing initial sync from local to remote (fresh installation)")
	if err := s.SyncToRemote(); err != nil {
		return err
	}

	// Create the marker file to indicate custom sync is set up
	logger.Info("✅ Creating sync marker for fresh installation")
	return s.createCustomSyncMarker()
}

// SyncToRemote syncs local changes to the remote repository
func (s *Syncer) SyncToRemote() error {
	logger.Info("Syncing local changes to remote...")

	// Security check before any push operations
	if err := s.checkRepositoryPrivacy(); err != nil {
		return fmt.Errorf("repository privacy check failed: %w", err)
	}

	// Sync deleted files from local to repository
	if err := s.syncDeletedFiles(); err != nil {
		logger.Warn("Failed to sync deleted files: %v", err)
	}

	// Copy Cursor config to repository
	if err := s.copyToRepository(); err != nil {
		return fmt.Errorf("failed to copy config to repository: %w", err)
	}

	// Check if there are changes to commit
	hasChanges, err := s.repo.HasChanges()
	if err != nil {
		return fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges && !s.forcePush {
		logger.Debug("No changes to sync to remote")
		// Even if no changes, ensure marker exists after successful sync
		if !s.hasCustomSyncMarker() {
			logger.Debug("Creating sync marker after successful sync operation")
			return s.createCustomSyncMarker()
		}
		return nil
	}

	// Add all changes
	if err := s.repo.Add("."); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit changes
	hostname, _ := os.Hostname()
	commitMessage := fmt.Sprintf("Auto-sync from %s at %s", hostname, time.Now().Format("2006-01-02 15:04:05"))

	if err := s.repo.Commit(commitMessage, "cursor-sync", "cursor-sync@local"); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// Push changes with robust conflict resolution
	pushSuccess := false
	if err := s.repo.Push(); err != nil {
		logger.Warn("Initial push failed: %v", err)

		// Check if this is a conflict error (local out of sync with remote)
		if strings.Contains(err.Error(), "cannot lock ref") ||
			strings.Contains(err.Error(), "rejected") ||
			strings.Contains(err.Error(), "non-fast-forward") ||
			strings.Contains(err.Error(), "object not found") {

			logger.Warn("Push conflict detected, attempting to resolve...")

			// Try to pull latest changes first to resolve the conflict
			if pullErr := s.repo.Pull(); pullErr != nil {
				logger.Warn("Failed to pull during conflict resolution: %v", pullErr)
			}

			// Try to resolve conflicts using configured strategy
			if resolveErr := s.repo.ResolveConflicts(s.config.Sync.ConflictResolve); resolveErr != nil {
				logger.Warn("Failed to resolve conflicts: %v", resolveErr)
			}

			// Try push again after conflict resolution
			if retryErr := s.repo.Push(); retryErr != nil {
				logger.Warn("Push failed after conflict resolution: %v", retryErr)
			} else {
				pushSuccess = true
				logger.Info("Successfully resolved push conflict")
			}
		} else {
			logger.Warn("Push failed with non-conflict error: %v", err)
		}
	} else {
		pushSuccess = true
	}

	// Even if push failed, we still want to mark the sync as successful
	// because the local changes were committed successfully
	if !pushSuccess {
		logger.Warn("⚠️  Push operation failed, but local changes were committed successfully")
		logger.Warn("⚠️  Changes will be pushed on the next successful sync cycle")
	}

	s.lastSync = time.Now()
	s.forcePush = false

	// IMPORTANT: Create marker file after every successful sync operation
	// This indicates local settings have been synced at least once
	if err := s.createCustomSyncMarker(); err != nil {
		logger.Warn("Failed to create sync marker (non-critical): %v", err)
	}

	if pushSuccess {
		logger.Info("Successfully synced local changes to remote")
	} else {
		logger.Info("⚠️  Sync completed with warnings (push failed but local changes committed)")
	}
	return nil
}

// SyncFromRemote syncs remote changes to local
func (s *Syncer) SyncFromRemote() error {
	logger.Info("Syncing remote changes to local...")

	// Security check before any pull operations
	if err := s.checkRepositoryPrivacy(); err != nil {
		return fmt.Errorf("repository privacy check failed: %w", err)
	}

	// Try to pull changes from remote with robust error handling
	pullSuccess := false
	if err := s.repo.Pull(); err != nil {
		logger.Warn("Initial pull failed: %v", err)

		// Try to resolve conflicts and pull again
		if resolveErr := s.repo.ResolveConflicts(s.config.Sync.ConflictResolve); resolveErr != nil {
			logger.Warn("Failed to resolve conflicts: %v", resolveErr)
		} else {
			// Try pull again after conflict resolution
			if retryErr := s.repo.Pull(); retryErr != nil {
				logger.Warn("Pull failed after conflict resolution: %v", retryErr)
			} else {
				pullSuccess = true
			}
		}
	} else {
		pullSuccess = true
	}

	// Even if pull failed, try to sync what we have locally
	// This ensures sync continues even if remote is problematic
	if !pullSuccess {
		logger.Warn("⚠️  Pull operation failed, but continuing with local sync to ensure data consistency")
	}

	// Sync deleted files from repository to local (if pull was successful)
	if pullSuccess {
		if err := s.syncDeletedFilesFromRemote(); err != nil {
			logger.Warn("Failed to sync deleted files from remote: %v", err)
		}
	}

	// Copy from repository to Cursor config
	if err := s.copyFromRepository(); err != nil {
		return fmt.Errorf("failed to copy from repository: %w", err)
	}

	s.lastSync = time.Now()
	s.forcePull = false

	// IMPORTANT: Create marker file after every successful sync operation
	// This indicates local settings have been synced at least once
	if err := s.createCustomSyncMarker(); err != nil {
		logger.Warn("Failed to create sync marker (non-critical): %v", err)
	}

	if pullSuccess {
		logger.Info("Successfully synced remote changes to local")
	} else {
		logger.Info("⚠️  Sync completed with warnings (pull failed but local sync succeeded)")
	}
	return nil
}

// syncFromRemote is the internal method for initial sync
func (s *Syncer) syncFromRemote() error {
	logger.Info("Performing initial sync from remote...")

	// For initial sync (no .custom.sync marker), we want to:
	// 1. Copy ALL files from remote to local (overwrite local files)
	// 2. BUT NOT delete local files that don't exist in remote
	// This ensures we get the remote settings but don't lose any local files

	// Copy from repository to Cursor config with force overwrite
	if err := s.copyFromRepositoryForce(); err != nil {
		return fmt.Errorf("failed to copy from repository: %w", err)
	}

	logger.Info("Initial sync completed")
	return nil
}

// ForcePush forces the next push operation
func (s *Syncer) ForcePush() {
	s.forcePush = true
}

// ForcePull forces the next pull operation
func (s *Syncer) ForcePull() {
	s.forcePull = true
}

// startHashWorkers starts the parallel hash calculation workers
func (s *Syncer) startHashWorkers() {
	logger.Info("🚀 Starting %d hash calculation workers", s.hashWorkers)
	for i := 0; i < s.hashWorkers; i++ {
		s.hashWg.Add(1)
		go s.hashWorker(i)
	}
	logger.Info("✅ Started %d hash calculation workers", s.hashWorkers)
}

// stopHashWorkers stops all hash calculation workers
func (s *Syncer) stopHashWorkers() {
	close(s.hashStopChan)
	s.hashWg.Wait()
	logger.Debug("Stopped all hash calculation workers")
}

// hashWorker is a worker goroutine that calculates file hashes
func (s *Syncer) hashWorker(workerID int) {
	defer s.hashWg.Done()

	for {
		select {
		case <-s.hashStopChan:
			return
		case filePath := <-s.hashJobChan:
			// Calculate hash with throttling
			hash, err := s.calculateSingleFileHash(filePath)
			s.hashResultChan <- HashResult{
				FilePath: filePath,
				Hash:     hash,
				Error:    err,
			}
		}
	}
}

// calculateSingleFileHash calculates hash for a single file with throttling
func (s *Syncer) calculateSingleFileHash(filePath string) (string, error) {
	// Throttle hash calculations to prevent CPU stress
	timeSinceLastHash := time.Since(s.lastHashTime)
	if timeSinceLastHash < s.hashThrottle {
		sleepTime := s.hashThrottle - timeSinceLastHash
		logger.Debug("Worker throttling hash calculation for %s, sleeping for %v", filepath.Base(filePath), sleepTime)
		time.Sleep(sleepTime)
	}

	// Calculate hash
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash)

	// Update last hash time
	s.hashCacheMutex.Lock()
	s.lastHashTime = time.Now()
	s.hashCacheMutex.Unlock()

	return hashStr, nil
}

// syncDeletedFiles removes files from the repository that no longer exist locally
func (s *Syncer) syncDeletedFiles() error {
	logger.Debug("Syncing deleted files from local to repository...")

	cursorPath := s.config.Cursor.ConfigPath
	userPath := filepath.Join(cursorPath, "User")
	repoPath := s.config.Repository.LocalPath
	repoUserPath := filepath.Join(repoPath, "User")

	var filesRemoved int

	// Walk through the repository and check if files still exist locally
	err := filepath.Walk(repoUserPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from User directory in repository
		relPath, err := filepath.Rel(repoUserPath, path)
		if err != nil {
			return nil
		}

		// Check if this path should be excluded
		if s.shouldExcludePath("User/" + relPath) {
			return nil
		}

		// Check if file exists locally
		localPath := filepath.Join(userPath, relPath)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			// File doesn't exist locally, remove it from repository
			if err := os.Remove(path); err != nil {
				logger.Warn("Failed to remove deleted file from repository: %s", relPath)
				return nil
			}
			filesRemoved++
			logger.Debug("🗑️  Removed deleted file from repository: %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to sync deleted files: %w", err)
	}

	if filesRemoved > 0 {
		logger.Info("🗑️  Synced deletions: %d files removed from repository", filesRemoved)
	} else {
		logger.Debug("🗑️  No files to delete from repository")
	}

	return nil
}

// syncDeletedFilesFromRemote removes files locally that no longer exist in the repository
func (s *Syncer) syncDeletedFilesFromRemote() error {
	logger.Debug("Syncing deleted files from repository to local...")

	cursorPath := s.config.Cursor.ConfigPath
	userPath := filepath.Join(cursorPath, "User")
	repoPath := s.config.Repository.LocalPath
	repoUserPath := filepath.Join(repoPath, "User")

	// Check if User directory exists in repository
	if _, err := os.Stat(repoUserPath); os.IsNotExist(err) {
		logger.Debug("User directory does not exist in repository, skipping deletion sync")
		return nil
	}

	var filesRemoved int

	// Walk through local User directory and check if files still exist in repository
	err := filepath.Walk(userPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from User directory locally
		relPath, err := filepath.Rel(userPath, path)
		if err != nil {
			return nil
		}

		// Check if this path should be excluded
		if s.shouldExcludePath("User/" + relPath) {
			return nil
		}

		// Check if file exists in repository
		repoPath := filepath.Join(repoUserPath, relPath)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			// File doesn't exist in repository, remove it locally
			if err := os.Remove(path); err != nil {
				logger.Warn("Failed to remove deleted file locally: %s", relPath)
				return nil
			}
			filesRemoved++
			logger.Debug("🗑️  Removed deleted file locally: %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to sync deleted files from remote: %w", err)
	}

	if filesRemoved > 0 {
		logger.Info("🗑️  Synced deletions from remote: %d files removed locally", filesRemoved)
	} else {
		logger.Debug("🗑️  No files to delete locally")
	}

	return nil
}

// copyToRepository copies Cursor configuration to the repository
// Uses rsync-like logic to only copy files that have actually changed
// Only targets the User folder
func (s *Syncer) copyToRepository() error {
	logger.Info("🚀 copyToRepository called - starting rsync mode")

	// First, clean up any excluded files from the repository
	if err := s.CleanupExcludedFiles(); err != nil {
		logger.Warn("Failed to cleanup excluded files: %v", err)
	}

	cursorPath := s.config.Cursor.ConfigPath
	userPath := filepath.Join(cursorPath, "User")
	repoPath := s.config.Repository.LocalPath

	// Check if User directory exists
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		return fmt.Errorf("User directory does not exist: %s", userPath)
	}

	var filesCopied, filesSkipped int

	err := filepath.Walk(userPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Get relative path from User directory
		relPath, err := filepath.Rel(userPath, path)
		if err != nil {
			return nil
		}

		// Skip socket files (they can't be read)
		if strings.HasSuffix(relPath, ".sock") {
			logger.Debug("Skipping socket file: %s", relPath)
			return nil
		}

		// Skip if should be excluded
		excludePath := "User/" + relPath
		if s.shouldExcludePath(excludePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := filepath.Join(repoPath, "User", relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, info.Mode())
		}

		// For files, check if we need to copy
		if s.shouldCopyFile(path, destPath, info) {
			if err := s.copyFile(path, destPath); err != nil {
				logger.Warn("Failed to copy file %s: %v", relPath, err)
				return nil // Continue with other files
			}
			filesCopied++
			logger.Debug("📄 Copied changed file: %s", relPath)
		} else {
			filesSkipped++
			logger.Debug("⏭️  Skipped unchanged file: %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to copy to repository: %w", err)
	}

	logger.Info("📊 Local sync completed: %d files copied, %d files skipped", filesCopied, filesSkipped)
	return nil
}

// copyFromRepository copies from repository to Cursor configuration
// Uses rsync-like logic to only copy files that have actually changed
// copyFromRepositoryForce is used for initial sync - forces overwrite of local files
// but does NOT delete local files that don't exist in remote
func (s *Syncer) copyFromRepositoryForce() error {
	logger.Debug("Copying from repository to Cursor config (FORCE mode for initial sync)...")

	cursorPath := s.config.Cursor.ConfigPath
	userPath := filepath.Join(cursorPath, "User")
	repoPath := s.config.Repository.LocalPath
	repoUserPath := filepath.Join(repoPath, "User")

	// Check if User directory exists in repository
	if _, err := os.Stat(repoUserPath); os.IsNotExist(err) {
		logger.Debug("User directory does not exist in repository, skipping sync")
		return nil
	}

	var filesCopied int

	err := filepath.Walk(repoUserPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Get relative path from User directory in repository
		relPath, err := filepath.Rel(repoUserPath, path)
		if err != nil {
			return nil
		}

		destPath := filepath.Join(userPath, relPath)

		if info.IsDir() {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(destPath, info.Mode()); err != nil {
				logger.Debug("Failed to create directory %s: %v", destPath, err)
			}
			return nil
		}

		// For initial sync, ALWAYS copy files from remote to local (force overwrite)
		// This ensures we get the remote settings but don't lose local files that aren't in remote
		if err := s.copyFile(path, destPath); err != nil {
			logger.Warn("Failed to copy file %s: %v", relPath, err)
			return nil // Continue with other files
		}
		filesCopied++
		logger.Debug("📄 FORCE copied file (initial sync): %s", relPath)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to copy from repository: %w", err)
	}

	logger.Info("📊 Initial sync completed: %d files copied from remote", filesCopied)
	return nil
}

// Only targets the User folder
func (s *Syncer) copyFromRepository() error {
	logger.Debug("Copying from repository to Cursor config (rsync mode)...")

	cursorPath := s.config.Cursor.ConfigPath
	userPath := filepath.Join(cursorPath, "User")
	repoPath := s.config.Repository.LocalPath
	repoUserPath := filepath.Join(repoPath, "User")

	// Check if User directory exists in repository
	if _, err := os.Stat(repoUserPath); os.IsNotExist(err) {
		logger.Debug("User directory does not exist in repository, skipping sync")
		return nil
	}

	var filesCopied, filesSkipped int

	err := filepath.Walk(repoUserPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Get relative path from User directory in repository
		relPath, err := filepath.Rel(repoUserPath, path)
		if err != nil {
			return nil
		}

		destPath := filepath.Join(userPath, relPath)

		if info.IsDir() {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(destPath, info.Mode()); err != nil {
				logger.Debug("Failed to create directory %s: %v", destPath, err)
			}
			return nil
		}

		// For files, check if we need to copy
		if s.shouldCopyFile(path, destPath, info) {
			if err := s.copyFile(path, destPath); err != nil {
				logger.Warn("Failed to copy file %s: %v", relPath, err)
				return nil // Continue with other files
			}
			filesCopied++
			logger.Debug("📄 Copied changed file: %s", relPath)
		} else {
			filesSkipped++
			logger.Debug("⏭️  Skipped unchanged file: %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to copy from repository: %w", err)
	}

	logger.Info("📊 Repository sync completed: %d files copied, %d files skipped", filesCopied, filesSkipped)
	return nil
}

// shouldCopyFile determines if a file should be copied based on content hash comparison
func (s *Syncer) shouldCopyFile(srcPath, destPath string, srcInfo os.FileInfo) bool {

	// Check if destination file exists
	destInfo, err := os.Stat(destPath)
	if err != nil {
		// Destination doesn't exist - definitely copy
		logger.Debug("RSYNC: Destination doesn't exist, copying: %s", filepath.Base(srcPath))
		return true
	}

	// Check file size first (fastest check)
	if srcInfo.Size() != destInfo.Size() {
		logger.Debug("RSYNC: Size differs, copying: %s (src: %d, dest: %d)", filepath.Base(srcPath), srcInfo.Size(), destInfo.Size())
		return true
	}

	logger.Debug("RSYNC: Sizes match, calculating hashes for: %s", filepath.Base(srcPath))

	// If sizes are equal, compare content hashes (most accurate)
	srcHash, err := s.calculateFileHashWithPolling(srcPath, s.config.Sync.HashPollingTimeout)
	if err != nil {
		logger.Debug("RSYNC: Could not calculate source hash, copying: %s (error: %v)", filepath.Base(srcPath), err)
		return true
	}

	destHash, err := s.calculateFileHashWithPolling(destPath, s.config.Sync.HashPollingTimeout)
	if err != nil {
		logger.Debug("RSYNC: Could not calculate destination hash, copying: %s (error: %v)", filepath.Base(srcPath), err)
		return true
	}

	if srcHash != destHash {
		logger.Debug("RSYNC: Content hash differs, copying: %s (src: %s, dest: %s)", filepath.Base(srcPath), srcHash[:8], destHash[:8])
		return true
	}

	logger.Debug("RSYNC: Skipping identical file (same hash): %s", filepath.Base(srcPath))
	return false
}

// calculateFileHash calculates SHA256 hash of a file with throttling and caching
func (s *Syncer) calculateFileHash(filePath string) (string, error) {
	logger.Debug("🔍 calculateFileHash called for: %s", filepath.Base(filePath))

	// Check cache first
	s.hashCacheMutex.RLock()
	if hash, exists := s.hashCache[filePath]; exists {
		s.hashCacheMutex.RUnlock()
		logger.Debug("🔍 Hash found in cache for: %s", filepath.Base(filePath))
		return hash, nil
	}
	s.hashCacheMutex.RUnlock()

	logger.Debug("🔍 Hash not in cache, calculating for: %s", filepath.Base(filePath))
	// Use parallel hash calculation
	return s.calculateFileHashParallel(filePath)
}

// calculateFileHashParallel calculates hash using parallel workers
func (s *Syncer) calculateFileHashParallel(filePath string) (string, error) {
	// Send job to worker
	select {
	case s.hashJobChan <- filePath:
	default:
		// If channel is full, fall back to synchronous calculation
		logger.Debug("Hash job channel full, using synchronous calculation for %s", filepath.Base(filePath))
		return s.calculateSingleFileHash(filePath)
	}

	// Wait for result
	select {
	case result := <-s.hashResultChan:
		if result.Error != nil {
			return "", result.Error
		}

		// Cache the result
		s.hashCacheMutex.Lock()
		s.hashCache[filePath] = result.Hash
		s.hashCacheMutex.Unlock()

		return result.Hash, nil
	case <-time.After(30 * time.Second): // Timeout after 30 seconds
		return "", fmt.Errorf("hash calculation timeout for %s", filePath)
	}
}

// clearHashCache clears the hash cache for a specific file or all files
func (s *Syncer) clearHashCache(filePath string) {
	s.hashCacheMutex.Lock()
	if filePath == "" {
		// Clear entire cache
		s.hashCache = make(map[string]string)
	} else {
		// Clear specific file
		delete(s.hashCache, filePath)
	}
	s.hashCacheMutex.Unlock()
}

// calculateFileHashesParallel calculates hashes for multiple files in parallel
func (s *Syncer) calculateFileHashesParallel(filePaths []string) map[string]string {
	if len(filePaths) == 0 {
		return make(map[string]string)
	}

	results := make(map[string]string)
	resultsMutex := sync.Mutex{}
	var wg sync.WaitGroup

	// Send all files to workers
	for _, filePath := range filePaths {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()

			hash, err := s.calculateFileHash(fp)
			if err != nil {
				logger.Debug("Failed to calculate hash for %s: %v", fp, err)
				return
			}

			resultsMutex.Lock()
			results[fp] = hash
			resultsMutex.Unlock()
		}(filePath)
	}

	// Wait for all calculations to complete
	wg.Wait()

	logger.Debug("Calculated hashes for %d files in parallel", len(results))
	return results
}

// calculateFileHashWithPolling calculates hash with polling if already in progress
func (s *Syncer) calculateFileHashWithPolling(filePath string, maxWaitTime time.Duration) (string, error) {
	// Check if hash calculation is already in progress for this file
	s.hashCacheMutex.RLock()
	if _, exists := s.hashCache[filePath]; exists {
		s.hashCacheMutex.RUnlock()
		return s.calculateFileHash(filePath)
	}
	s.hashCacheMutex.RUnlock()

	// Start hash calculation with polling
	startTime := time.Now()
	for time.Since(startTime) < maxWaitTime {
		hash, err := s.calculateFileHash(filePath)
		if err == nil {
			return hash, nil
		}

		// Wait before retrying
		time.Sleep(100 * time.Millisecond)
	}

	return "", fmt.Errorf("hash calculation timeout after %v", maxWaitTime)
}

// Close cleans up resources and stops hash workers
func (s *Syncer) Close() error {
	s.stopHashWorkers()
	return nil
}

func (s *Syncer) copyFile(src, dst string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write destination file
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	logger.Debug("Copied file: %s -> %s", src, dst)
	return nil
}

// CleanupExcludedFiles removes files from the repository that should be excluded
// This ensures that when users update their exclusion list, previously synced files
// that should now be excluded are automatically removed from the repository
func (s *Syncer) CleanupExcludedFiles() error {
	logger.Debug("Cleaning up excluded files from repository...")

	repoPath := s.config.Repository.LocalPath
	var filesToRemove []string

	// Walk through the repository and find files that should be excluded
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible files
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Get relative path from repository root
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Check if this path should be excluded
		if s.shouldExcludePath(relPath) {
			filesToRemove = append(filesToRemove, path)
			logger.Debug("Marked for removal (excluded): %s", relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan repository for excluded files: %w", err)
	}

	// Remove the excluded files
	for _, filePath := range filesToRemove {
		if err := os.RemoveAll(filePath); err != nil {
			logger.Warn("Failed to remove excluded file %s: %v", filePath, err)
			continue
		}
		logger.Debug("Removed excluded file from repository: %s", filePath)
	}

	if len(filesToRemove) > 0 {
		logger.Info("🧹 Cleaned up %d excluded files from repository", len(filesToRemove))
	} else {
		logger.Debug("No excluded files found in repository")
	}

	return nil
}

func (s *Syncer) shouldExcludePath(path string) bool {
	// Always exclude the custom sync marker file (local only)
	if strings.HasSuffix(path, ".custom.sync") {
		return true
	}

	for _, excludePattern := range s.config.Cursor.ExcludePaths {
		// Handle ** glob pattern for recursive matching
		if strings.Contains(excludePattern, "**") {
			if s.matchesRecursivePattern(path, excludePattern) {
				return true
			}
		} else {
			// Handle regular patterns
			matched, _ := filepath.Match(excludePattern, path)
			if matched || strings.HasPrefix(path, excludePattern) {
				return true
			}
		}
	}
	return false
}

// matchesRecursivePattern checks if a path matches a ** glob pattern
func (s *Syncer) matchesRecursivePattern(path, pattern string) bool {
	// Convert ** pattern to regex-like matching
	// **/node_modules/ -> matches any path containing /node_modules/
	// **/node_modules -> matches any path ending with /node_modules

	// Remove ** from pattern
	cleanPattern := strings.ReplaceAll(pattern, "**", "")

	// Handle trailing slash
	if strings.HasSuffix(cleanPattern, "/") {
		// Pattern like **/node_modules/ - match any path containing /node_modules/
		return strings.Contains(path, cleanPattern)
	} else {
		// Pattern like **/node_modules - match any path ending with /node_modules
		return strings.HasSuffix(path, cleanPattern) || strings.Contains(path, cleanPattern+"/")
	}
}

// ShouldPush determines if a push is needed based on time interval
func (s *Syncer) ShouldPush() bool {
	return s.forcePush || time.Since(s.lastSync) >= s.config.Sync.PushInterval
}

// ShouldPull determines if a pull is needed based on time interval
func (s *Syncer) ShouldPull() bool {
	return s.forcePull || time.Since(s.lastSync) >= s.config.Sync.PullInterval
}

// hasCustomSyncMarker checks if the custom sync marker file exists
func (s *Syncer) hasCustomSyncMarker() bool {
	markerPath := filepath.Join(s.config.Cursor.ConfigPath, ".custom.sync")
	_, err := os.Stat(markerPath)
	return err == nil
}

// createCustomSyncMarker creates the custom sync marker file
func (s *Syncer) createCustomSyncMarker() error {
	markerPath := filepath.Join(s.config.Cursor.ConfigPath, ".custom.sync")

	// Create the marker file with timestamp and sync information
	content := fmt.Sprintf(`cursor-sync marker file

This file indicates that cursor-sync has synchronized these Cursor settings.

✅ Local settings have been synced at least once
✅ It's safe to perform bidirectional sync operations
✅ Local files are not "fresh/virgin" - they contain synced data

Last sync: %s
Repository: %s

🚨 DO NOT DELETE THIS FILE
If deleted, cursor-sync will treat local settings as "fresh" and overwrite them from remote.
`, time.Now().Format("2006-01-02 15:04:05"), s.config.Repository.URL)

	if err := os.WriteFile(markerPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create custom sync marker: %w", err)
	}

	logger.Debug("Created/updated custom sync marker at: %s", markerPath)
	return nil
}

// checkRepositoryPrivacy verifies that the repository is private
func (s *Syncer) checkRepositoryPrivacy() error {
	logger.Info("Checking repository privacy for security...")

	checker := privacy.NewRepositoryChecker()
	isPrivate, err := checker.CheckRepositoryPrivacy(s.config.Repository.URL)

	if err != nil {
		privacy.ShowPrivacyCheckError(s.config.Repository.URL, err)
		return fmt.Errorf("cannot verify repository privacy - sync blocked for security")
	}

	if !isPrivate {
		privacy.ShowPrivacyWarning(s.config.Repository.URL)
		return fmt.Errorf("public repository detected - sync blocked for security")
	}

	logger.Info("✅ Repository privacy verified - proceeding with sync")
	return nil
}
