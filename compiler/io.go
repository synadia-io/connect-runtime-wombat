package compiler

import (
	"github.com/synadia-labs/vent/public/control"
)

func compileSource(steps control.Steps) (map[string]any, error) {
	m := map[string]any{
		steps.Source.Type: steps.Source.Config,
	}

	return m, nil
}

func compileSink(steps control.Steps) (map[string]any, error) {
	m := map[string]any{
		steps.Sink.Type: steps.Sink.Config,
	}

	// -- add the transformer
	if steps.Transformer != nil {
		return attachTransformerAsProcessor(m, steps)
	}

	return m, nil
}
