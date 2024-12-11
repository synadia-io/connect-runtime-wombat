package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/synadia-labs/vent/public/control"
	"github.com/synadia-labs/vent/public/runtime"
	"github.com/synadia-labs/vent/runtimes/wombat/compiler"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("usage: wombat <config>")
		os.Exit(1)
	}

	rt, err := runtime.FromEnv()
	if err != nil {
		panic(err)
	}

	// launching a workload will start the workload and block
	if err := rt.Launch(context.Background(), Run, args[0]); err != nil {
		panic(err)
	}
}

func Run(ctx context.Context, runtime *runtime.Runtime, cfg control.ConnectorConfig) error {
	if cfg.Steps == nil {
		return fmt.Errorf("invalid configuration: no steps")
	}

	art, err := compiler.Compile(*cfg.Steps)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":4195",
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
	signal.Notify(sigs, os.Interrupt, os.Kill)

	select {
	case <-sigs:
		log.Info().Msg("received signal, shutting down")
		server.Shutdown(ctx)
		stream.Stop(ctx)
	case err := <-streamChan:
		log.Info().Msg("stream stopped, shutting down")
		server.Shutdown(ctx)
		return err
	case err := <-httpChan:
		log.Info().Msg("http server stopped, shutting down")
		stream.Stop(ctx)
		return err
	}

	return nil
}
