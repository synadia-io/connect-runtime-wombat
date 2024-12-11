package compiler_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-labs/vent/public/control"
	"github.com/synadia-labs/vent/runtimes/wombat/compiler"
	"github.com/synadia-labs/vent/runtimes/wombat/test"
)

var _ = Describe("Inlet", func() {
	Describe("Compiling an inlet", func() {
		When("the inlet configuration is invalid", func() {
			It("should return an error", func() {
				invalidInlet := test.Inlet(test.InvalidSource(), test.CoreProducer(test.UnauthenticatedNatsConfig()))
				artifact, err := compiler.Compile(invalidInlet)
				Expect(err).NotTo(HaveOccurred())

				sb, err := compiler.Validate(context.Background(), test.Runtime(), artifact, nil)
				Expect(sb).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		When("the inlet has a valid source and producer", func() {
			var inlet control.Steps

			BeforeEach(func() {
				inlet = test.Inlet(test.GenerateSource(), test.CoreProducer(test.UnauthenticatedNatsConfig()))
			})

			It("should generate a valid wombat artifact", func() {
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
	})
})
