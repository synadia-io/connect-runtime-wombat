package runner

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func Run(ctx context.Context, runtime *runtime.Runtime, steps model.Steps) error {
	art, err := compiler.Compile(runtime, steps)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":0",
		Handler: mux,
	}

	stream, err := compiler.Validate(ctx, runtime, art, mux)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	streamChan := make(chan error, 1)
	go func() {
		streamChan <- stream.Run(ctx)
	}()

	httpChan := make(chan error, 1)
	go func() {
		httpChan <- server.ListenAndServe()
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to stop stream")
		}
	case <-sigs:
		log.Info().Msg("received signal, shutting down")
		if err := server.Shutdown(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to shutdown server")
		}
		if err := stream.Stop(context.TODO()); err != nil {
			log.Error().Err(err).Msg("failed to stop stream")
		}
	case err := <-streamChan:
		log.Info().Msg("stream stopped, shutting down")
		if shutdownErr := server.Shutdown(context.TODO()); shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("failed to shutdown server")
		}
		return err
	case err := <-httpChan:
		log.Info().Msg("http server stopped, shutting down")
		if stopErr := stream.Stop(context.TODO()); stopErr != nil {
			log.Error().Err(stopErr).Msg("failed to stop stream")
		}
		return err
	}

	return nil
}
