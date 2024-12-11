package test

import (
	"github.com/synadia-labs/vent/public/control"
)

func ServiceTransformer(natsConfig control.NatsConfig, svcSubject string) *control.Transformer {
	return &control.Transformer{
		Service: &control.ServiceTransformer{
			Endpoint:   svcSubject,
			NatsConfig: natsConfig,
		},
	}
}
