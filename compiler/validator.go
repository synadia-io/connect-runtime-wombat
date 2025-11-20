package compiler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/rs/zerolog/log"

	// Import custom NATS components for registration
	_ "github.com/synadia-io/connect-runtime-wombat/components"
	"github.com/synadia-io/connect/v2/runtime"
)

// Validate takes a compiled Wombat YAML configuration string and validates it
// using the Benthos service builder. If validation succeeds, it returns a
// ready-to-run stream.
//
// The function performs the following steps:
//  1. Creates a new Benthos stream builder
//  2. Configures the builder with the runtime logger and HTTP mux
//  3. Parses and validates the YAML configuration
//  4. Logs the configuration in base64 format for debugging
//  5. Builds and returns the stream
//
// Parameters:
//   - ctx: Context for cancellation (currently unused but reserved for future use)
//   - runtime: Runtime configuration containing the logger
//   - code: The compiled YAML configuration string
//   - mux: HTTP multiplexer for registering health and metrics endpoints
//
// Returns:
//   - A configured Benthos stream ready to run
//   - An error if the configuration is invalid
func Validate(ctx context.Context, runtime *runtime.Runtime, code string, mux *http.ServeMux) (*service.Stream, error) {
	sb := service.NewStreamBuilder()
	sb.SetLogger(runtime.Logger)
	sb.SetHTTPMux(mux)

	if err := sb.SetYAML(code); err != nil {
		return nil, fmt.Errorf("invalid artifact: %w", err)
	}

	y, _ := sb.AsYAML()
	if y != "" {
		log.Info().Msgf("stream def: %s", base64.StdEncoding.EncodeToString([]byte(y)))
	}

	return sb.Build()
}
