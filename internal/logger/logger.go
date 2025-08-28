package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init initializes the logger
func Init(verbose bool) {
	log = logrus.New()

	// Set log level
	if verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// InitWithConfig initializes the logger with configuration
func InitWithConfig(level, logDir string, verbose bool) error {
	log = logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	if verbose {
		logLevel = logrus.DebugLevel
	}

	log.SetLevel(logLevel)

	// Set formatter
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Setup file logging if log directory is provided
	if logDir != "" {
		if err := setupFileLogging(logDir); err != nil {
			return fmt.Errorf("failed to setup file logging: %w", err)
		}
	}

	return nil
}

func setupFileLogging(logDir string) error {
	// Create log directory
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create daily log directory
	today := time.Now().Format("2006-01-02")
	dailyLogDir := filepath.Join(logDir, today)
	if err := os.MkdirAll(dailyLogDir, 0755); err != nil {
		return fmt.Errorf("failed to create daily log directory: %w", err)
	}

	// Create log file
	logFile := filepath.Join(dailyLogDir, "cursor-sync.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Set output to both file and stdout
	log.SetOutput(file)

	// Clean up old logs
	go cleanupOldLogs(logDir, 30)

	return nil
}

func cleanupOldLogs(logDir string, maxDays int) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -maxDays)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse directory name as date
		date, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue
		}

		if date.Before(cutoff) {
			oldDir := filepath.Join(logDir, entry.Name())
			os.RemoveAll(oldDir)
			log.Debugf("Cleaned up old log directory: %s", oldDir)
		}
	}
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if log != nil {
		log.Debugf(format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if log != nil {
		log.Infof(format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if log != nil {
		log.Warnf(format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if log != nil {
		log.Errorf(format, args...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	if log != nil {
		log.Fatalf(format, args...)
	} else {
		fmt.Printf("FATAL: "+format+"\n", args...)
		os.Exit(1)
	}
}

// WithField returns a logger with a field
func WithField(key string, value interface{}) *logrus.Entry {
	if log != nil {
		return log.WithField(key, value)
	}
	return nil
}

// WithFields returns a logger with fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	if log != nil {
		return log.WithFields(fields)
	}
	return nil
}
