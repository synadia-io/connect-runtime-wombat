package test

import (
	"fmt"

	"github.com/synadia-io/connect/builders"
)

func UnauthenticatedNatsConfig() *builders.NatsConfigBuilder {
	return builders.NatsConfig("nats://localhost:4222")
}

func NatsConfig(port int) *builders.NatsConfigBuilder {
	return builders.NatsConfig(fmt.Sprintf("nats://localhost:%d", port))
}
