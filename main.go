// Package main provides the entry point for the connect-runtime-wombat executable.
// This runtime wraps Wombat (a Benthos fork) to provide its extensive component
// ecosystem within the Synadia Connect platform.
//
// Usage:
//
//	connect-runtime-wombat --config=<json>
//
// Where <json> is Wombat/Benthos configuration in JSON format.
//
// The runtime injects NATS credentials from environment variables
// into any NATS components found in the configuration.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/synadia-io/connect-runtime-wombat/runner"
	"github.com/synadia-io/connect-runtime-wombat/utils"
)

var (
	Version        = "dev"
	CommitHash     = "unknown"
	BuildTimestamp = "unknown"
)

// main is the entry point for the connect-runtime-wombat executable.
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

	if len(os.Args) != 2 || !strings.HasPrefix(os.Args[1], "--config=") {
		logger.Error().Msg("Invalid arguments provided")
		fmt.Println("usage: --config=<json>")
		os.Exit(1)
	}

	configJSON := os.Args[1][9:] // Skip "--config="
	logger.Debug().Int("config_length", len(configJSON)).Msg("Parsing configuration")

	if err := runWithConfig(ctx, configJSON); err != nil {
		logger.Error().Err(err).Msg("Failed to run connector")
		os.Exit(1)
	}
}

// runWithConfig parses the JSON config, injects NEX credentials, and runs the stream.
func runWithConfig(ctx context.Context, configJSON string) error {
	logger := utils.LoggerWithCorrelation(ctx)

	var config map[string]any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	injectNexCredentials(ctx, config)

	finalConfig, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	logger.Info().Msg("Configuration processed, launching runner")

	return runner.Run(ctx, string(finalConfig))
}

// injectNexCredentials adds NEX workload NATS credentials to the config if applicable.
func injectNexCredentials(ctx context.Context, config map[string]any) {
	logger := utils.LoggerWithCorrelation(ctx)

	natsServers := os.Getenv("NEX_WORKLOAD_NATS_SERVERS")
	nkeySeed := os.Getenv("NEX_WORKLOAD_NATS_NKEY")
	b64JWT := os.Getenv("NEX_WORKLOAD_NATS_B64_JWT")

	// Check if input or output uses NATS variants
	inputSection, _ := config["input"].(map[string]any)
	outputSection, _ := config["output"].(map[string]any)

	natsTypes := []string{"nats", "nats_jetstream", "nats_kv"}
	var inputNatsType, outputNatsType string
	var inputNatsConfig, outputNatsConfig map[string]any

	for _, t := range natsTypes {
		if cfg, ok := inputSection[t].(map[string]any); ok {
			inputNatsType = t
			inputNatsConfig = cfg
			break
		}
	}
	for _, t := range natsTypes {
		if cfg, ok := outputSection[t].(map[string]any); ok {
			outputNatsType = t
			outputNatsConfig = cfg
			break
		}
	}

	hasNats := inputNatsType != "" || outputNatsType != ""
	if hasNats {
		if natsServers == "" {
			logger.Warn().Msg("NEX_WORKLOAD_NATS_SERVERS not set, NATS connector may fail to connect")
		}
		if nkeySeed == "" && b64JWT == "" {
			logger.Warn().Msg("NEX_WORKLOAD_NATS_NKEY and NEX_WORKLOAD_NATS_B64_JWT not set, NATS auth disabled")
		}
	}

	// Inject credentials into NATS configs
	if inputNatsConfig != nil {
		if natsServers != "" {
			inputNatsConfig["urls"] = []string{natsServers}
		}
		injectNatsAuth(ctx, inputNatsConfig, nkeySeed, b64JWT)
	}
	if outputNatsConfig != nil {
		if natsServers != "" {
			outputNatsConfig["urls"] = []string{natsServers}
		}
		injectNatsAuth(ctx, outputNatsConfig, nkeySeed, b64JWT)
	}
}

// injectNatsAuth adds auth credentials to a NATS config section.
func injectNatsAuth(ctx context.Context, cfg map[string]any, nkeySeed, b64JWT string) {
	if nkeySeed == "" && b64JWT == "" {
		return
	}

	logger := utils.LoggerWithCorrelation(ctx)

	auth, ok := cfg["auth"].(map[string]any)
	if !ok {
		auth = make(map[string]any)
		cfg["auth"] = auth
	}

	if nkeySeed != "" {
		auth["user_nkey_seed"] = nkeySeed
	}
	if b64JWT != "" {
		jwt, err := base64.StdEncoding.DecodeString(b64JWT)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to decode JWT")
			os.Exit(1)
		}
		auth["user_jwt"] = string(jwt)
	}
}
