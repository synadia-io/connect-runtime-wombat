package compiler_test

import (
    "context"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/redpanda-data/benthos/v4/public/service"
    "github.com/synadia-io/connect-runtime-wombat/compiler"
    "github.com/synadia-io/connect-runtime-wombat/test"
    . "github.com/synadia-io/connect/builders"
    "github.com/synadia-io/connect/model"

    _ "github.com/synadia-io/connect-runtime-wombat/components"
)

var _ = Describe("Compiling an inlet", func() {
    When("the configuration is invalid", func() {
        It("should return an error", func() {
            invalidInlet := Steps().
                Source(test.InvalidSource()).
                Producer(test.CoreProducer(test.UnauthenticatedNatsConfig())).
                Build()
            artifact, err := compiler.Compile(invalidInlet)
            Expect(err).NotTo(HaveOccurred())

            sb, err := compiler.Validate(context.Background(), test.Runtime(), artifact, nil)
            Expect(sb).To(BeNil())
            Expect(err).To(HaveOccurred())
        })
    })

    When("the inlet has a valid source and producer", func() {
        It("should generate a valid wombat artifact", func() {
            inlet := Steps().
                Source(test.GenerateSource()).
                Producer(test.CoreProducer(test.UnauthenticatedNatsConfig())).
                Build()

            artifact, err := compiler.Compile(inlet)
            Expect(err).NotTo(HaveOccurred())

            Expect(artifact).NotTo(BeNil())
            GinkgoLogr.Info(artifact)

            sb := service.NewStreamBuilder()
            Expect(sb.SetYAML(artifact)).NotTo(HaveOccurred())
            _, err = sb.Build()
            Expect(err).NotTo(HaveOccurred())
        })
    })

    When("the inlet contains a service transformer", func() {
        var v model.Steps

        BeforeEach(func() {
            v = Steps().
                Source(SourceStep("stdin")).
                Transformer(TransformerStep().Service(ServiceTransformerStep("my.service", NatsConfig(DefaultNatsUrl)))).
                Producer(ProducerStep(NatsConfig(DefaultNatsUrl)).Core(ProducerStepCore("foo.bar"))).
                Build()
        })

        It("should generate a valid wombat artifact", func() {
            artifact, err := compiler.Compile(v)
            Expect(err).NotTo(HaveOccurred())
            GinkgoLogr.Info(artifact)

            expected := `
input:
    stdin: {}
    processors:
        - nats_request_reply:
            urls:
                - nats://localhost:4222
            subject: my.service
            timeout: 5s
            metadata:
                include_patterns: [ ".*" ]
metrics:
    prometheus: {}
output:
    nats:
        urls: 
            - nats://localhost:4222
        subject: foo.bar
        max_in_flight: 1
        metadata:
            include_patterns: [ ".*" ]
`

            cl, err := test.DiffYaml(expected, artifact)
            Expect(err).NotTo(HaveOccurred())
            Expect(cl).To(BeEmpty())
        })
    })
})
