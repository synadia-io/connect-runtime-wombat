package compiler

import (
	"fmt"

	"github.com/synadia-io/connect/model"
)

// compileConsumer transforms a Connect consumer specification into a Wombat input configuration.
// A consumer reads from NATS (core, stream, or key-value) and processes messages.
//
// The function validates that exactly one consumer type is specified (core, stream, or kv)
// and optionally adds transformer processors if provided.
//
// Parameters:
//   - m: The consumer step containing the NATS configuration
//   - t: Optional transformer step for message processing
//
// Returns:
//   - A Fragment containing the Wombat input configuration
//   - An error if validation fails or multiple consumer types are specified
func compileConsumer(m model.ConsumerStep, t *model.TransformerStep) (Fragment, error) {
	types := 0
	var result Fragment
	if m.Core != nil {
		result = compileCoreConsumer(m)
		types++
	}

	if m.Stream != nil {
		result = compileStreamConsumer(m)
		types++
	}

	if m.Kv != nil {
		result = compileKvConsumer(m)
		types++
	}

	if types != 1 {
		return nil, fmt.Errorf("exactly one consumer type (core, stream, kv) must be defined")
	}

	if t != nil {
		result.Fragments("processors", compileTransformer(*t))
	}

	return result, nil
}

// compileCoreConsumer creates a Wombat configuration for consuming from core NATS subjects.
// Core NATS provides at-most-once delivery without persistence.
func compileCoreConsumer(m model.ConsumerStep) Fragment {
	return Frag().
		Fragment("nats", natsBaseFragment(m.Nats).
			String("subject", m.Core.Subject).
			StringP("queue", m.Core.Queue))
}

func compileStreamConsumer(m model.ConsumerStep) Fragment {
	return Frag().
		Fragment("nats_jetstream", natsBaseFragment(m.Nats).
			String("subject", m.Stream.Subject))

}

func compileKvConsumer(m model.ConsumerStep) Fragment {
	return Frag().
		Fragment("nats_kv", natsBaseFragment(m.Nats).
			String("bucket", m.Kv.Bucket).
			String("key", m.Kv.Key))
}
