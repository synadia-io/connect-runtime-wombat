// Package runner provides the main execution logic for the Wombat runtime.
// It orchestrates the compilation, validation, and execution of Connect
// specifications as Wombat data pipelines.
//
// The runner handles:
//   - Compilation of Connect models to Wombat configurations
//   - Validation of the generated configurations
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

	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/v2/model"
	"github.com/synadia-io/connect/v2/runtime"
)

// Run executes the main runtime logic for a Connect specification.
// It compiles the specification to Wombat format, validates it, and runs
// the resulting data pipeline.
//
// The function manages two concurrent components:
//   - A Wombat stream that processes data according to the specification
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
//   - runtime: Runtime configuration including component definitions
//   - steps: The Connect model steps to execute
//
// Returns an error if compilation, validation, or execution fails.
func Run(ctx context.Context, runtime *runtime.Runtime, steps model.Steps) error {
	logger := utils.LoggerWithCorrelation(ctx)
	logger.Info().
		Str("namespace", runtime.Namespace).
		Str("instance", runtime.Instance).
		Str("connector", runtime.Connector).
		Msg("Starting wombat runner")

	// Compile the Connect specification to Wombat YAML
	logger.Debug().Msg("Compiling configuration")
	art, err := compiler.CompileWithContext(ctx, runtime, steps)
	if err != nil {
		logger.Error().Err(err).Msg("Compilation failed")
		return fmt.Errorf("compilation failed: %w", err)
	}

	logger.Debug().Int("config_bytes", len(art)).Msg("Configuration compiled successfully")

	// Create HTTP server for health and metrics endpoints
	// Using port 0 allows the OS to assign an available port
	logger.Debug().Msg("Setting up HTTP server")
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":0",
		Handler: mux,
	}

	// Validate the compiled configuration and create the stream
	// The mux is passed to allow registration of HTTP endpoints
	logger.Debug().Msg("Validating configuration and creating stream")
	stream, err := compiler.Validate(ctx, runtime, art, mux)
	if err != nil {
		logger.Error().Err(err).Msg("Validation failed")
		return compiler.NewValidationError("configuration", "failed to validate and create stream", err)
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
			return compiler.NewRuntimeError("stream", "stream execution failed", err)
		}
		return nil
	case err := <-httpChan:
		// HTTP server stopped - shutdown stream
		logger.Info().Err(err).Msg("HTTP server stopped, shutting down")
		if stopErr := stream.Stop(context.TODO()); stopErr != nil {
			logger.Error().Err(stopErr).Msg("Failed to stop stream")
		}
		if err != nil {
			return compiler.NewRuntimeError("http_server", "HTTP server failed", err)
		}
		return nil
	}

	logger.Info().Msg("Shutdown completed")
	return nil
}
