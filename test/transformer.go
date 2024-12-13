package test

import (
	"github.com/synadia-io/connect/model"
)

func ServiceTransformer(natsConfig model.NatsConfig, svcSubject string) *model.Transformer {
	return &model.Transformer{
		Service: &model.ServiceTransformer{
			Endpoint:   svcSubject,
			NatsConfig: natsConfig,
		},
	}
}
