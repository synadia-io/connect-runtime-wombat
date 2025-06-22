package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/rs/zerolog"
)

type contextKey string

const (
	// CorrelationIDKey is the context key for correlation IDs
	CorrelationIDKey contextKey = "correlation_id"
)

// GenerateCorrelationID generates a new correlation ID
func GenerateCorrelationID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return "fallback-" + hex.EncodeToString([]byte(string(rune(len(bytes)))))
	}
	return hex.EncodeToString(bytes)
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID retrieves the correlation ID from the context
func GetCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return ""
}

// LoggerWithCorrelation creates a logger with correlation ID from context
func LoggerWithCorrelation(ctx context.Context) zerolog.Logger {
	logger := InitLogger()
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		logger = logger.With().Str("correlation_id", correlationID).Logger()
	}
	return logger
}
