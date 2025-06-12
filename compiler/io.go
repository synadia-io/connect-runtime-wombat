package compiler

import (
	"github.com/synadia-io/connect/model"
)

func compileSource(m model.SourceStep, t *model.TransformerStep) Fragment {
	result := Frag().
		Map(m.Type, m.Config)

	if t != nil {
		result.Fragments("processors", compileTransformer(*t))
	}

	return result
}

func compileSink(m model.SinkStep) Fragment {
	return Frag().Map(m.Type, m.Config)
}
