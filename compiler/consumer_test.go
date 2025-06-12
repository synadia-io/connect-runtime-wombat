package compiler

import (
	"testing"

	. "github.com/synadia-io/connect/builders"
	"github.com/synadia-io/connect/model"
)

var ncb = NatsConfig(DefaultNatsUrl)

type consumerStepTest struct {
	name        string
	errored     bool
	step        *ConsumerStepBuilder
	transformer *TransformerStepBuilder
	exp         Fragment
}

func TestCompileConsumer(t *testing.T) {
	runConsumerStepTests(t,
		consumerStepTest{"should error if no consumer type is set", true,
			ConsumerStep(ncb),
			nil,
			nil,
		},
		consumerStepTest{"should error if multiple consumer types are set", true,
			ConsumerStep(ncb).Core(ConsumerStepCore("foo")).Kv(ConsumerStepKv("foo", "bar")),
			nil,
			nil,
		},
	)
}

func TestCompileCoreConsumer(t *testing.T) {
	runConsumerStepTests(t,
		consumerStepTest{"should render a core consumer", false,
			ConsumerStep(ncb).Core(ConsumerStepCore("foo").Queue("bar")),
			nil,
			Frag().Fragment("nats", Frag().
				Strings("urls", DefaultNatsUrl).
				String("subject", "foo").
				String("queue", "bar")),
		},
	)
}

func TestCompileStreamConsumer(t *testing.T) {
	runConsumerStepTests(t,
		consumerStepTest{"should render a stream consumer", false,
			ConsumerStep(ncb).Stream(ConsumerStepStream("foo")),
			nil,
			Frag().Fragment("nats_jetstream", Frag().
				Strings("urls", DefaultNatsUrl).
				String("subject", "foo")),
		},
	)
}

func TestCompileKvConsumer(t *testing.T) {
	runConsumerStepTests(t,
		consumerStepTest{"should render a kv consumer", false,
			ConsumerStep(ncb).Kv(ConsumerStepKv("foo", "bar")),
			nil,
			Frag().Fragment("nats_kv", Frag().
				Strings("urls", DefaultNatsUrl).
				String("bucket", "foo").
				String("key", "bar")),
		},
	)
}

func runConsumerStepTests(t *testing.T, tests ...consumerStepTest) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tf *model.TransformerStep
			if tt.transformer != nil {
				tr := tt.transformer.Build()
				tf = &tr
			}

			res, err := compileConsumer(tt.step.Build(), tf)
			if tt.errored && err == nil {
				t.Errorf("expected error, got nil")
			}

			if !tt.exp.EqualsMap(res) {
				t.Errorf("expected %v, got %v", tt.exp, tt.step)
			}
		})
	}
}
