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
)

func Run(ctx context.Context, runtime *runtime.Runtime, steps model.Steps) error {
    art, err := compiler.Compile(runtime, steps)
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
    case <-ctx.Done():
        log.Info().Msg("shutting down")
        server.Shutdown(context.TODO())
        stream.Stop(context.TODO())
    case <-sigs:
        log.Info().Msg("received signal, shutting down")
        server.Shutdown(context.TODO())
        stream.Stop(context.TODO())
    case err := <-streamChan:
        log.Info().Msg("stream stopped, shutting down")
        server.Shutdown(context.TODO())
        return err
    case err := <-httpChan:
        log.Info().Msg("http server stopped, shutting down")
        stream.Stop(context.TODO())
        return err
    }

    return nil
}
