package test

import (
	"github.com/synadia-io/connect/v2/runtime"
)

func Runtime(opts ...runtime.Opt) *runtime.Runtime {
	r := runtime.NewRuntime(
		runtime.WithNamespace("MY_NAMESPACE"),
		runtime.WithGroup("MY_CONNECTOR"),
		runtime.WithInstance("MY_INSTANCE"),
	)

	for _, opt := range opts {
		opt(r)
	}

	return r
}
