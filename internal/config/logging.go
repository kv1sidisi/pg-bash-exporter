package config

import (
	"io"
	"log"
	"log/slog"
	"os"
)

// SetupLogger sets up logger.
func SetupLogger(cfg Logging) {
	var logOutput io.Writer

	if cfg.Path == "" {
		logOutput = os.Stdout
	} else {
		file, err := os.OpenFile(cfg.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		logOutput = file
	}

	var logLevel slog.Level

	switch cfg.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(logOutput, &slog.HandlerOptions{Level: logLevel})

	logger := slog.New(handler)

	slog.SetDefault(logger)
}
