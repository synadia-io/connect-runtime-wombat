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

	"github.com/rs/zerolog/log"
	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
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
	// Compile the Connect specification to Wombat YAML
	art, err := compiler.Compile(runtime, steps)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	// Create HTTP server for health and metrics endpoints
	// Using port 0 allows the OS to assign an available port
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":0",
		Handler: mux,
	}

	// Validate the compiled configuration and create the stream
	// The mux is passed to allow registration of HTTP endpoints
	stream, err := compiler.Validate(ctx, runtime, art, mux)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Run the stream in a goroutine and capture its completion
	streamChan := make(chan error, 1)
	go func() {
		streamChan <- stream.Run(ctx)
	}()

	// Run the HTTP server in a goroutine and capture its completion
	httpChan := make(chan error, 1)
	go func() {
		httpChan <- server.ListenAndServe()
	}()

	// Set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown trigger and coordinate cleanup
	select {
	case <-ctx.Done():
		// Context cancelled - initiate graceful shutdown
		log.Info().Msg("shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to stop stream")
		}
	case <-sigs:
		// OS signal received - initiate graceful shutdown
		log.Info().Msg("received signal, shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to stop stream")
		}
	case err := <-streamChan:
		// Stream completed or errored - shutdown HTTP server
		log.Info().Msg("stream stopped, shutting down")
		if shutdownErr := server.Shutdown(context.TODO()); shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("failed to shutdown server")
		}
		return err
	case err := <-httpChan:
		// HTTP server stopped - shutdown stream
		log.Info().Msg("http server stopped, shutting down")
		if stopErr := stream.Stop(context.TODO()); stopErr != nil {
			log.Error().Err(stopErr).Msg("failed to stop stream")
		}
		return err
	}

	return nil
}
