package compiler

import (
	"fmt"

	"github.com/synadia-io/connect/model"
)

// compileProducer transforms a Connect producer specification into a Wombat output configuration.
// A producer writes messages to NATS (core, stream, or key-value).
//
// The function validates that exactly one producer type is specified (core, stream, or kv)
// and returns the appropriate Wombat configuration.
//
// Parameters:
//   - m: The producer step containing the NATS configuration
//
// Returns:
//   - A Fragment containing the Wombat output configuration
//   - An error if validation fails or multiple producer types are specified
func compileProducer(m model.ProducerStep) (Fragment, error) {
	types := 0
	if m.Core != nil {
		types++
	}
	if m.Stream != nil {
		types++
	}
	if m.Kv != nil {
		types++
	}
	if types != 1 {
		return nil, fmt.Errorf("exactly one consumer type (core, stream, kv) must be defined")
	}

	if m.Core != nil {
		return compileCoreProducer(m), nil
	}

	if m.Stream != nil {
		return compileStreamProducer(m), nil
	}

	if m.Kv != nil {
		return compileKvProducer(m), nil
	}

	return nil, fmt.Errorf("at least one producer type (core, stream, kv) must be defined")
}

// compileCoreProducer creates a Wombat configuration for publishing to core NATS subjects.
// Core NATS provides at-most-once delivery without persistence.
func compileCoreProducer(m model.ProducerStep) Fragment {
	return Frag().
		Fragment("nats", natsBaseFragment(m.Nats).
			String("subject", m.Core.Subject).
			Int("max_in_flight", m.Threads).
			Fragment("metadata", Frag().
				Strings("include_patterns", ".*")))
}

func compileStreamProducer(m model.ProducerStep) Fragment {
	return Frag().
		Fragment("nats_jetstream", natsBaseFragment(m.Nats).
			String("subject", m.Stream.Subject).
			Int("max_in_flight", m.Threads).
			Fragment("metadata", Frag().
				Strings("include_patterns", ".*")))
}

func compileKvProducer(m model.ProducerStep) Fragment {
	return Frag().
		Fragment("nats_kv", natsBaseFragment(m.Nats).
			String("bucket", m.Kv.Bucket).
			String("key", m.Kv.Key).
			Int("max_in_flight", m.Threads))
}
