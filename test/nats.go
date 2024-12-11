package test

import (
	"fmt"
	"github.com/synadia-labs/vent/public/control"
)

func AuthenticatedNatsConfig() control.NatsConfig {
	return control.NatsConfig{
		Url:         "nats://localhost:4222",
		AuthEnabled: true,
		Jwt:         "configured-jwt",
		Seed:        "configured-seed",
	}
}

func UnauthenticatedNatsConfig() control.NatsConfig {
	return control.NatsConfig{
		Url: "nats://localhost:4222",
	}
}

func NatsConfig(port int) control.NatsConfig {
	return control.NatsConfig{
		Url:         fmt.Sprintf("nats://localhost:%d", port),
		AuthEnabled: false,
	}
}
