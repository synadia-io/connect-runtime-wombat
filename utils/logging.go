package utils

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// InitLogger initializes and returns a configured zerolog logger.
// The log level can be controlled via the CONNECT_LOG_LEVEL environment variable.
// Valid values: debug, info, warn, error. Defaults to info.
func InitLogger() zerolog.Logger {
	// Set output to stdout with pretty printing for development
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()

	// Configure log level based on environment variable
	switch os.Getenv("CONNECT_LOG_LEVEL") {
	case "debug", "DEBUG":
		logger = logger.Level(zerolog.DebugLevel)
	case "warn", "WARN":
		logger = logger.Level(zerolog.WarnLevel)
	case "error", "ERROR":
		logger = logger.Level(zerolog.ErrorLevel)
	default:
		logger = logger.Level(zerolog.InfoLevel)
	}

	return logger
}
