package test

import (
    "github.com/synadia-io/connect/builders"
)

func CoreProducer(nats *builders.NatsConfigBuilder) *builders.ProducerStepBuilder {
    return builders.ProducerStep(nats).Core(builders.ProducerStepCore("foo.bar"))
}

func CoreProducerWithSubject(nats *builders.NatsConfigBuilder, subject string) *builders.ProducerStepBuilder {
    return builders.ProducerStep(nats).Core(builders.ProducerStepCore(subject))
}
