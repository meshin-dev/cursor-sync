package cursor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"cursor-sync/internal/logger"
)

// CursorDetector handles detection and validation of Cursor installation
type CursorDetector struct {
	configPath string
}

// NewDetector creates a new CursorDetector with the given config path
func NewDetector(configPath string) *CursorDetector {
	return &CursorDetector{
		configPath: configPath,
	}
}

// DetectAndValidate performs comprehensive Cursor installation detection and validation
func (d *CursorDetector) DetectAndValidate() error {
	// Step 1: Validate the configured path exists
	if err := d.validateConfigPath(); err != nil {
		return err
	}

	// Step 2: Check for Cursor installation indicators
	if err := d.validateCursorInstallation(); err != nil {
		return err
	}

	// Step 3: Check for User directory (where settings are stored)
	if err := d.validateUserDirectory(); err != nil {
		return err
	}

	logger.Info("✅ Cursor installation detected and validated: %s", d.configPath)
	return nil
}

// validateConfigPath checks if the configured Cursor path exists
func (d *CursorDetector) validateConfigPath() error {
	if d.configPath == "" {
		return fmt.Errorf("cursor config path is empty")
	}

	// Expand home directory if needed
	expandedPath, err := expandPath(d.configPath)
	if err != nil {
		return fmt.Errorf("failed to expand cursor config path: %w", err)
	}
	d.configPath = expandedPath

	// Check if directory exists
	info, err := os.Stat(d.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cursor configuration directory not found: %s\n\n%s",
				d.configPath, getCursorNotFoundHelp())
		}
		return fmt.Errorf("failed to access cursor config directory %s: %w", d.configPath, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("cursor config path is not a directory: %s", d.configPath)
	}

	return nil
}

// validateCursorInstallation checks for key indicators that Cursor is installed
func (d *CursorDetector) validateCursorInstallation() error {
	// Check for essential Cursor directories/files that indicate installation
	indicators := []struct {
		path        string
		description string
		required    bool
	}{
		{"User", "User settings directory", true},
		{"extensions", "Extensions directory", false},
		{"logs", "Logs directory", false},
	}

	foundIndicators := 0
	requiredIndicators := 0

	for _, indicator := range indicators {
		if indicator.required {
			requiredIndicators++
		}

		indicatorPath := filepath.Join(d.configPath, indicator.path)
		if _, err := os.Stat(indicatorPath); err == nil {
			foundIndicators++
			logger.Debug("Found Cursor indicator: %s", indicator.description)
		} else if indicator.required {
			return fmt.Errorf("required Cursor directory missing: %s (%s)\n\n%s",
				indicatorPath, indicator.description, getCursorNotFoundHelp())
		}
	}

	if foundIndicators == 0 {
		return fmt.Errorf("no Cursor installation indicators found in: %s\n\n%s",
			d.configPath, getCursorNotFoundHelp())
	}

	logger.Debug("Found %d/%d Cursor installation indicators", foundIndicators, len(indicators))
	return nil
}

// validateUserDirectory checks that the User directory exists and contains expected structure
func (d *CursorDetector) validateUserDirectory() error {
	userDir := filepath.Join(d.configPath, "User")

	info, err := os.Stat(userDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cursor User directory not found: %s\n\n%s",
				userDir, getCursorUserDirHelp())
		}
		return fmt.Errorf("failed to access cursor User directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("cursor User path is not a directory: %s", userDir)
	}

	// Check for typical settings files (optional but good indicators)
	settingsFiles := []string{
		"settings.json",
		"keybindings.json",
	}

	foundSettings := 0
	for _, file := range settingsFiles {
		filePath := filepath.Join(userDir, file)
		if _, err := os.Stat(filePath); err == nil {
			foundSettings++
			logger.Debug("Found Cursor settings file: %s", file)
		}
	}

	if foundSettings == 0 {
		logger.Info("No existing settings files found - this appears to be a fresh Cursor installation")

		// Create basic settings.json if it doesn't exist to ensure sync has something to work with
		if err := d.ensureBasicSettings(); err != nil {
			logger.Warn("Failed to create basic settings: %v", err)
		}
	} else {
		logger.Debug("Found %d existing settings files", foundSettings)
	}

	return nil
}

// ensureBasicSettings creates a minimal settings.json if none exists
func (d *CursorDetector) ensureBasicSettings() error {
	userDir := filepath.Join(d.configPath, "User")
	settingsPath := filepath.Join(userDir, "settings.json")

	// Only create if it doesn't exist
	if _, err := os.Stat(settingsPath); err == nil {
		return nil
	}

	// Create minimal settings
	basicSettings := `{
    "editor.fontSize": 14,
    "editor.tabSize": 4,
    "workbench.colorTheme": "Default Dark Modern"
}
`

	if err := os.WriteFile(settingsPath, []byte(basicSettings), 0644); err != nil {
		return fmt.Errorf("failed to create basic settings.json: %w", err)
	}

	logger.Info("Created basic settings.json for fresh Cursor installation")
	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, path[1:]), nil
}

// GetDefaultCursorPath returns the default Cursor configuration path for the current OS
func GetDefaultCursorPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch runtime.GOOS {
	case "darwin": // macOS
		return filepath.Join(home, "Library", "Application Support", "Cursor")
	case "linux":
		return filepath.Join(home, ".config", "Cursor")
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Cursor")
	default:
		return ""
	}
}

// getCursorNotFoundHelp returns helpful instructions when Cursor is not found
func getCursorNotFoundHelp() string {
	return `
❌ CURSOR NOT FOUND

cursor-sync requires Cursor IDE to be installed and configured.

To resolve this:

1. 📥 Install Cursor IDE from: https://cursor.sh/
2. 🚀 Launch Cursor at least once to create configuration directory
3. ✅ Verify installation by checking for directory:
   macOS: ~/Library/Application Support/Cursor/
   Linux: ~/.config/Cursor/
   Windows: %APPDATA%/Cursor/

4. 🔄 Run cursor-sync again

Current expected path: ` + GetDefaultCursorPath() + `

If Cursor is installed in a different location, update your config/sync.yaml:
cursor:
  config_path: "/path/to/your/cursor/config"
`
}

// getCursorUserDirHelp returns help for missing User directory
func getCursorUserDirHelp() string {
	return `
❌ CURSOR USER DIRECTORY NOT FOUND

The Cursor User directory contains your settings, keybindings, and other configurations.

To resolve this:

1. 🚀 Launch Cursor IDE at least once
2. ⚙️  Open Settings (Cmd/Ctrl + ,) to create User directory
3. 📝 Make any small change to settings to ensure files are created
4. ✅ Verify User directory exists with settings files

Expected User directory: ` + filepath.Join(GetDefaultCursorPath(), "User") + `
`
}

// ShowValidationError displays a prominent error message for Cursor validation failures
func ShowValidationError(err error) {
	fmt.Println("\n================================================================================")
	fmt.Println("⚠️  CURSOR VALIDATION FAILED")
	fmt.Println("================================================================================")
	fmt.Printf("\nError: %v\n", err)
	fmt.Println("\n================================================================================")
}
