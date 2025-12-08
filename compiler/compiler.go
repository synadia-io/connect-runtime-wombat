// Package compiler provides functionality to transform Synadia Connect models
// into Wombat/Benthos YAML configurations.
//
// The compiler handles the translation of Connect's high-level abstractions
// (sources, sinks, producers, consumers, transformers) into Wombat's
// configuration format, including:
//   - Input/output configuration mapping
//   - Processor chain construction
//   - Metrics integration with NATS
//   - Field type and validation mapping
//
// The package uses a Fragment builder pattern to construct YAML configurations
// in a type-safe manner.
package compiler

import (
	"context"
	"fmt"
	"time"

	// Import custom NATS components for registration
	_ "github.com/synadia-io/connect-runtime-wombat/components"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
	"gopkg.in/yaml.v3"
)

const (
	// AccountMetricHeader is the header name for the account/namespace in metrics
	AccountMetricHeader = "account"
	// ConnectorMetricHeader is the header name for the connector ID in metrics
	ConnectorMetricHeader = "connector_id"
	// InstanceMetricHeader is the header name for the instance ID in metrics
	InstanceMetricHeader = "instance_id"
)

// metricsAPIPrefix generates the NEX-compatible metrics subject prefix
// for publishing metrics to NATS. The format follows the NEX feed pattern.
func metricsAPIPrefix(namespace string) string {
	return fmt.Sprintf("$NEX.FEED.%s.metrics", namespace)
}

// Compile transforms a Connect model specification into a Wombat YAML configuration.
// It handles the translation of sources/sinks or producers/consumers along with
// any transformers into the appropriate Wombat input/output/processor configuration.
//
// The function also configures metrics publishing to NATS if the runtime provides
// the necessary connection details.
//
// Parameters:
//   - rt: Runtime configuration containing component definitions and NATS connection details
//   - steps: The Connect model steps to compile (source/sink or producer/consumer with optional transformers)
//
// Returns:
//   - A YAML string containing the complete Wombat configuration
//   - An error if the specification is invalid or compilation fails
func Compile(rt *runtime.Runtime, steps model.Steps) (string, error) {
	return CompileWithContext(context.Background(), rt, steps)
}

func CompileWithContext(ctx context.Context, rt *runtime.Runtime, steps model.Steps) (string, error) {
	start := time.Now()
	logger := utils.LoggerWithCorrelation(ctx)
	logger.Debug().
		Str("namespace", rt.Namespace).
		Str("instance", rt.Instance).
		Str("connector", rt.Connector).
		Msg("Starting compilation")

	// Determine connector type for metrics
	connectorType := "unknown"
	if steps.Producer != nil && steps.Source != nil {
		connectorType = "inlet"
	} else if steps.Consumer != nil && steps.Sink != nil {
		connectorType = "outlet"
	}

	mainCfg := Frag()

	if rt.NatsUrl != "" && rt.Namespace != "" && rt.Instance != "" {
		logger.Debug().
			Str("nats_url", rt.NatsUrl).
			Str("metrics_subject", fmt.Sprintf("%s.%s", metricsAPIPrefix(rt.Namespace), rt.Instance)).
			Msg("Configuring NATS metrics")

		natsCfg := Frag().
			String("url", rt.NatsUrl).
			String("subject", fmt.Sprintf("%s.%s", metricsAPIPrefix(rt.Namespace), rt.Instance)).
			StringMap("headers", map[string]string{
				AccountMetricHeader:   rt.Namespace,
				ConnectorMetricHeader: rt.Connector,
				InstanceMetricHeader:  rt.Instance,
			})

		if rt.NatsJwt != "" && rt.NatsSeed != "" {
			logger.Debug().Msg("Adding NATS JWT authentication")
			natsCfg.
				String("jwt", rt.NatsJwt).
				String("seed", rt.NatsSeed)
		}

		mainCfg.
			Fragment("metrics", Frag().
				Fragment("nats", natsCfg))
	}

	var err error
	if steps.Producer != nil && steps.Source != nil {
		logger.Debug().Msg("Compiling inlet connector (source -> producer)")
		producer, err := compileProducer(*steps.Producer)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to compile producer")
			RecordCompilationMetrics(start, false, connectorType)
			return "", NewCompilationError("output", "producer", "failed to compile producer", err)
		}

		mainCfg.Fragment("input", compileSource(*steps.Source, steps.Transformer))
		mainCfg.Fragment("output", producer)
	} else if steps.Consumer != nil && steps.Sink != nil {
		logger.Debug().Msg("Compiling outlet connector (consumer -> sink)")
		consumer, err := compileConsumer(*steps.Consumer, steps.Transformer)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to compile consumer")
			RecordCompilationMetrics(start, false, connectorType)
			return "", NewCompilationError("input", "consumer", "failed to compile consumer", err)
		}

		mainCfg.Fragment("input", consumer)
		mainCfg.Fragment("output", compileSink(*steps.Sink))
	} else {
		logger.Error().Msg("Invalid steps configuration: missing required components")
		RecordCompilationMetrics(start, false, connectorType)
		return "", NewCompilationError("validation", "steps", "invalid steps configuration: missing required components", nil)
	}

	logger.Debug().Msg("Marshaling configuration to YAML")
	b, err := yaml.Marshal(mainCfg)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal configuration")
		RecordCompilationMetrics(start, false, connectorType)
		return "", NewCompilationError("marshal", "yaml", "failed to marshal configuration", err)
	}

	RecordCompilationMetrics(start, true, connectorType)
	logger.Debug().Int("config_length", len(b)).Msg("Compilation completed successfully")
	return string(b), nil
}
