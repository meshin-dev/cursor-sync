package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cursor-sync/internal/auth"
	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
	"cursor-sync/internal/privacy"
)

// Installer handles the installation process
type Installer struct {
	repoURL string
	force   bool
}

// New creates a new installer
func New(repoURL string, force bool) *Installer {
	return &Installer{
		repoURL: repoURL,
		force:   force,
	}
}

// Install performs the full installation process
func (i *Installer) Install() error {
	logger.Info("Starting cursor-sync installation...")

	// Check GitHub token availability first
	if !auth.HasValidToken() {
		fmt.Println("❌ GitHub token required for installation")
		auth.ShowTokenRequiredMessage()
		fmt.Println("Please run 'cursor-sync token <your-github-token>' first")
		return fmt.Errorf("GitHub token required for installation")
	}

	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if already installed
	if !i.force {
		configDir := filepath.Join(home, ".cursor-sync")
		if _, err := os.Stat(configDir); err == nil {
			return fmt.Errorf("cursor-sync is already installed. Use --force to reinstall")
		}
	}

	// Create configuration directory
	configDir := filepath.Join(home, ".cursor-sync")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if user already has a configuration from setup
	userConfigPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(userConfigPath); err == nil {
		logger.Info("Found existing configuration from setup: %s", userConfigPath)
		logger.Info("Using existing configuration - skipping copy from project config")
	} else {
		// Copy configuration from project config/sync.yaml (for manual setup)
		if err := i.copyProjectConfig(configDir); err != nil {
			return fmt.Errorf("failed to copy configuration: %w", err)
		}
	}

	// Verify repository privacy before proceeding
	if err := i.checkRepositoryPrivacy(); err != nil {
		return fmt.Errorf("repository privacy check failed: %w", err)
	}

	// Build the binary
	if err := i.buildBinary(); err != nil {
		return fmt.Errorf("failed to build binary: %w", err)
	}

	// Create LaunchAgent plist
	if err := i.createLaunchAgent(home); err != nil {
		return fmt.Errorf("failed to create LaunchAgent: %w", err)
	}

	// Load LaunchAgent
	if err := i.loadLaunchAgent(home); err != nil {
		return fmt.Errorf("failed to load LaunchAgent: %w", err)
	}

	logger.Info("Installation completed successfully")
	return nil
}

func (i *Installer) copyProjectConfig(configDir string) error {
	logger.Info("Copying project configuration...")

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Source and destination paths
	srcPath := filepath.Join(wd, "config", "sync.yaml")
	destPath := filepath.Join(configDir, "config.yaml")

	// Check if source exists
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("config/sync.yaml not found. Please copy config/sync.example.yaml to config/sync.yaml and edit it first")
	}

	// Read and copy config file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Check if the config still has placeholder values
	configContent := string(data)
	if strings.Contains(configContent, "REPLACE_WITH_YOUR_USERNAME") ||
		strings.Contains(configContent, "REPLACE_WITH_YOUR_REPO") {
		return fmt.Errorf("config/sync.yaml still contains placeholder values. Please edit config/sync.yaml and replace the repository URL with your actual repository")
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Info("Configuration copied to: %s", destPath)
	return nil
}

func (i *Installer) buildBinary() error {
	logger.Info("Building cursor-sync binary...")

	// Find project root
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create bin directory
	binDir := filepath.Join(wd, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Build binary
	binaryPath := filepath.Join(binDir, "cursor-sync")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = wd

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build binary: %w\nOutput: %s", err, string(output))
	}

	// Make binary executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	logger.Info("Binary built successfully at: %s", binaryPath)
	return nil
}

func (i *Installer) createLaunchAgent(home string) error {
	logger.Info("Creating LaunchAgent plist...")

	// Get current working directory for binary path
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	binaryPath := filepath.Join(wd, "bin", "cursor-sync")
	logPath := filepath.Join(home, ".cursor-sync", "logs", "daemon.log")

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.cursorsync</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>%s</string>
    </dict>
    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>`, binaryPath, logPath, logPath, home)

	// Create LaunchAgents directory
	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Write plist file
	plistPath := filepath.Join(launchAgentsDir, "com.user.cursorsync.plist")
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	logger.Info("LaunchAgent plist created at: %s", plistPath)
	return nil
}

func (i *Installer) loadLaunchAgent(home string) error {
	logger.Info("Loading LaunchAgent...")

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.user.cursorsync.plist")

	// Unload first in case it's already loaded
	exec.Command("launchctl", "unload", plistPath).Run()

	// Load the LaunchAgent
	cmd := exec.Command("launchctl", "load", plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to load LaunchAgent: %w\nOutput: %s", err, string(output))
	}

	logger.Info("LaunchAgent loaded successfully")
	return nil
}

// checkRepositoryPrivacy verifies the repository is private during installation
func (i *Installer) checkRepositoryPrivacy() error {
	// Load configuration using the same mechanism as the rest of the application
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	repoURL := cfg.Repository.URL
	if repoURL == "" {
		return fmt.Errorf("repository URL not found in configuration")
	}

	logger.Info("Verifying repository privacy for: %s", repoURL)

	checker := privacy.NewRepositoryChecker()
	isPrivate, err := checker.CheckRepositoryPrivacy(repoURL)

	if err != nil {
		privacy.ShowPrivacyCheckError(repoURL, err)
		return fmt.Errorf("cannot verify repository privacy - installation blocked")
	}

	if !isPrivate {
		privacy.ShowPrivacyWarning(repoURL)
		return fmt.Errorf("public repository detected - installation blocked")
	}

	logger.Info("✅ Repository privacy verified - proceeding with installation")
	return nil
}
