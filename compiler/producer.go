package compiler

import (
	"fmt"
	"github.com/synadia-io/connect/model"
)

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
