package main

import (
	"context"
	"fmt"
	"os"

	"github.com/synadia-io/connect-runtime-wombat/runner"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/runtime"
)

func main() {
	// Generate correlation ID for this application instance
	correlationID := utils.GenerateCorrelationID()
	ctx := utils.WithCorrelationID(context.Background(), correlationID)

	logger := utils.LoggerWithCorrelation(ctx)
	logger.Info().Msg("Starting connect-runtime-wombat")

	args := os.Args[1:]
	if len(args) != 1 {
		logger.Error().Msg("Invalid arguments provided")
		fmt.Println("usage: wombat <config>")
		os.Exit(1)
	}

	logger.Debug().Str("config", args[0]).Msg("Parsing configuration")

	rt, err := runtime.FromEnv()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize runtime from environment")
		panic(err)
	}

	logger.Info().Msg("Runtime initialized successfully")

	// launching a workload will start the workload and block
	logger.Info().Str("config", args[0]).Msg("Launching workload")
	if err := rt.Launch(ctx, runner.Run, args[0]); err != nil {
		logger.Error().Err(err).Msg("Failed to launch workload")
		panic(err)
	}
}
