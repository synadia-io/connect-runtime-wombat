package main_test

import (
    "context"
    "fmt"
    "github.com/synadia-io/connect-runtime-wombat/runner"
    "github.com/synadia-io/connect-runtime-wombat/test"
    . "github.com/synadia-io/connect/builders"
    "strings"
    "sync"
    "sync/atomic"
    "time"

    "github.com/google/uuid"
    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/micro"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("Running an inlet", func() {
    When("the inlet configuration is invalid", func() {
        It("should return an error", func() {
            invalidInlet := Steps().
                Source(SourceStep("invalid")).
                Producer(test.CoreProducer(NatsConfig(DefaultNatsUrl))).
                Build()
            err := runner.Run(context.Background(), test.Runtime(), invalidInlet)
            Expect(err).To(HaveOccurred())
        })
    })

    When("the inlet has a valid input and target", func() {
        It("should consume messages and send them to nats", func() {
            // -- generate a subject name
            subject := fmt.Sprintf("test.%s", uuid.New().String())

            expectedMsgCount := 5
            var wg sync.WaitGroup
            wg.Add(expectedMsgCount)

            var msgCount atomic.Int32
            s, err := nc.Subscribe(subject, func(msg *nats.Msg) {
                msgCount.Add(1)
                wg.Done()
            })
            Expect(err).NotTo(HaveOccurred())
            defer s.Drain()

            inlet := Steps().
                Source(test.GenerateSource()).
                Producer(test.CoreProducerWithSubject(test.NatsConfig(TestPort), subject)).
                Build()
            err = runner.Run(context.Background(), test.Runtime(), inlet)
            Expect(err).NotTo(HaveOccurred())

            // Sometimes the runner finishes before the nats connection has received the final message
            err = waitTimeout(&wg, 100*time.Millisecond)
            Expect(err).NotTo(HaveOccurred())

            Expect(msgCount.Load()).To(BeNumerically("==", expectedMsgCount))
        })

        When("the inlet has a transformer ", func() {
            It("should transform its messages and send them to nats", func() {
                serviceCallCount := 0
                serviceCallLock := sync.Mutex{}

                // -- generate a service name
                serviceName := uuid.New().String()
                err := test.AttachService(nc, serviceName, func(request micro.Request) {
                    serviceCallLock.Lock()
                    defer serviceCallLock.Unlock()
                    serviceCallCount++

                    result := strings.ToUpper(string(request.Data()))

                    _ = request.Respond([]byte(result))
                })
                Expect(err).NotTo(HaveOccurred())

                // -- generate a subject name
                subject := fmt.Sprintf("test.%s", uuid.New().String())

                outputLock := sync.Mutex{}
                var outputMessages []nats.Msg
                s, err := nc.Subscribe(subject, func(msg *nats.Msg) {
                    outputLock.Lock()
                    defer outputLock.Unlock()
                    outputMessages = append(outputMessages, *msg)
                })
                Expect(err).NotTo(HaveOccurred())
                defer s.Drain()

                inlet := Steps().
                    Source(test.GenerateSource()).
                    Transformer(TransformerStep().
                        Service(ServiceTransformerStep(fmt.Sprintf("service.%s", serviceName), test.NatsConfig(TestPort)))).
                    Producer(test.CoreProducerWithSubject(test.NatsConfig(TestPort), subject)).
                    Build()
                err = runner.Run(context.Background(), test.Runtime(), inlet)
                Expect(err).NotTo(HaveOccurred())

                Expect(serviceCallCount).To(BeNumerically("==", 5))
                for _, msg := range outputMessages {
                    Expect(string(msg.Data)).To(Equal(strings.ToUpper(string(msg.Data))))
                }
            })
        })
    })
})

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
    c := make(chan struct{})
    go func() {
        defer close(c)
        wg.Wait()
    }()
    select {
    case <-c:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("wait group timed out after: %v", timeout)
    }
}
