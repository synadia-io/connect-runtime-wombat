package compiler_test

import (
	"context"
	"github.com/Jeffail/gabs/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect-runtime-wombat/test"
	. "github.com/synadia-io/connect/builders"
	"github.com/synadia-io/connect/model"
	"gopkg.in/yaml.v3"
	"strings"

	_ "github.com/synadia-io/connect-runtime-wombat/components"
)

var _ = Describe("Compiling an inlet", func() {
	When("the configuration is invalid", func() {
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
		It("should generate a valid wombat artifact", func() {
			inlet := Steps().
				Source(test.GenerateSource()).
				Producer(test.CoreProducer(test.UnauthenticatedNatsConfig())).
				Build()

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
			artifact, err := compiler.Compile(test.Runtime(), v)
			Expect(err).NotTo(HaveOccurred())
			GinkgoLogr.Info(artifact)

			// parse yaml
			var m map[string]any
			Expect(yaml.Unmarshal([]byte(artifact), &m)).To(Succeed())
			am := gabs.Wrap(m)

			Expect(am.Exists(strings.Split("input.stdin", ".")...)).To(BeTrue())
			Expect(am.Exists(strings.Split("input.processors.0.nats_request_reply", ".")...)).To(BeTrue())
			Expect(am.Path("input.processors.0.nats_request_reply.urls").Data()).To(ContainElement("nats://localhost:4222"))
			Expect(am.Path("input.processors.0.nats_request_reply.subject").Data()).To(Equal("my.service"))
			Expect(am.Path("input.processors.0.nats_request_reply.timeout").Data()).To(Equal("5s"))
			Expect(am.Path("input.processors.0.nats_request_reply.metadata.include_patterns").Data()).To(ContainElement(".*"))

			Expect(am.Exists(strings.Split("output.nats", ".")...)).To(BeTrue())
			Expect(am.Path("output.nats.urls").Data()).To(ContainElement("nats://localhost:4222"))
			Expect(am.Path("output.nats.subject").Data()).To(Equal("foo.bar"))
			Expect(am.Path("output.nats.max_in_flight").Data()).To(Equal(1))
			Expect(am.Path("output.nats.metadata.include_patterns").Data()).To(ContainElement(".*"))
		})
	})

	When("the connector contains an explode transformer", func() {
		var v model.Steps

		BeforeEach(func() {
			v = Steps().
				Source(SourceStep("stdin")).
				Transformer(TransformerStep().Explode(ExplodeTransformerStep().Format(model.ExplodeTransformerStepFormatCsv).Delimiter("\t"))).
				Producer(ProducerStep(NatsConfig(DefaultNatsUrl)).Core(ProducerStepCore("foo.bar"))).
				Build()
		})

		It("should generate a valid wombat artifact", func() {
			artifact, err := compiler.Compile(test.Runtime(), v)
			Expect(err).NotTo(HaveOccurred())
			GinkgoLogr.Info(artifact)

			// parse yaml
			var m map[string]any
			Expect(yaml.Unmarshal([]byte(artifact), &m)).To(Succeed())
			am := gabs.Wrap(m)

			Expect(am.Exists(strings.Split("input.stdin", ".")...)).To(BeTrue())
			Expect(am.Exists(strings.Split("input.processors.0.unarchive", ".")...)).To(BeTrue())
			Expect(am.Path("input.processors.0.unarchive.format").Data()).To(Equal(string(model.ExplodeTransformerStepFormatCsv)))

			Expect(am.Exists(strings.Split("output.nats", ".")...)).To(BeTrue())
			Expect(am.Path("output.nats.urls").Data()).To(ContainElement("nats://localhost:4222"))
			Expect(am.Path("output.nats.subject").Data()).To(Equal("foo.bar"))
			Expect(am.Path("output.nats.max_in_flight").Data()).To(Equal(1))
			Expect(am.Path("output.nats.metadata.include_patterns").Data()).To(ContainElement(".*"))
		})
	})

	When("the connector contains a combine transformer", func() {
		var v model.Steps

		BeforeEach(func() {
			v = Steps().
				Source(SourceStep("stdin")).
				Transformer(TransformerStep().Combine(CombineTransformerStep().Format(model.CombineTransformerStepFormatLines))).
				Producer(ProducerStep(NatsConfig(DefaultNatsUrl)).Core(ProducerStepCore("foo.bar"))).
				Build()
		})

		It("should generate a valid wombat artifact", func() {
			artifact, err := compiler.Compile(test.Runtime(), v)
			Expect(err).NotTo(HaveOccurred())
			GinkgoLogr.Info(artifact)

			// parse yaml
			var m map[string]any
			Expect(yaml.Unmarshal([]byte(artifact), &m)).To(Succeed())
			am := gabs.Wrap(m)

			Expect(am.Exists(strings.Split("input.stdin", ".")...)).To(BeTrue())
			Expect(am.Exists(strings.Split("input.processors.0.archive", ".")...)).To(BeTrue())
			Expect(am.Path("input.processors.0.archive.format").Data()).To(Equal(string(model.CombineTransformerStepFormatLines)))

			Expect(am.Exists(strings.Split("output.nats", ".")...)).To(BeTrue())
			Expect(am.Path("output.nats.urls").Data()).To(ContainElement("nats://localhost:4222"))
			Expect(am.Path("output.nats.subject").Data()).To(Equal("foo.bar"))
			Expect(am.Path("output.nats.max_in_flight").Data()).To(Equal(1))
			Expect(am.Path("output.nats.metadata.include_patterns").Data()).To(ContainElement(".*"))
		})
	})
})
