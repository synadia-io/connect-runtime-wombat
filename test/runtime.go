package test

import (
    "log/slog"
    "os"

    "github.com/synadia-io/connect/runtime"
)

func Runtime() *runtime.Runtime {
    r := runtime.NewRuntime(slog.LevelInfo, "MY_NAMESPACE", "MY_CONNECTOR", "MY_INSTANCE")

    os.Setenv(runtime.NamespaceEnvVar, r.Namespace)
    os.Setenv(runtime.InstanceEnvVar, r.Instance)
    os.Setenv(runtime.GroupEnvVar, r.Connector)

    r.Logger = slog.Default()
    return r
}
