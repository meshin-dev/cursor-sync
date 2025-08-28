package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"cursor-sync/internal/config"
	"cursor-sync/internal/daemon"
	"cursor-sync/internal/logger"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the sync daemon",
	Long: `Run the cursor-sync daemon which watches for file changes in the Cursor
configuration directory and automatically syncs them with the remote Git repository.

The daemon will:
- Watch for file changes in real-time
- Sync changes at configured intervals
- Handle conflicts by preferring newer commits
- Log all activities with detailed information`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting Cursor Sync daemon...")

		cfg, err := config.Load()
		if err != nil {
			logger.Fatal("Failed to load configuration: %v", err)
		}

		// Create daemon instance
		d, err := daemon.New(cfg)
		if err != nil {
			logger.Fatal("Failed to create daemon: %v", err)
		}

		// Setup signal handling for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigChan
			logger.Info("Received shutdown signal, stopping daemon...")
			cancel()
		}()

		// Start daemon
		if err := d.Start(ctx); err != nil {
			logger.Fatal("Daemon failed: %v", err)
		}

		logger.Info("Daemon stopped")
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
