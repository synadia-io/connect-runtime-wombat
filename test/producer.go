package test

import (
	"github.com/synadia-io/connect/model"
)

func CoreProducer(cfg model.NatsConfig) model.Producer {
	return CoreProducerWithSubject(cfg, "foo.bar")
}

func CoreProducerWithSubject(cfg model.NatsConfig, subject string) model.Producer {
	return model.Producer{
		NatsConfig: cfg,
		Subject:    subject,
	}
}

func StreamProducer(cfg model.NatsConfig) model.Producer {
	return model.Producer{
		NatsConfig: cfg,
		Subject:    "foo.bar",
		JetStream:  &model.ProducerJetStreamOptions{},
	}
}
