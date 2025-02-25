package test

import (
    "log/slog"

    "github.com/synadia-io/connect/runtime"
)

func Runtime() *runtime.Runtime {
    r := runtime.NewRuntime(slog.LevelInfo)
    r.Logger = slog.Default()
    return r
}
