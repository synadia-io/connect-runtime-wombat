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
)

var _ = Describe("Inlet", func() {
    Describe("Compiling an inlet", func() {
        When("the inlet configuration is invalid", func() {
            It("should return an error", func() {
                invalidInlet := Steps().
                    Source(test.InvalidSource()).
                    Producer(test.CoreProducer(test.UnauthenticatedNatsConfig())).
                    Build()
                artifact, err := compiler.Compile(test.Runtime(), invalidInlet)
                Expect(err).NotTo(HaveOccurred())

                sb, err := compiler.Validate(context.Background(), test.Runtime(), artifact, nil)
                Expect(sb).To(BeNil())
                Expect(err).To(HaveOccurred())
            })
        })

        When("the inlet has a valid source and producer", func() {
            var inlet model.Steps

            BeforeEach(func() {
                inlet = Steps().
                    Source(test.GenerateSource()).Producer(test.CoreProducer(test.UnauthenticatedNatsConfig())).
                    Build()
            })

            It("should generate a valid wombat artifact", func() {
                artifact, err := compiler.Compile(test.Runtime(), inlet)
                Expect(err).NotTo(HaveOccurred())

                Expect(artifact).NotTo(BeNil())
                GinkgoLogr.Info(artifact)

                sb := service.NewStreamBuilder()
                Expect(sb.SetYAML(artifact)).NotTo(HaveOccurred())
                _, err = sb.Build()
                Expect(err).NotTo(HaveOccurred())
            })
        })
    })
})
