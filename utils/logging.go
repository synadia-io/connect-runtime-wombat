// Package utils provides utility functions for the runtime
package utils

import (
	"os"

	"github.com/rs/zerolog"
)

// InitLogger initializes a logger with the appropriate log level based on environment
func InitLogger() zerolog.Logger {
	// Set log level from environment
	level := zerolog.InfoLevel
	switch os.Getenv("CONNECT_LOG_LEVEL") {
	case "debug", "DEBUG":
		level = zerolog.DebugLevel
	case "trace", "TRACE":
		level = zerolog.TraceLevel
	case "warn", "WARN":
		level = zerolog.WarnLevel
	case "error", "ERROR":
		level = zerolog.ErrorLevel
	}

	// Configure logger
	logger := zerolog.New(os.Stderr).
		With().
		Timestamp().
		Str("service", "connect-runtime-wombat").
		Logger().
		Level(level)

	// Pretty print in development
	if os.Getenv("CONNECT_ENV") == "development" {
		logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	return logger
}

// GetLogLevel returns the current log level from environment
func GetLogLevel() string {
	level := os.Getenv("CONNECT_LOG_LEVEL")
	if level == "" {
		return "info"
	}
	return level
}
