package compiler

import (
	"github.com/synadia-io/connect/model"
)

func compileSource(steps model.Steps) (map[string]any, error) {
	m := map[string]any{
		steps.Source.Type: steps.Source.Config,
	}

	return m, nil
}

func compileSink(steps model.Steps) (map[string]any, error) {
	m := map[string]any{
		steps.Sink.Type: steps.Sink.Config,
	}

	// -- add the transformer
	if steps.Transformer != nil {
		return attachTransformerAsProcessor(m, steps)
	}

	return m, nil
}
