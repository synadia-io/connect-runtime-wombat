package compiler

import (
    "fmt"
    "github.com/synadia-io/connect/model"
)

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
