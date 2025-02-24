package test

import (
    "github.com/synadia-io/connect/model"
)

func CoreProducer(cfg model.NatsConfig) model.ProducerStep {
    return CoreProducerWithSubject(cfg, "foo.bar")
}

func CoreProducerWithSubject(cfg model.NatsConfig, subject string) model.ProducerStep {
    return model.ProducerStep{
        Nats:    cfg,
        Subject: subject,
    }
}
