package compiler

import (
    "github.com/synadia-io/connect/model"
)

func compileConsumer(steps model.Steps) (map[string]any, error) {
    if steps.Consumer.Jetstream != nil {
        return compileJetStreamConsumer(steps)
    }

    return compileCoreConsumer(steps)
}

func compileCoreConsumer(steps model.Steps) (map[string]any, error) {
    cfg := map[string]any{
        "subject": steps.Consumer.Subject,
    }

    attachNatsConfig(cfg, steps.Consumer.Nats)

    if steps.Consumer.Queue != nil {
        cfg["queue"] = *steps.Consumer.Queue
    }

    return map[string]any{"nats": cfg}, nil
}

func compileJetStreamConsumer(steps model.Steps) (map[string]any, error) {
    cfg := map[string]any{
        "subject":         steps.Consumer.Subject,
        "max_ack_pending": 1,
        "deliver":         "all",
        "bind":            false,
    }

    attachNatsConfig(cfg, steps.Consumer.Nats)

    if steps.Consumer.Queue != nil {
        cfg["queue"] = steps.Consumer.Queue
    }

    if steps.Consumer.Jetstream.DeliverPolicy != nil {
        cfg["deliver"] = *steps.Consumer.Jetstream.DeliverPolicy
    }

    if steps.Consumer.Jetstream.MaxAckWait != nil {
        cfg["ack_wait"] = *steps.Consumer.Jetstream.MaxAckWait
    }

    if steps.Consumer.Jetstream.Bind != nil {
        cfg["bind"] = *steps.Consumer.Jetstream.Bind
    }

    if steps.Consumer.Jetstream.Durable != nil {
        cfg["durable"] = *steps.Consumer.Jetstream.Durable
    }

    if steps.Consumer.Jetstream.MaxAckPending != nil {
        cfg["max_ack_pending"] = *steps.Consumer.Jetstream.MaxAckPending
    }

    return map[string]any{"nats_jetstream": cfg}, nil
}
