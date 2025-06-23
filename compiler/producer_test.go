package compiler

import (
	"testing"

	. "github.com/synadia-io/connect/builders"
)

type producerStepTest struct {
	name    string
	errored bool
	step    *ProducerStepBuilder
	exp     Fragment
}

func TestCompileProducer(t *testing.T) {
	runProducerStepTests(t,
		producerStepTest{"should error if no producer type is set", true,
			ProducerStep(ncb),
			nil,
		},
		producerStepTest{"should error if multiple producer types are set", true,
			ProducerStep(ncb).Core(ProducerStepCore("foo")).Kv(ProducerStepKv("foo", "bar")),
			nil,
		},
	)
}

func TestCompileCoreProducer(t *testing.T) {
	runProducerStepTests(t,
		producerStepTest{"should render a core producer", false,
			ProducerStep(ncb).Core(ProducerStepCore("foo")),
			Frag().Fragment("nats", Frag().
				Strings("urls", DefaultNatsUrl).
				String("subject", "foo").
				Int("max_in_flight", 1).
				Fragment("metadata", Frag().
					Strings("include_patterns", ".*"))),
		},
	)
}

func TestCompileStreamProducer(t *testing.T) {
	runProducerStepTests(t,
		producerStepTest{"should render a stream producer", false,
			ProducerStep(ncb).Stream(ProducerStepStream("foo")),
			Frag().Fragment("nats_jetstream", Frag().
				Strings("urls", DefaultNatsUrl).
				String("subject", "foo").
				Int("max_in_flight", 1).
				Fragment("metadata", Frag().
					Strings("include_patterns", ".*"))),
		},
	)
}

func TestCompileKvProducer(t *testing.T) {
	runProducerStepTests(t,
		producerStepTest{"should render a kv producer", false,
			ProducerStep(ncb).Kv(ProducerStepKv("foo", "bar")),
			Frag().Fragment("nats_kv", Frag().
				Strings("urls", DefaultNatsUrl).
				String("bucket", "foo").
				String("key", "bar").
				Int("max_in_flight", 1)),
		},
	)
}

func runProducerStepTests(t *testing.T, tests ...producerStepTest) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := compileProducer(tt.step.Build())
			if tt.errored && err == nil {
				t.Errorf("expected error, got nil")
			}

			if !res.EqualsMap(map[string]any(tt.exp)) {
				t.Errorf("expected %v, got %v", tt.exp, res)
			}
		})
	}
}
