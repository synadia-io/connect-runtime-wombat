// Package main provides the entry point for the connect-runtime-wombat executable.
// This runtime wraps Wombat (a Benthos fork) to provide its extensive component
// ecosystem within the Synadia Connect platform.
//
// Usage:
//
//	connect-runtime-wombat <config>
//
// Where <config> is the path to a JSON configuration file containing the
// connector specification.
//
// The runtime expects certain environment variables to be set by the Connect
// platform for proper operation. See runtime.FromEnv() for details.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/synadia-io/connect-runtime-wombat/runner"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/runtime"
)

var (
	Version        = "dev"
	CommitHash     = "unknown"
	BuildTimestamp = "unknown"
)

// main is the entry point for the connect-runtime-wombat executable.
// It expects exactly one argument: the path to a configuration file.
// The runtime configuration is loaded from environment variables set by
// the Connect platform.
func main() {
	// Generate correlation ID for this application instance
	correlationID := utils.GenerateCorrelationID()
	ctx := utils.WithCorrelationID(context.Background(), correlationID)

	logger := utils.LoggerWithCorrelation(ctx)
	logger.Info().
		Str("version", Version).
		Str("commit", CommitHash).
		Str("built", BuildTimestamp).
		Msg("Starting connect-runtime-wombat")

	args := os.Args[1:]
	if len(args) != 1 {
		logger.Error().Msg("Invalid arguments provided")
		fmt.Println("usage: wombat <config>")
		os.Exit(1)
	}

	logger.Debug().Str("config", args[0]).Msg("Parsing configuration")

	// Initialize runtime from environment variables
	// This includes NATS connection details and other platform configuration
	rt, err := runtime.FromEnv()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize runtime from environment")
		os.Exit(1)
	}

	preFlightErr := preFlightCheck(rt)
	if preFlightErr != nil {
		logger.Warn().Err(preFlightErr).Msg("Could not retrieve config required for metrics")
	}

	logger.Info().Msg("Runtime initialized successfully")

	// Launch the workload with the provided configuration
	// This will compile the Connect specification to Wombat format,
	// start the data pipeline, and block until completion or error
	logger.Info().Str("config", args[0]).Msg("Launching workload")
	if err := rt.Launch(ctx, runner.Run, args[0]); err != nil {
		logger.Error().Err(err).Msg("Failed to launch workload")
		os.Exit(1)
	}
}

// preFlightCheck checks if the runtime has all required values for metrics configuration
func preFlightCheck(rt *runtime.Runtime) error {
	var emptyFields []string

	if rt.NatsUrl == "" {
		emptyFields = append(emptyFields, "NatsUrl")
	}
	if rt.Namespace == "" {
		emptyFields = append(emptyFields, "Namespace")
	}
	if rt.Instance == "" {
		emptyFields = append(emptyFields, "Instance")
	}

	if len(emptyFields) > 0 {
		return fmt.Errorf("the following required field(s) are empty: %s", strings.Join(emptyFields, ", "))
	}

	return nil
}
