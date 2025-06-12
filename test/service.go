package test

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

func AttachService(nc *nats.Conn, name string, handlerFunc micro.HandlerFunc) error {
	_, err := micro.AddService(nc, micro.Config{
		Name:    name,
		Version: "1.0.0",
		Endpoint: &micro.EndpointConfig{
			Subject: fmt.Sprintf("service.%s", name),
			Handler: handlerFunc,
		},
	})
	return err
}
