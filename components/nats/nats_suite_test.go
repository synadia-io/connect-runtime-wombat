package nats_test

import (
    "github.com/nats-io/nats.go"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats-server/v2/test"
)

var srv *server.Server
var nc *nats.Conn

func TestNats(t *testing.T) {
    BeforeSuite(func() {
        var err error
        srv = test.RunRandClientPortServer()
        nc, err = nats.Connect(srv.ClientURL())
        Expect(err).To(BeNil())
    })

    AfterSuite(func() {
        nc.Close()
        srv.Shutdown()
    })

    RegisterFailHandler(Fail)
    RunSpecs(t, "Nats Suite")
}
