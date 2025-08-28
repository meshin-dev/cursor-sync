package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"cursor-sync/internal/config"
	"cursor-sync/internal/logger"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  "Show the current status of the cursor-sync daemon",
	Run: func(cmd *cobra.Command, args []string) {
		status, err := getDaemonStatus()
		if err != nil {
			logger.Error("Failed to get daemon status: %v", err)
			return
		}

		fmt.Printf("Cursor Sync Status: %s\n", status)

		// Show additional info if running
		if status == "running" {
			cfg, err := config.Load()
			if err == nil {
				fmt.Printf("Repository: %s\n", cfg.Repository.URL)
				fmt.Printf("Pull interval: %v\n", cfg.Sync.PullInterval)
				fmt.Printf("Push interval: %v\n", cfg.Sync.PushInterval)
			}
		}
	},
}

// pauseCmd represents the pause command
var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause sync daemon",
	Long:  "Temporarily pause the cursor-sync daemon without stopping it completely",
	Run: func(cmd *cobra.Command, args []string) {
		if err := controlDaemon("pause"); err != nil {
			logger.Error("Failed to pause daemon: %v", err)
			return
		}
		fmt.Println("✅ Cursor Sync paused")
	},
}

// resumeCmd represents the resume command
var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume sync daemon",
	Long:  "Resume the paused cursor-sync daemon",
	Run: func(cmd *cobra.Command, args []string) {
		if err := controlDaemon("resume"); err != nil {
			logger.Error("Failed to resume daemon: %v", err)
			return
		}
		fmt.Println("✅ Cursor Sync resumed")
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop sync daemon",
	Long:  "Stop the cursor-sync daemon completely",
	Run: func(cmd *cobra.Command, args []string) {
		if err := controlDaemon("stop"); err != nil {
			logger.Error("Failed to stop daemon: %v", err)
			return
		}
		fmt.Println("✅ Cursor Sync stopped")
	},
}

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start sync daemon",
	Long:  "Start the cursor-sync daemon and perform initial sync",
	Run: func(cmd *cobra.Command, args []string) {
		if err := controlDaemon("start"); err != nil {
			logger.Error("Failed to start daemon: %v", err)
			return
		}
		fmt.Println("✅ Cursor Sync started")
		fmt.Println("🔄 Initial sync will be performed automatically")
                fmt.Println("📋 Check logs with: cursor-sync logs")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(startCmd)
}

func getDaemonStatus() (string, error) {
	// Check if LaunchAgent is loaded
	cmd := exec.Command("launchctl", "list", "com.user.cursorsync")
	output, err := cmd.Output()
	if err != nil {
		return "stopped", nil
	}

	if len(output) > 0 {
		return "running", nil
	}

	return "stopped", nil
}

func controlDaemon(action string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := fmt.Sprintf("%s/Library/LaunchAgents/com.user.cursorsync.plist", home)

	switch action {
	case "start":
		return exec.Command("launchctl", "load", plistPath).Run()
	case "stop":
		return exec.Command("launchctl", "unload", plistPath).Run()
	case "pause":
		// Create pause file
		pauseFile := fmt.Sprintf("%s/.cursor-sync/paused", home)
		file, err := os.Create(pauseFile)
		if err != nil {
			return err
		}
		file.Close()
		logger.Info("Created pause file at " + pauseFile)
		return nil
	case "resume":
		// Remove pause file
		pauseFile := fmt.Sprintf("%s/.cursor-sync/paused", home)
		return os.Remove(pauseFile)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
