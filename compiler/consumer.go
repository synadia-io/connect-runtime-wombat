package compiler

import (
	"github.com/synadia-labs/vent/public/control"
)

func compileConsumer(steps control.Steps) (map[string]any, error) {
	if steps.Consumer.JetStream != nil {
		return compileJetStreamConsumer(steps)
	}

	return compileCoreConsumer(steps)
}

func compileCoreConsumer(steps control.Steps) (map[string]any, error) {
	cfg := map[string]any{
		"subject": steps.Consumer.Subject,
	}

	attachNatsConfig(cfg, steps.Consumer.NatsConfig)

	if steps.Consumer.Queue != "" {
		cfg["queue"] = steps.Consumer.Queue
	}

	return map[string]any{"nats": cfg}, nil
}

func compileJetStreamConsumer(steps control.Steps) (map[string]any, error) {
	cfg := map[string]any{
		"subject":         steps.Consumer.Subject,
		"max_ack_pending": 1,
		"deliver":         "all",
		"bind":            false,
	}

	attachNatsConfig(cfg, steps.Consumer.NatsConfig)

	if steps.Consumer.Queue != "" {
		cfg["queue"] = steps.Consumer.Queue
	}

	if steps.Consumer.JetStream.DeliverPolicy != "" {
		cfg["deliver"] = steps.Consumer.JetStream.DeliverPolicy
	}

	if steps.Consumer.JetStream.MaxAckWait != "" {
		cfg["ack_wait"] = steps.Consumer.JetStream.MaxAckWait
	}

	if steps.Consumer.JetStream.Bind {
		cfg["bind"] = true
	}

	if steps.Consumer.JetStream.Durable != "" {
		cfg["durable"] = steps.Consumer.JetStream.Durable
	}

	if steps.Consumer.JetStream.MaxAckPending != 0 {
		cfg["max_ack_pending"] = steps.Consumer.JetStream.MaxAckPending
	}

	return map[string]any{"nats_jetstream": cfg}, nil
}
