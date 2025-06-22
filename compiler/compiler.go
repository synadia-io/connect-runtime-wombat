package compiler

import (
	"context"
	"fmt"
	"time"

	_ "github.com/synadia-io/connect-runtime-wombat/components"
	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
	"gopkg.in/yaml.v3"
)

const (
	AccountMetricHeader   = "account"
	ConnectorMetricHeader = "connector_id"
	InstanceMetricHeader  = "instance_id"
)

// metricsAPIPrefix generates the NEX-compatible metrics subject prefix
func metricsAPIPrefix(namespace string) string {
	return fmt.Sprintf("$NEX.FEED.%s.metrics", namespace)
}

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
	logger.Info().Int("config_length", len(b)).Msg("Compilation completed successfully")
	return string(b), nil
}
