package compiler

import (
	"github.com/synadia-io/connect/v2/model"
)

// compileTransformer transforms a Connect transformer specification into a Wombat processor configuration.
// Transformers modify messages as they flow through the pipeline.
//
// Supported transformer types:
//   - Composite: A sequence of transformers applied in order
//   - Service: Call an external NATS service for transformation
//   - Mapping: Transform messages using Bloblang expressions
//   - Explode: Split arrays/objects into individual messages
//   - Combine: Batch multiple messages together
//
// Parameters:
//   - transformer: The transformer step containing the transformation logic
//
// Returns a Fragment containing the Wombat processor configuration, or nil if no transformer type is specified.
func compileTransformer(transformer model.TransformerStep) Fragment {
	if transformer.Composite != nil {
		return compileCompositeTransformer(transformer.Composite)
	}

	if transformer.Service != nil {
		return compileServiceTransformer(transformer.Service)
	}

	if transformer.Mapping != nil {
		return compileMappingTransformer(transformer.Mapping)
	}

	if transformer.Explode != nil {
		return compileExplodeTransformer(transformer.Explode)
	}

	if transformer.Combine != nil {
		return compileCombineTransformer(transformer.Combine)
	}

	return nil
}

// compileServiceTransformer creates a Wombat processor that calls an external NATS service.
// The service transformer sends the message to a NATS endpoint and replaces it with the response.
func compileServiceTransformer(t *model.ServiceTransformerStep) Fragment {
	return Frag().
		Fragment("nats_request_reply", natsBaseFragment(t.Nats).
			String("subject", t.Endpoint).
			String("timeout", t.Timeout).
			Fragment("metadata", Frag().
				Strings("include_patterns", ".*")))
}

// compileCompositeTransformer creates a sequence of processors from multiple transformers.
// Each transformer in the sequence is applied to the message in order.
func compileCompositeTransformer(t *model.CompositeTransformerStep) Fragment {
	var seq []Fragment
	for _, ct := range t.Sequential {
		seq = append(seq, compileTransformer(ct))
	}

	return Frag().Fragment("processors", Frag().Fragments("sequence", seq...))
}

func compileMappingTransformer(t *model.MappingTransformerStep) Fragment {
	return Frag().String("mapping", t.Sourcecode)
}

func compileExplodeTransformer(t *model.ExplodeTransformerStep) Fragment {
	if t.Format == model.ExplodeTransformerStepFormatCsv && t.Delimiter != "," {
		return Frag().Fragment("unarchive", Frag().
			String("format", string(t.Format)).
			String("delimiter", t.Delimiter))
	}

	return Frag().Fragment("unarchive", Frag().
		String("format", string(t.Format)))
}

func compileCombineTransformer(t *model.CombineTransformerStep) Fragment {
	if t.Format == model.CombineTransformerStepFormatTar || t.Format == model.CombineTransformerStepFormatZip {
		return Frag().Fragment("archive", Frag().
			String("format", string(t.Format)).
			String("path", t.Path))

	}

	return Frag().Fragment("archive", Frag().
		String("format", string(t.Format)))
}
