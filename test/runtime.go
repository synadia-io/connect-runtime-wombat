package test

import (
	"github.com/synadia-labs/vent/public/runtime"
	"log/slog"
)

func Runtime() *runtime.Runtime {
	r := runtime.NewRuntime("dummy-account", "my-deployment-id", "my-exec-id", slog.LevelInfo)
	r.Logger = slog.Default()
	return r
}
