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
	"fmt"

	// Import custom NATS components for registration
	_ "github.com/synadia-io/connect-runtime-wombat/components"
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
	mainCfg := Frag()

	if rt.NatsUrl != "" && rt.Namespace != "" && rt.Instance != "" {
		natsCfg := Frag().
			String("url", rt.NatsUrl).
			String("subject", fmt.Sprintf("%s.%s", metricsAPIPrefix(rt.Namespace), rt.Instance)).
			StringMap("headers", map[string]string{
				AccountMetricHeader:   rt.Namespace,
				ConnectorMetricHeader: rt.Connector,
				InstanceMetricHeader:  rt.Instance,
			})

		if rt.NatsJwt != "" && rt.NatsSeed != "" {
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
		producer, err := compileProducer(*steps.Producer)
		if err != nil {
			return "", fmt.Errorf("output: %w", err)
		}

		mainCfg.Fragment("input", compileSource(*steps.Source, steps.Transformer))
		mainCfg.Fragment("output", producer)
	} else if steps.Consumer != nil && steps.Sink != nil {
		consumer, err := compileConsumer(*steps.Consumer, steps.Transformer)
		if err != nil {
			return "", fmt.Errorf("source: %w", err)
		}

		mainCfg.Fragment("input", consumer)
		mainCfg.Fragment("output", compileSink(*steps.Sink))
	} else {
		return "", fmt.Errorf("invalid steps")
	}

	b, err := yaml.Marshal(mainCfg)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	return string(b), nil
}
