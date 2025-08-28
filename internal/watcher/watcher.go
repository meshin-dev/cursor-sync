package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
)

// FileChange represents a file system change
type FileChange struct {
	Path   string
	Action string // "create", "modify", "delete"
}

// Watcher watches for file system changes
type Watcher struct {
	fsWatcher     *fsnotify.Watcher
	config        *config.Config
	changeChan    chan FileChange
	debounceTime  time.Duration
	lastChangeMap map[string]time.Time
	disabled      bool
	disabledMutex sync.RWMutex
	watchMutex    sync.Mutex
}

// New creates a new file watcher
func New(cfg *config.Config) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		fsWatcher:     fsWatcher,
		config:        cfg,
		changeChan:    make(chan FileChange, 100),
		debounceTime:  cfg.Sync.DebounceTime,
		lastChangeMap: make(map[string]time.Time),
	}, nil
}

// Start starts watching for file changes
func (w *Watcher) Start(ctx context.Context) error {
	// Add watch paths
	if err := w.addWatchPaths(); err != nil {
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	logger.Info("File watcher started")

	// Start event processing goroutine
	go w.processEvents(ctx)

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info("Stopping file watcher...")
	return w.fsWatcher.Close()
}

// Changes returns a channel that receives file change notifications
func (w *Watcher) Changes() <-chan FileChange {
	return w.changeChan
}

// Disable temporarily disables the file watcher
func (w *Watcher) Disable() {
	w.disabledMutex.Lock()
	defer w.disabledMutex.Unlock()
	w.disabled = true
	logger.Debug("File watcher disabled")
}

// Enable re-enables the file watcher
func (w *Watcher) Enable() {
	w.disabledMutex.Lock()
	defer w.disabledMutex.Unlock()
	w.disabled = false
	logger.Debug("File watcher enabled")
}

// RestartWatching restarts the watching process for the entire User directory
func (w *Watcher) RestartWatching() error {
	w.watchMutex.Lock()
	defer w.watchMutex.Unlock()

	logger.Debug("Restarting file watching process...")

	// Remove all current watches
	for _, path := range w.fsWatcher.WatchList() {
		w.fsWatcher.Remove(path)
	}

	// Re-add all watch paths
	if err := w.addWatchPaths(); err != nil {
		return fmt.Errorf("failed to restart watching: %w", err)
	}

	logger.Debug("File watching process restarted successfully")
	return nil
}

func (w *Watcher) addWatchPaths() error {
	basePath := w.config.Cursor.ConfigPath
	userPath := filepath.Join(basePath, "User")

	// Check if User directory exists
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		return fmt.Errorf("User directory does not exist: %s", userPath)
	}

	logger.Debug("Adding User directory watch path: %s", userPath)
	if err := w.fsWatcher.Add(userPath); err != nil {
		return fmt.Errorf("failed to add User path: %w", err)
	}

	// Add all subdirectories recursively within User (watch everything except excluded paths)
	return w.addDirectoryWatch(userPath)
}

func (w *Watcher) addDirectoryWatch(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if info.IsDir() && !w.shouldExcludePath(path) {
			logger.Debug("Adding watch for directory: %s", path)
			if err := w.fsWatcher.Add(path); err != nil {
				logger.Warn("Failed to add watch for %s: %v", path, err)
			}
		}
		return nil
	})
}

// addNewDirectoryToWatch adds a newly created directory to the watch list
func (w *Watcher) addNewDirectoryToWatch(dirPath string) {
	w.watchMutex.Lock()
	defer w.watchMutex.Unlock()

	if !w.shouldExcludePath(dirPath) {
		logger.Debug("Adding new directory to watch: %s", dirPath)
		if err := w.fsWatcher.Add(dirPath); err != nil {
			logger.Warn("Failed to add new directory to watch %s: %v", dirPath, err)
		}
	}
}

func (w *Watcher) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			if w.shouldProcessEvent(event) {
				w.handleEvent(event)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			logger.Error("File watcher error: %v", err)
		}
	}
}

func (w *Watcher) shouldProcessEvent(event fsnotify.Event) bool {
	// Check if watcher is disabled
	w.disabledMutex.RLock()
	if w.disabled {
		w.disabledMutex.RUnlock()
		return false
	}
	w.disabledMutex.RUnlock()

	// Process create, write, and remove events
	if event.Op&fsnotify.Create == 0 && event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Remove == 0 {
		return false
	}

	// Check if path should be excluded
	if w.shouldExcludePath(event.Name) {
		return false
	}

	// Check if path matches watch patterns
	if !w.matchesWatchPattern(event.Name) {
		return false
	}

	// Debounce rapid changes
	now := time.Now()
	if lastChange, exists := w.lastChangeMap[event.Name]; exists {
		if now.Sub(lastChange) < w.debounceTime {
			return false
		}
	}

	w.lastChangeMap[event.Name] = now

	return true
}

func (w *Watcher) shouldExcludePath(path string) bool {
	userPath := filepath.Join(w.config.Cursor.ConfigPath, "User")
	relativePath, err := filepath.Rel(userPath, path)
	if err != nil {
		return false
	}

	for _, excludePattern := range w.config.Cursor.ExcludePaths {
		// Remove "User/" prefix from exclude patterns for comparison
		pattern := strings.TrimPrefix(excludePattern, "User/")
		matched, _ := filepath.Match(pattern, relativePath)
		if matched || strings.Contains(relativePath, pattern) {
			return true
		}
	}

	return false
}

func (w *Watcher) matchesWatchPattern(path string) bool {
	// If no include patterns specified, include all non-excluded files
	if len(w.config.Cursor.IncludePaths) == 0 {
		return true
	}

	relativePath, err := filepath.Rel(w.config.Cursor.ConfigPath, path)
	if err != nil {
		return false
	}

	// Check against include patterns
	for _, pattern := range w.config.Cursor.IncludePaths {
		matched, _ := filepath.Match(pattern, relativePath)
		if matched || strings.Contains(relativePath, pattern) {
			return true
		}
	}

	return false
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	logger.Debug("File changed: %s (%s)", event.Name, event.Op.String())

	// Handle directory creation by adding it to watch list
	if event.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			w.addNewDirectoryToWatch(event.Name)
		}
	}

	// Determine the action based on the event type
	var action string
	switch {
	case event.Op&fsnotify.Create != 0:
		action = "create"
	case event.Op&fsnotify.Write != 0:
		action = "modify"
	case event.Op&fsnotify.Remove != 0:
		action = "delete"
	default:
		action = "unknown"
	}

	change := FileChange{
		Path:   event.Name,
		Action: action,
	}

	// Send change notification (non-blocking)
	select {
	case w.changeChan <- change:
	default:
		logger.Warn("Change channel full, dropping event for: %s", event.Name)
	}
}
