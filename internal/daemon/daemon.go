package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cursor-sync/internal/auth"
	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
	"cursor-sync/internal/sync"
	"cursor-sync/internal/watcher"
)

// Daemon represents the main sync daemon
type Daemon struct {
	config  *config.Config
	syncer  *sync.Syncer
	watcher *watcher.Watcher
	paused  bool
}

// New creates a new daemon instance
func New(cfg *config.Config) (*Daemon, error) {
	// Check GitHub token availability first
	if !auth.HasValidToken() {
		auth.ShowTokenRequiredMessage()
		return nil, fmt.Errorf("GitHub token required for operation")
	}

	// Initialize logger with config
	if err := logger.InitWithConfig(cfg.Logging.Level, cfg.Logging.LogDir, false); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create syncer
	syncer, err := sync.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create syncer: %w", err)
	}

	// Create file watcher if enabled
	var fileWatcher *watcher.Watcher
	if cfg.Sync.WatchEnabled {
		fileWatcher, err = watcher.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create file watcher: %w", err)
		}
	}

	return &Daemon{
		config:  cfg,
		syncer:  syncer,
		watcher: fileWatcher,
		paused:  false,
	}, nil
}

// Start starts the daemon
func (d *Daemon) Start(ctx context.Context) error {
	logger.Info("Starting Cursor Sync daemon...")

	// Initialize syncer
	if err := d.syncer.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize syncer: %w", err)
	}

	// Perform initial sync on startup
	logger.Info("Performing initial sync on daemon startup...")
	if err := d.performInitialSync(); err != nil {
		logger.Error("Initial sync failed: %v", err)
		// Don't fail daemon startup, just log the error
	} else {
		logger.Info("Initial sync completed successfully")
	}

	// Start DUAL SYNC SYSTEM: Real-time (primary) + Periodic (fallback)

	// PRIMARY: Start real-time file watcher (fsnotify)
	if d.watcher != nil {
		logger.Info("🚀 Starting PRIMARY sync method: Real-time file watching (fsnotify)")
		go func() {
			if err := d.watcher.Start(ctx); err != nil {
				logger.Error("File watcher error: %v", err)
			}
		}()

		// Handle real-time file changes
		go d.handleFileChanges(ctx)
	} else {
		logger.Warn("⚠️  Real-time file watching disabled - relying on periodic sync only")
	}

	// FALLBACK: Start periodic sync timers
	logger.Info("🚀 Starting FALLBACK sync method: Periodic intervals")
	pullTicker := time.NewTicker(d.config.Sync.PullInterval)
	pushTicker := time.NewTicker(d.config.Sync.PushInterval)

	defer pullTicker.Stop()
	defer pushTicker.Stop()

	// Start periodic sync loops (running in parallel with real-time)
	go d.syncLoop(ctx, pullTicker, pushTicker)

	logger.Info("Daemon started successfully")

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info("Daemon shutting down...")
	return nil
}

// syncLoop handles periodic sync operations (fallback method)
func (d *Daemon) syncLoop(ctx context.Context, pullTicker, pushTicker *time.Ticker) {
	logger.Info("🕒 Periodic sync active (fallback method) - Pull: %v, Push: %v",
		d.config.Sync.PullInterval, d.config.Sync.PushInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Periodic sync loop shutting down")
			return
		case <-pullTicker.C:
			if !d.isPaused() {
				logger.Debug("🔄 Periodic pull sync triggered")
				d.performPull()
			}
		case <-pushTicker.C:
			if !d.isPaused() {
				logger.Debug("🔄 Periodic push sync triggered")
				d.performPush()
			}
		}
	}
}

// handleFileChanges handles real-time file changes via fsnotify (primary sync method)
func (d *Daemon) handleFileChanges(ctx context.Context) {
	changes := d.watcher.Changes()

	// Configurable debounce to avoid excessive syncs (minimum 10 seconds)
	debounceTime := d.config.Sync.DebounceTime
	var pendingChanges bool
	debounceTimer := time.NewTimer(debounceTime)
	debounceTimer.Stop()

	logger.Info("🔍 Real-time file watcher active (fsnotify) - primary sync method")
	logger.Info("⏱️  Debounce time configured: %v", debounceTime)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Real-time file watcher shutting down")
			return
		case fileChange := <-changes:
			if !d.isPaused() {
				logger.Debug("📁 File change detected: %s (%s)", fileChange.Path, fileChange.Action)
				logger.Debug("⏳ Starting/resetting %v debounce timer", debounceTime)
				pendingChanges = true
				debounceTimer.Reset(debounceTime)
			}
		case <-debounceTimer.C:
			if pendingChanges && !d.isPaused() {
				logger.Info("⚡ Real-time sync triggered after %v debounce period", debounceTime)

				// Perform comprehensive sync (pull then push)
				d.performRealtimeSync()
				pendingChanges = false
			}
		}
	}
}

func (d *Daemon) performPull() {
	logger.Debug("📥 Performing periodic pull sync...")

	// Disable file watcher during sync to prevent infinite loops
	if d.watcher != nil {
		d.watcher.Disable()
		defer d.watcher.Enable()
	}

	if err := d.syncer.SyncFromRemote(); err != nil {
		logger.Error("Periodic pull sync failed: %v", err)
	} else {
		logger.Debug("✅ Periodic pull sync completed")
	}
}

func (d *Daemon) performPush() {
	logger.Debug("📤 Performing periodic push sync...")

	// Disable file watcher during sync to prevent infinite loops
	if d.watcher != nil {
		d.watcher.Disable()
		defer d.watcher.Enable()
	}

	if err := d.syncer.SyncToRemote(); err != nil {
		logger.Error("Periodic push sync failed: %v", err)
	} else {
		logger.Debug("✅ Periodic push sync completed")
	}
}

// performRealtimeSync performs a real-time sync (triggered by file changes)
// When user makes local changes, we ONLY push them to remote (they're the freshest)
func (d *Daemon) performRealtimeSync() {
	logger.Info("⚡ Performing real-time sync sequence...")

	// Disable file watcher during sync to prevent infinite loops
	if d.watcher != nil {
		d.watcher.Disable()
		defer d.watcher.Enable()
	}

	// When user makes local changes, ONLY push them to remote
	// DO NOT pull from remote as it would overwrite the user's changes
	logger.Debug("📤 Real-time sync: pushing local changes to remote...")
	if err := d.syncer.SyncToRemote(); err != nil {
		logger.Error("Real-time push failed: %v", err)
		// Don't fail the entire sync operation, just log the error
		// The periodic sync will handle any remaining conflicts
	} else {
		logger.Info("✅ Real-time sync completed successfully")
	}
}

// ForceInitialSync triggers an initial sync (used for restart scenarios)
func (d *Daemon) ForceInitialSync() error {
	logger.Info("Forcing initial sync...")
	return d.performInitialSync()
}

// performInitialSync performs a comprehensive initial sync on daemon startup
func (d *Daemon) performInitialSync() error {
	if d.isPaused() {
		logger.Info("Daemon is paused, skipping initial sync")
		return nil
	}

	logger.Info("🔄 Starting initial sync sequence...")

	// Step 1: Pull from remote to get any changes that happened while daemon was off
	logger.Info("📥 Step 1: Pulling remote changes...")
	if err := d.syncer.SyncFromRemote(); err != nil {
		logger.Error("Failed to pull remote changes during initial sync: %v", err)
		// Continue with push even if pull fails
	} else {
		logger.Info("✅ Remote changes pulled successfully")
	}

	// Step 2: Push any local changes that might have accumulated
	logger.Info("📤 Step 2: Pushing local changes...")
	if err := d.syncer.SyncToRemote(); err != nil {
		logger.Error("Failed to push local changes during initial sync: %v", err)
		return fmt.Errorf("initial push sync failed: %w", err)
	} else {
		logger.Info("✅ Local changes pushed successfully")
	}

	logger.Info("🎉 Initial sync sequence completed")
	return nil
}

func (d *Daemon) isPaused() bool {
	// Check if pause file exists
	home, err := os.UserHomeDir()
	if err != nil {
		return d.paused
	}

	pauseFile := filepath.Join(home, ".cursor-sync", "paused")
	_, err = os.Stat(pauseFile)

	return err == nil
}
