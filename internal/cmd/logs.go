package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View cursor-sync logs",
	Long: `View cursor-sync logs from the current or previous days.

Examples:
  cursor-sync logs           # Show today's logs  
  cursor-sync logs --tail    # Follow logs in real-time
  cursor-sync logs --date 2024-01-15  # Show logs from specific date`,
	Run: func(cmd *cobra.Command, args []string) {
		tail, _ := cmd.Flags().GetBool("tail")
		date, _ := cmd.Flags().GetString("date")
		lines, _ := cmd.Flags().GetInt("lines")

		if err := viewLogs(tail, date, lines); err != nil {
			fmt.Printf("❌ Failed to view logs: %v\n", err)
		}
	},
}

func viewLogs(tail bool, date string, lines int) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	logsDir := filepath.Join(home, ".cursor-sync", "logs")

	// Determine log file
	var logFile string
	if date != "" {
		logFile = filepath.Join(logsDir, date+".log")
	} else {
		// Today's log
		today := time.Now().Format("2006-01-02")
		logFile = filepath.Join(logsDir, today+".log")
	}

	// Check if log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		fmt.Printf("📄 No logs found for %s\n", date)
		fmt.Printf("Log file: %s\n", logFile)
		return nil
	}

	fmt.Printf("📋 Viewing logs: %s\n", logFile)
	fmt.Println()

	if tail {
		// Follow logs in real-time
		fmt.Println("Following logs (press Ctrl+C to exit)...")
		fmt.Printf("tail -f %s\n", logFile)
	} else {
		// Show last N lines
		fmt.Printf("Showing last %d lines:\n", lines)
		fmt.Printf("tail -%d %s\n", lines, logFile)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolP("tail", "f", false, "Follow logs in real-time")
	logsCmd.Flags().StringP("date", "d", "", "Show logs from specific date (YYYY-MM-DD)")
	logsCmd.Flags().IntP("lines", "n", 50, "Number of lines to show")
}
