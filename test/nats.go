package test

import (
	"fmt"
	"github.com/synadia-io/connect/model"
)

func AuthenticatedNatsConfig() model.NatsConfig {
	return model.NatsConfig{
		Url:         "nats://localhost:4222",
		AuthEnabled: true,
		Jwt:         "configured-jwt",
		Seed:        "configured-seed",
	}
}

func UnauthenticatedNatsConfig() model.NatsConfig {
	return model.NatsConfig{
		Url: "nats://localhost:4222",
	}
}

func NatsConfig(port int) model.NatsConfig {
	return model.NatsConfig{
		Url:         fmt.Sprintf("nats://localhost:%d", port),
		AuthEnabled: false,
	}
}
