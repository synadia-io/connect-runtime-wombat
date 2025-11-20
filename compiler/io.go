package compiler

import (
	"github.com/synadia-io/connect/v2/model"
)

// compileSource transforms a Connect source specification into a Wombat input configuration.
// Sources read data from external systems (files, databases, APIs, etc.) and produce messages.
//
// The function creates a configuration map with the source type as the key and its
// configuration as the value. If a transformer is provided, it's added as a processor.
//
// Parameters:
//   - m: The source step containing the type and configuration
//   - t: Optional transformer step for processing source messages
//
// Returns a Fragment containing the Wombat input configuration.
func compileSource(m model.SourceStep, t *model.TransformerStep) Fragment {
	result := Frag().
		Map(m.Type, m.Config)

	if t != nil {
		result.Fragments("processors", compileTransformer(*t))
	}

	return result
}

// compileSink transforms a Connect sink specification into a Wombat output configuration.
// Sinks write messages to external systems (files, databases, APIs, etc.).
//
// The function creates a configuration map with the sink type as the key and its
// configuration as the value.
//
// Parameters:
//   - m: The sink step containing the type and configuration
//
// Returns a Fragment containing the Wombat output configuration.
func compileSink(m model.SinkStep) Fragment {
	return Frag().Map(m.Type, m.Config)
}
