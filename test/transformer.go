package test

import (
	"github.com/synadia-io/connect/model"
)

func ServiceTransformer(natsConfig model.NatsConfig, svcSubject string) *model.TransformerStep {
	return &model.TransformerStep{
		Service: &model.ServiceTransformerStep{
			Endpoint: svcSubject,
			Nats:     natsConfig,
		},
	}
}
