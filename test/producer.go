package test

import (
	"github.com/synadia-labs/vent/public/control"
)

func CoreProducer(cfg control.NatsConfig) control.Producer {
	return CoreProducerWithSubject(cfg, "foo.bar")
}

func CoreProducerWithSubject(cfg control.NatsConfig, subject string) control.Producer {
	return control.Producer{
		NatsConfig: cfg,
		Subject:    subject,
	}
}

func StreamProducer(cfg control.NatsConfig) control.Producer {
	return control.Producer{
		NatsConfig: cfg,
		Subject:    "foo.bar",
		JetStream:  &control.ProducerJetStreamOptions{},
	}
}
