package test

import (
    "log/slog"

    "github.com/synadia-io/connect/runtime"
)

func Runtime() *runtime.Runtime {
    r := runtime.NewRuntime(slog.LevelInfo, "MY_NAMESPACE", "MY_CONNECTOR", "MY_INSTANCE")
    r.Logger = slog.Default()
    return r
}
