package main_test

import (
	"fmt"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/synadia-io/connect/runtime"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var srv *server.Server
var nc *nats.Conn
var natsUrl string

const TestPort = 60002

var _ = BeforeSuite(func() {
	var err error

	opts := test.DefaultTestOptions
	opts.Port = TestPort
	opts.JetStream = true
	srv = test.RunServer(&opts)

	natsUrl = fmt.Sprintf("nats://localhost:%d", TestPort)
	err = os.Setenv(runtime.NatsUrlVar, natsUrl)
	Expect(err).ToNot(HaveOccurred())

	nc, err = nats.Connect(natsUrl)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	if nc != nil {
		nc.Close()
	}

	if srv != nil {
		srv.Shutdown()
	}
})

func TestWombat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Wombat Suite")
}
