// Package runner provides the main execution logic for the Wombat runtime.
//
// The runner handles:
//   - Concurrent management of the data stream and HTTP server
//   - Graceful shutdown on signals or errors
package runner

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/synadia-io/connect-runtime-wombat/components"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-io/connect-runtime-wombat/utils"
)

// Run executes the main runtime logic for a Wombat configuration.
//
// The function manages two concurrent components:
//   - A Wombat stream that processes data according to the configuration
//   - An HTTP server for health checks and metrics endpoints
//
// Shutdown is triggered by:
//   - Context cancellation
//   - OS signals (SIGINT, SIGTERM)
//   - Stream completion or error
//   - HTTP server error
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - configYAML: The Wombat configuration as YAML/JSON string
//
// Returns an error if validation or execution fails.
func Run(ctx context.Context, configYAML string) error {
	logger := utils.LoggerWithCorrelation(ctx)
	logger.Info().Msg("Starting wombat runner")

	// Create and configure stream builder
	logger.Debug().Msg("Configuring stream builder")
	builder := service.NewStreamBuilder()
	builder.DisableLinting()

	// Parse the configuration
	if err := builder.SetYAML(configYAML); err != nil {
		logger.Error().Err(err).Msg("Failed to parse configuration")
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	logger.Debug().Int("config_bytes", len(configYAML)).Msg("Configuration parsed successfully")

	// Create HTTP server for health and metrics endpoints
	// Using port 0 allows the OS to assign an available port
	logger.Debug().Msg("Setting up HTTP server")
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":0",
		Handler: mux,
	}

	// Build the stream
	logger.Debug().Msg("Building stream")
	stream, err := builder.Build()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to build stream")
		return fmt.Errorf("failed to build stream: %w", err)
	}

	logger.Info().Msg("Configuration validated, starting stream and HTTP server")

	// Run the stream in a goroutine and capture its completion
	streamChan := make(chan error, 1)
	go func() {
		logger.Debug().Msg("Starting wombat stream")
		streamChan <- stream.Run(ctx)
	}()

	// Run the HTTP server in a goroutine and capture its completion
	httpChan := make(chan error, 1)
	go func() {
		logger.Debug().Msg("Starting HTTP server")
		httpChan <- server.ListenAndServe()
	}()

	// Set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	logger.Info().Msg("Runner started, waiting for shutdown signal")

	// Wait for shutdown trigger and coordinate cleanup
	select {
	case <-ctx.Done():
		// Context cancelled - initiate graceful shutdown
		logger.Info().Msg("Context cancelled, shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to stop stream")
		}
	case sig := <-sigs:
		// OS signal received - initiate graceful shutdown
		logger.Info().Str("signal", sig.String()).Msg("Received signal, shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to stop stream")
		}
	case err := <-streamChan:
		// Stream completed or errored - shutdown HTTP server
		logger.Info().Err(err).Msg("Stream stopped, shutting down")
		if shutdownErr := server.Shutdown(context.TODO()); shutdownErr != nil {
			logger.Error().Err(shutdownErr).Msg("Failed to shutdown server")
		}
		if err != nil {
			return fmt.Errorf("stream execution failed: %w", err)
		}
		return nil
	case err := <-httpChan:
		// HTTP server stopped - shutdown stream
		logger.Info().Err(err).Msg("HTTP server stopped, shutting down")
		if stopErr := stream.Stop(context.TODO()); stopErr != nil {
			logger.Error().Err(stopErr).Msg("Failed to stop stream")
		}
		if err != nil {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil
	}

	logger.Info().Msg("Shutdown completed")
	return nil
}
