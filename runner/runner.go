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
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
)

func Run(ctx context.Context, runtime *runtime.Runtime, steps model.Steps) error {
	logger := utils.LoggerWithCorrelation(ctx)
	logger.Info().
		Str("namespace", runtime.Namespace).
		Str("instance", runtime.Instance).
		Str("connector", runtime.Connector).
		Msg("Starting wombat runner")

	logger.Debug().Msg("Compiling configuration")
	art, err := compiler.CompileWithContext(ctx, runtime, steps)
	if err != nil {
		logger.Error().Err(err).Msg("Compilation failed")
		return fmt.Errorf("compilation failed: %w", err)
	}

	logger.Debug().Int("config_bytes", len(art)).Msg("Configuration compiled successfully")

	logger.Debug().Msg("Setting up HTTP server")
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":0",
		Handler: mux,
	}

	logger.Debug().Msg("Validating configuration and creating stream")
	stream, err := compiler.Validate(ctx, runtime, art, mux)
	if err != nil {
		logger.Error().Err(err).Msg("Validation failed")
		return compiler.NewValidationError("configuration", "failed to validate and create stream", err)
	}

	logger.Info().Msg("Configuration validated, starting stream and HTTP server")

	streamChan := make(chan error, 1)
	go func() {
		logger.Debug().Msg("Starting wombat stream")
		streamChan <- stream.Run(ctx)
	}()

	httpChan := make(chan error, 1)
	go func() {
		logger.Debug().Msg("Starting HTTP server")
		httpChan <- server.ListenAndServe()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	logger.Info().Msg("Runner started, waiting for shutdown signal")

	select {
	case <-ctx.Done():
		logger.Info().Msg("Context cancelled, shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to stop stream")
		}
	case sig := <-sigs:
		logger.Info().Str("signal", sig.String()).Msg("Received signal, shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			logger.Error().Err(err).Msg("Failed to stop stream")
		}
	case err := <-streamChan:
		logger.Info().Err(err).Msg("Stream stopped, shutting down")
		if shutdownErr := server.Shutdown(context.TODO()); shutdownErr != nil {
			logger.Error().Err(shutdownErr).Msg("Failed to shutdown server")
		}
		if err != nil {
			return compiler.NewRuntimeError("stream", "stream execution failed", err)
		}
		return nil
	case err := <-httpChan:
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
