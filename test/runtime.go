package test

import (
	"log/slog"

	"github.com/synadia-io/connect/runtime"
)

func Runtime() *runtime.Runtime {
	r := runtime.NewRuntime("dummy-account", "my-connector-id", "my-deployment-id", "my-instance-id", slog.LevelInfo)
	r.Logger = slog.Default()
	return r
}
