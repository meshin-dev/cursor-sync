package interactive

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"cursor-sync/internal/auth"
	"cursor-sync/internal/config"
	"cursor-sync/internal/cursor"
	"cursor-sync/internal/privacy"
)

// min returns the minimum of two integers (Go 1.21+ has this built-in)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetupWizard handles interactive configuration setup
type SetupWizard struct {
	scanner *bufio.Scanner
}

// NewSetupWizard creates a new interactive setup wizard
func NewSetupWizard() *SetupWizard {
	return &SetupWizard{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// RunInteractiveSetup performs comprehensive interactive setup for missing configurations
func (s *SetupWizard) RunInteractiveSetup() error {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🚀 CURSOR-SYNC INTERACTIVE SETUP")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
	fmt.Println("Welcome! Let's get cursor-sync configured properly.")
	fmt.Println("This wizard will help you set up missing required configurations.")
	fmt.Println()

	// Step 1: Check and setup GitHub token
	if err := s.setupGitHubToken(); err != nil {
		return fmt.Errorf("failed to setup GitHub token: %w", err)
	}

	// Step 2: Check and setup repository configuration
	if err := s.setupRepositoryConfig(); err != nil {
		return fmt.Errorf("failed to setup repository configuration: %w", err)
	}

	fmt.Println("\n🎉 Setup completed successfully!")
	fmt.Println("✅ GitHub token configured")
	fmt.Println("✅ Repository configuration saved")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  cursor-sync install    # Install the daemon")
	fmt.Println("  cursor-sync start      # Start syncing")
	fmt.Println()

	return nil
}

// CheckAndPromptMissingConfig checks for missing required configurations and prompts interactively
func (s *SetupWizard) CheckAndPromptMissingConfig() (*config.Config, error) {
	missingItems := s.detectMissingConfig()

	if len(missingItems) == 0 {
		// All required config is present, load normally
		return config.Load()
	}

	fmt.Println("\n⚠️  Missing Required Configuration")
	fmt.Println("cursor-sync needs some additional setup to continue.")
	fmt.Println()
	fmt.Println("Missing items:")
	for _, item := range missingItems {
		fmt.Printf("  ❌ %s\n", item)
	}
	fmt.Println()

	if !s.promptYesNo("Would you like to set these up now interactively?") {
		return nil, fmt.Errorf("setup cancelled - required configuration missing")
	}

	// Run interactive setup for missing items
	if err := s.runPartialSetup(missingItems); err != nil {
		return nil, fmt.Errorf("interactive setup failed: %w", err)
	}

	// Try loading config again
	return config.Load()
}

// detectMissingConfig detects what required configuration is missing
func (s *SetupWizard) detectMissingConfig() []string {
	var missing []string

	// Check GitHub token
	if !auth.HasValidToken() {
		missing = append(missing, "GitHub Personal Access Token")
	}

	// Check repository URL in config
	cfg, err := config.Load()
	if err != nil || cfg.Repository.URL == "" || strings.Contains(cfg.Repository.URL, "your-username") || strings.Contains(cfg.Repository.URL, "your-repo") {
		missing = append(missing, "Repository URL configuration")
	}

	// Check if user wants to customize Cursor installation path
	missing = append(missing, "Cursor Installation Path (optional)")

	return missing
}

// runPartialSetup runs setup only for specific missing items
func (s *SetupWizard) runPartialSetup(missingItems []string) error {
	for _, item := range missingItems {
		switch item {
		case "GitHub Personal Access Token":
			if err := s.setupGitHubToken(); err != nil {
				return err
			}
		case "Repository URL configuration":
			if err := s.setupRepositoryConfig(); err != nil {
				return err
			}
		case "Cursor Installation Path (optional)":
			if err := s.setupCursorInstallationPath(); err != nil {
				return fmt.Errorf("cursor installation path setup failed: %w", err)
			}
		default:
			// Skip unknown items silently for forward compatibility
		}
	}
	return nil
}

// setupGitHubToken handles interactive GitHub token setup with guided URLs
func (s *SetupWizard) setupGitHubToken() error {
	// Check if token already exists and is valid
	if auth.HasValidToken() {
		fmt.Println("✅ GitHub token is already configured and valid")
		return nil
	}

	fmt.Println("🔑 GitHub Personal Access Token Setup")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()
	fmt.Println("cursor-sync requires a GitHub Personal Access Token for secure repository access.")
	fmt.Println()

	// Retry loop until token is properly configured
	for {
		fmt.Println("📋 STEP-BY-STEP TOKEN CREATION:")
		fmt.Println()
		fmt.Println("1. 🌐 Open this URL in your browser:")
		fmt.Println("   👉 https://github.com/settings/tokens/new")
		fmt.Println()
		fmt.Println("2. 📝 Fill out the token creation form:")
		fmt.Println("   • Note: cursor-sync token")
		fmt.Println("   • Expiration: 90 days (or your preference)")
		fmt.Println("   • ✅ Check 'repo' scope (Full control of private repositories)")
		fmt.Println()
		fmt.Println("3. 🟢 Click 'Generate token' at the bottom")
		fmt.Println()
		fmt.Println("4. 📋 Copy the generated token (starts with ghp_ or github_pat_)")
		fmt.Println()
		fmt.Println("⚠️  IMPORTANT: You won't be able to see the token again after leaving the page!")
		fmt.Println()

		// Ask if user is ready
		if !s.promptYesNo("Have you created the token and copied it?") {
			fmt.Println()
			fmt.Println("No problem! Take your time. The setup will wait for you.")
			fmt.Println("Press Enter when you're ready to continue...")
			s.scanner.Scan()
			continue
		}

		// Get token input
		fmt.Println()
		fmt.Print("🔐 Paste your GitHub Personal Access Token: ")

		// Use secure input for token
		tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("\n❌ Failed to read token input: %v\n", err)
			fmt.Println("Let's try again...")
			continue
		}
		fmt.Println() // New line after hidden input

		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			fmt.Println("❌ No token entered. Let's try again...")
			continue
		}

		// Validate token format
		if !strings.HasPrefix(token, "ghp_") && !strings.HasPrefix(token, "github_pat_") {
			fmt.Printf("⚠️  Token doesn't start with expected prefix (ghp_ or github_pat_)\n")
			fmt.Printf("Your token starts with: %s...\n", token[:min(4, len(token))])
			fmt.Println()
			fmt.Println("Common issues:")
			fmt.Println("• Make sure you copied the entire token")
			fmt.Println("• Check you're using a Personal Access Token (classic)")
			fmt.Println("• Ensure you didn't copy extra spaces or characters")
			fmt.Println()
			if !s.promptYesNo("Continue with this token anyway?") {
				continue
			}
		}

		// Save token
		fmt.Println("💾 Saving token...")
		if err := auth.SaveGitHubToken(token); err != nil {
			fmt.Printf("❌ Failed to save token: %v\n", err)
			fmt.Println("Let's try again with a different token...")
			continue
		}

		// Validate token by testing GitHub API
		fmt.Println("🔍 Validating token with GitHub API...")
		if !auth.HasValidToken() {
			fmt.Println("❌ Token validation failed!")
			fmt.Println()
			fmt.Println("This could mean:")
			fmt.Println("• Token is expired or invalid")
			fmt.Println("• Token doesn't have 'repo' scope")
			fmt.Println("• Network connectivity issues")
			fmt.Println()
			fmt.Println("Let's create a new token...")
			continue
		}

		fmt.Println("✅ GitHub token saved and validated successfully!")
		fmt.Println("🎉 Token is working and has proper permissions!")
		break
	}

	return nil
}

// setupRepositoryConfig handles interactive repository configuration with guided creation
func (s *SetupWizard) setupRepositoryConfig() error {
	fmt.Println("📦 Repository Configuration Setup")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()
	fmt.Println("cursor-sync needs a PRIVATE Git repository to store your settings.")
	fmt.Println("🔒 CRITICAL: Repository MUST be private to protect sensitive data!")
	fmt.Println()

	// Load current config or create new
	cfg, err := s.loadOrCreateConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if repo URL is already configured and valid
	if cfg.Repository.URL != "" &&
		!strings.Contains(cfg.Repository.URL, "your-username") &&
		!strings.Contains(cfg.Repository.URL, "your-repo") &&
		!strings.Contains(cfg.Repository.URL, "octocat/Hello-World") &&
		!strings.Contains(cfg.Repository.URL, "cursor-sync-bucket") {
		fmt.Printf("📍 Current repository: %s\n", cfg.Repository.URL)
		if s.promptYesNo("Keep this repository URL?") {
			return s.validateAndSaveConfig(cfg)
		}
	}

	// Repository setup retry loop
	for {
		fmt.Println("📋 STEP-BY-STEP REPOSITORY CREATION:")
		fmt.Println()
		fmt.Println("1. 🌐 Open this URL in your browser:")
		fmt.Println("   👉 https://github.com/new")
		fmt.Println()
		fmt.Println("2. 📝 Fill out the repository creation form:")
		fmt.Println("   • Repository name: cursor-sync-bucket (recommended)")
		fmt.Println("   • Description: Cursor IDE settings sync bucket")
		fmt.Println("   • 🔒 IMPORTANT: Select 'Private' (NOT Public!)")
		fmt.Println("   • ✅ Initialize with README (optional)")
		fmt.Println()
		fmt.Println("3. 🟢 Click 'Create repository'")
		fmt.Println()
		fmt.Println("4. 📋 Copy the repository URL from the page")
		fmt.Println("   • Should look like: https://github.com/YOUR-USERNAME/cursor-sync-bucket.git")
		fmt.Println()
		fmt.Println("💡 Why 'cursor-sync-bucket'?")
		fmt.Println("   • Clear purpose: stores your Cursor settings")
		fmt.Println("   • Avoids confusion with the cursor-sync tool itself")
		fmt.Println("   • Standard naming convention")
		fmt.Println()

		// Ask if user has created repository
		if !s.promptYesNo("Have you created your private repository?") {
			fmt.Println()
			fmt.Println("No problem! Take your time creating the repository.")
			fmt.Println("Remember: it MUST be private for security!")
			fmt.Println("Press Enter when ready to continue...")
			s.scanner.Scan()
			continue
		}

		// Get repository URL
		fmt.Println()
		fmt.Println("📝 Repository URL Examples:")
		fmt.Println("  ✅ https://github.com/johndoe/cursor-sync-bucket.git")
		fmt.Println("  ✅ https://github.com/alice/my-cursor-settings.git")
		fmt.Println("  ✅ git@github.com:bob/cursor-sync-bucket.git")
		fmt.Println()
		fmt.Print("🔗 Enter your repository URL: ")

		if !s.scanner.Scan() {
			fmt.Println("❌ Failed to read input. Let's try again...")
			continue
		}

		repoURL := strings.TrimSpace(s.scanner.Text())
		if repoURL == "" {
			fmt.Println("❌ Repository URL cannot be empty. Let's try again...")
			continue
		}

		// Basic URL format validation
		if !strings.Contains(repoURL, "github.com") {
			fmt.Printf("⚠️  This doesn't look like a GitHub URL: %s\n", repoURL)
			fmt.Println("Expected format: https://github.com/username/repo.git")
			if !s.promptYesNo("Continue anyway?") {
				continue
			}
		}

		// Validate repository accessibility and privacy
		fmt.Println("🔍 Validating repository...")
		if err := s.validateRepositoryURL(repoURL); err != nil {
			fmt.Printf("❌ Repository validation failed: %v\n", err)
			fmt.Println()
			fmt.Println("Common issues:")
			fmt.Println("• Repository doesn't exist or URL is incorrect")
			fmt.Println("• Repository is not accessible with your token")
			fmt.Println("• Token doesn't have 'repo' scope for private repositories")
			fmt.Println()
			fmt.Println("Let's try again...")
			continue
		}

		// Update config
		cfg.Repository.URL = repoURL

		// Branch configuration (optional)
		fmt.Println()
		fmt.Print("📂 Repository branch (press Enter for 'main'): ")
		if s.scanner.Scan() {
			branch := strings.TrimSpace(s.scanner.Text())
			if branch != "" {
				cfg.Repository.Branch = branch
			}
		}

		// Save configuration
		fmt.Println("💾 Saving configuration...")
		if err := s.saveConfig(cfg); err != nil {
			fmt.Printf("❌ Failed to save configuration: %v\n", err)
			fmt.Println("Let's try again...")
			continue
		}

		fmt.Println("✅ Repository configuration saved successfully!")
		fmt.Println("🔒 Repository privacy verified - your settings are secure!")
		break
	}

	return nil
}

// setupCursorInstallationPath handles interactive Cursor installation path configuration
func (s *SetupWizard) setupCursorInstallationPath() error {
	fmt.Println("📂 Cursor Installation Path Configuration")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()
	fmt.Println("cursor-sync can work with different IDE installations:")
	fmt.Println("• ✅ Cursor IDE (primary focus)")
	fmt.Println("• ✅ VS Code (also supported)")
	fmt.Println("• ✅ Custom installation paths")
	fmt.Println()

	// Load current config
	cfg, err := s.loadOrCreateConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Show default paths for different IDEs and operating systems
	fmt.Println("📍 Default Installation Paths:")
	fmt.Println()
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "darwin": // macOS
		fmt.Println("🍎 macOS:")
		fmt.Println("  • Cursor:  ~/Library/Application Support/Cursor")
		fmt.Println("  • VS Code: ~/Library/Application Support/Code")
		fmt.Println()
	case "linux":
		fmt.Println("🐧 Linux:")
		fmt.Println("  • Cursor:  ~/.config/Cursor")
		fmt.Println("  • VS Code: ~/.config/Code")
		fmt.Println()
	case "windows":
		fmt.Println("🪟 Windows:")
		fmt.Println("  • Cursor:  %APPDATA%\\Cursor")
		fmt.Println("  • VS Code: %APPDATA%\\Code")
		fmt.Println()
	}

	// Get the current default Cursor path
	defaultPath := cursor.GetDefaultCursorPath()
	if cfg.Cursor.ConfigPath != "" && cfg.Cursor.ConfigPath != defaultPath {
		fmt.Printf("📍 Current configured path: %s\n", cfg.Cursor.ConfigPath)
		if s.promptYesNo("Keep current path?") {
			return nil
		}
	}

	// Auto-detect existing installations
	fmt.Println("🔍 Auto-detecting IDE installations...")

	detectedPaths := s.detectIDEInstallations()
	if len(detectedPaths) > 0 {
		fmt.Println("✅ Found installations:")
		for i, path := range detectedPaths {
			fmt.Printf("  %d. %s\n", i+1, path.description)
		}
		fmt.Println()

		// Ask if user wants to use one of the detected paths
		if s.promptYesNo("Use one of the detected installations?") {
			for {
				fmt.Printf("Enter choice (1-%d): ", len(detectedPaths))
				if !s.scanner.Scan() {
					continue
				}
				choice := strings.TrimSpace(s.scanner.Text())

				// Handle numeric choice
				for i, path := range detectedPaths {
					if choice == fmt.Sprintf("%d", i+1) {
						cfg.Cursor.ConfigPath = path.path
						if err := s.saveConfig(cfg); err != nil {
							return fmt.Errorf("failed to save config: %w", err)
						}
						fmt.Printf("✅ Set IDE path to: %s\n", path.path)
						return nil
					}
				}
				fmt.Println("❌ Invalid choice. Please try again.")
			}
		}
	}

	// Manual path entry
	fmt.Println("📝 Enter custom installation path:")
	fmt.Printf("💡 Default (press Enter): %s\n", defaultPath)
	fmt.Print("🔗 Custom path: ")

	if !s.scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	customPath := strings.TrimSpace(s.scanner.Text())
	if customPath == "" {
		customPath = defaultPath
	}

	// Expand tilde in path
	if strings.HasPrefix(customPath, "~") {
		customPath = filepath.Join(home, customPath[1:])
	}

	// Validate the path
	if err := s.validateIDEPath(customPath); err != nil {
		fmt.Printf("⚠️  Path validation warning: %v\n", err)
		if !s.promptYesNo("Continue anyway?") {
			return s.setupCursorInstallationPath() // Retry
		}
	}

	// Save the configuration
	cfg.Cursor.ConfigPath = customPath
	if err := s.saveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✅ IDE path configured: %s\n", customPath)
	return nil
}

// IDEPath represents a detected IDE installation
type IDEPath struct {
	path        string
	description string
	ideType     string
}

// detectIDEInstallations auto-detects common IDE installations
func (s *SetupWizard) detectIDEInstallations() []IDEPath {
	var paths []IDEPath
	home, _ := os.UserHomeDir()

	// Common IDE paths by operating system
	var candidatePaths []IDEPath

	switch runtime.GOOS {
	case "darwin":
		candidatePaths = []IDEPath{
			{filepath.Join(home, "Library", "Application Support", "Cursor"), "Cursor IDE (macOS)", "cursor"},
			{filepath.Join(home, "Library", "Application Support", "Code"), "VS Code (macOS)", "vscode"},
			{filepath.Join(home, "Library", "Application Support", "Code - Insiders"), "VS Code Insiders (macOS)", "vscode-insiders"},
		}
	case "linux":
		candidatePaths = []IDEPath{
			{filepath.Join(home, ".config", "Cursor"), "Cursor IDE (Linux)", "cursor"},
			{filepath.Join(home, ".config", "Code"), "VS Code (Linux)", "vscode"},
			{filepath.Join(home, ".config", "Code - Insiders"), "VS Code Insiders (Linux)", "vscode-insiders"},
		}
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			candidatePaths = []IDEPath{
				{filepath.Join(appdata, "Cursor"), "Cursor IDE (Windows)", "cursor"},
				{filepath.Join(appdata, "Code"), "VS Code (Windows)", "vscode"},
				{filepath.Join(appdata, "Code - Insiders"), "VS Code Insiders (Windows)", "vscode-insiders"},
			}
		}
	}

	// Check which paths actually exist
	for _, candidate := range candidatePaths {
		if s.pathExists(candidate.path) {
			paths = append(paths, candidate)
		}
	}

	return paths
}

// pathExists checks if a path exists
func (s *SetupWizard) pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// validateIDEPath validates an IDE installation path
func (s *SetupWizard) validateIDEPath(path string) error {
	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}

	// Check if it looks like an IDE configuration directory
	userDir := filepath.Join(path, "User")
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return fmt.Errorf("doesn't look like an IDE config directory (missing User folder)")
	}

	return nil
}

// validateRepositoryURL validates the repository URL and checks privacy
func (s *SetupWizard) validateRepositoryURL(repoURL string) error {
	// Basic URL validation
	if !strings.Contains(repoURL, "github.com") {
		return fmt.Errorf("currently only GitHub repositories are supported")
	}

	// Check repository privacy if we have a token
	if auth.HasValidToken() {
		checker := privacy.NewRepositoryChecker()
		isPrivate, err := checker.CheckRepositoryPrivacy(repoURL)
		if err != nil {
			return fmt.Errorf("failed to verify repository privacy: %w", err)
		}

		if !isPrivate {
			fmt.Println("\n⚠️  WARNING: This appears to be a PUBLIC repository!")
			fmt.Println("Your Cursor settings may contain sensitive information like:")
			fmt.Println("  • API keys and tokens")
			fmt.Println("  • Personal configurations")
			fmt.Println("  • Workspace paths")
			fmt.Println()
			fmt.Println("🔒 RECOMMENDATION: Use a PRIVATE repository for security.")

			if !s.promptYesNo("Continue with this PUBLIC repository? (NOT recommended)") {
				return fmt.Errorf("repository rejected - use a private repository instead")
			}
		} else {
			fmt.Println("✅ Repository is private - good for security!")
		}
	}

	return nil
}

// loadOrCreateConfig loads existing config or creates a default one
func (s *SetupWizard) loadOrCreateConfig() (*config.Config, error) {
	// Try to load existing config
	cfg, err := config.Load()
	if err == nil {
		return cfg, nil
	}

	// Create default config if loading fails
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &config.Config{
		Repository: config.Repository{
			URL:       "",
			LocalPath: filepath.Join(home, ".cursor-sync", "settings"),
			Branch:    "main",
		},
		Sync: config.Sync{
			PullInterval:    5 * 60, // 5 minutes in seconds for YAML
			PushInterval:    5 * 60,
			WatchEnabled:    true,
			ConflictResolve: "newer",
		},
		Cursor: config.Cursor{
			ConfigPath: filepath.Join(home, "Library", "Application Support", "Cursor"),
			ExcludePaths: []string{
				"logs/", "CachedExtensions/", "CachedExtensionVSIXs/",
				"tmp/", "GPUCache/", "Crashpad/", "CachedData/",
				"User/workspaceStorage/", "User/History/",
			},
			IncludePaths: []string{},
		},
		Logging: config.Logging{
			Level:    "info",
			LogDir:   filepath.Join(home, ".cursor-sync", "logs"),
			MaxSize:  10,
			MaxDays:  30,
			Compress: true,
		},
	}, nil
}

// saveConfig saves the configuration to the config file
func (s *SetupWizard) saveConfig(cfg *config.Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".cursor-sync")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// validateAndSaveConfig validates and saves the current configuration
func (s *SetupWizard) validateAndSaveConfig(cfg *config.Config) error {
	if err := s.validateRepositoryURL(cfg.Repository.URL); err != nil {
		return err
	}
	return s.saveConfig(cfg)
}

// promptYesNo prompts for a yes/no question
func (s *SetupWizard) promptYesNo(question string) bool {
	for {
		fmt.Printf("%s (y/N): ", question)
		if !s.scanner.Scan() {
			return false
		}

		response := strings.ToLower(strings.TrimSpace(s.scanner.Text()))
		switch response {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		default:
			fmt.Println("Please enter 'y' for yes or 'n' for no.")
		}
	}
}
