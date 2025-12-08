package syntax

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
	"gopkg.in/yaml.v3"
)

const NatsUrl = "nats://localhost:4222"
const NatsJWT = "ey...."
const NatsSeed = "SU...."

type TesterConfig struct {
	// DumpOnErrorDirectory is the directory to dump artifacts to when an error occurs during testing. If empty, no artifacts are dumped on error
	DumpOnErrorDirectory string

	// DumpDirectory is the directory to dump all compiled artifacts to. If empty, no artifacts are dumped
	DumpDirectory string
}

func NewTester(cfg TesterConfig) *Tester {
	return &Tester{
		cfg: cfg,
		rt: &runtime.Runtime{
			Namespace: "test-namespace",
			Instance:  "test-instance",
			Connector: "test-connector",
			NatsUrl:   NatsUrl,
			NatsJwt:   NatsJWT,
			NatsSeed:  NatsSeed,
			Logger:    slog.Default(),
			LogLevel:  slog.LevelError,
		},
	}
}

type Tester struct {
	cfg TesterConfig
	rt  *runtime.Runtime
}

func (t *Tester) TestComponent(specFile string) error {
	// -- read the spec file
	b, err := os.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to read spec file %s: %w", specFile, err)
	}

	// -- parse the spec file
	var comp model.Component
	if err := yaml.Unmarshal(b, &comp); err != nil {
		return fmt.Errorf("%s: parse: %w", specFile, err)
	}

	// -- generate default steps
	steps, err := t.generateDefault(comp)
	if err != nil {
		return fmt.Errorf("%s: generate: %w", comp.Name, err)
	}

	// -- compile the steps
	artifact, err := compiler.CompileWithContext(context.Background(), t.rt, *steps)
	if err != nil {
		return fmt.Errorf("%s: compile: %w", comp.Name, err)
	}

	// -- optionally dump the artifact
	if t.cfg.DumpDirectory != "" {
		if err := os.MkdirAll(t.cfg.DumpDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create dump directory %s: %w", t.cfg.DumpDirectory, err)
		}

		dumpFile := fmt.Sprintf("%s/%s-%s.yaml", t.cfg.DumpDirectory, comp.Kind, comp.Name)

		if err := os.WriteFile(dumpFile, []byte(artifact), 0644); err != nil {
			return fmt.Errorf("failed to write dump file %s: %w", dumpFile, err)
		}
	}

	// -- lint the artifact
	if err := service.NewStreamBuilder().SetYAML(artifact); err != nil {
		if t.cfg.DumpOnErrorDirectory != "" {
			// write the artifact to the error dir for inspection
			errFile := fmt.Sprintf("%s/%s-%s.yaml", t.cfg.DumpOnErrorDirectory, comp.Kind, comp.Name)

			_ = os.WriteFile(errFile, []byte(artifact), 0644)

			return fmt.Errorf("%s: %s: validate: %w (artifact written to %s)", comp.Kind, comp.Name, err, errFile)
		}

		return fmt.Errorf("%s: %s: validate: %w", comp.Kind, comp.Name, err)
	}

	return nil
}

func (t *Tester) generateDefault(component model.Component) (*model.Steps, error) {
	switch component.Kind {
	case model.ComponentKindSource:
		return generateDefaultInlet(component)
	case model.ComponentKindSink:
		return generateDefaultOutlet(component)
	default:
		return nil, fmt.Errorf("component %s is not a source or sink", component.Name)
	}
}
