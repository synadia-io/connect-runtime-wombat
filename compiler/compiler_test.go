package compiler_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect-runtime-wombat/test"
	. "github.com/synadia-io/connect/builders"
	"github.com/synadia-io/connect/model"
	"github.com/synadia-io/connect/runtime"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Metrics Configuration", func() {
	var inlet model.Steps

	BeforeEach(func() {
		inlet = Steps().
			Source(test.GenerateSource()).
			Producer(test.CoreProducer(test.UnauthenticatedNatsConfig())).
			Build()
	})

	Context("when NATS URL is set (NEX_WORKLOAD_NATS_SERVERS environment)", func() {
		It("should include metrics configuration in the compiled YAML", func() {
			// Create runtime with NATS URL set (simulating NEX_WORKLOAD_NATS_SERVERS)
			rt := test.Runtime(
				runtime.WithNatsUrl("nats://localhost:4222"),
			)

			// Compile the inlet
			artifact, err := compiler.Compile(rt, inlet)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifact).NotTo(BeEmpty())

			// Parse the YAML to verify metrics configuration
			var config map[string]interface{}
			err = yaml.Unmarshal([]byte(artifact), &config)
			Expect(err).NotTo(HaveOccurred())

			// Assert that metrics configuration exists
			metrics, exists := config["metrics"]
			Expect(exists).To(BeTrue(), "metrics section should exist when NATS URL is set")
			Expect(metrics).NotTo(BeNil())

			// Verify NATS metrics configuration
			metricsMap, ok := metrics.(map[string]interface{})
			Expect(ok).To(BeTrue(), "metrics should be a map")

			natsMetrics, exists := metricsMap["nats"]
			Expect(exists).To(BeTrue(), "nats metrics configuration should exist")
			Expect(natsMetrics).NotTo(BeNil())

			// Verify NATS configuration details
			natsConfig, ok := natsMetrics.(map[string]interface{})
			Expect(ok).To(BeTrue(), "nats metrics should be a map")

			Expect(natsConfig["url"]).To(Equal("nats://localhost:4222"))
			expectedSubject := fmt.Sprintf("$NEX.FEED.%s.metrics.%s", rt.Namespace, rt.Instance)
			Expect(natsConfig["subject"]).To(Equal(expectedSubject))

			// Verify headers
			headers, exists := natsConfig["headers"]
			Expect(exists).To(BeTrue(), "headers should exist")
			headersMap, ok := headers.(map[string]interface{})
			Expect(ok).To(BeTrue(), "headers should be a map")
			Expect(headersMap["account"]).To(Equal(rt.Namespace))
			Expect(headersMap["connector_id"]).To(Equal(rt.Connector))
			Expect(headersMap["instance_id"]).To(Equal(rt.Instance))
		})

		It("should include JWT authentication when JWT and seed are provided", func() {
			// Create runtime with NATS URL, JWT and seed
			rt := test.Runtime(
				runtime.WithNatsUrl("nats://localhost:4222"),
				runtime.WithNatsJwt("test-jwt-token"),
				runtime.WithNatsSeed("test-seed"),
			)

			// Compile the inlet
			artifact, err := compiler.Compile(rt, inlet)
			Expect(err).NotTo(HaveOccurred())

			// Parse the YAML
			var config map[string]interface{}
			err = yaml.Unmarshal([]byte(artifact), &config)
			Expect(err).NotTo(HaveOccurred())

			// Navigate to NATS metrics configuration
			metrics := config["metrics"].(map[string]interface{})
			natsMetrics := metrics["nats"].(map[string]interface{})

			// Verify JWT and seed are included
			Expect(natsMetrics["jwt"]).To(Equal("test-jwt-token"))
			Expect(natsMetrics["seed"]).To(Equal("test-seed"))
		})
	})

	Context("when NATS URL is NOT set", func() {
		It("should NOT include metrics configuration in the compiled YAML", func() {
			// Create runtime without NATS URL (default test runtime)
			rt := test.Runtime()
			// Ensure NatsUrl is empty
			rt.NatsUrl = ""

			// Compile the inlet
			artifact, err := compiler.Compile(rt, inlet)
			Expect(err).NotTo(HaveOccurred())
			Expect(artifact).NotTo(BeEmpty())

			// Parse the YAML to verify metrics configuration is absent
			var config map[string]interface{}
			err = yaml.Unmarshal([]byte(artifact), &config)
			Expect(err).NotTo(HaveOccurred())

			// Assert that metrics configuration does NOT exist
			_, exists := config["metrics"]
			Expect(exists).To(BeFalse(), "metrics section should NOT exist when NATS URL is not set")
		})
	})

	Context("when only partial runtime configuration is provided", func() {
		It("should NOT include metrics when namespace is missing", func() {
			rt := test.Runtime(
				runtime.WithNatsUrl("nats://localhost:4222"),
			)
			rt.Namespace = "" // Clear namespace

			artifact, err := compiler.Compile(rt, inlet)
			Expect(err).NotTo(HaveOccurred())

			var config map[string]interface{}
			err = yaml.Unmarshal([]byte(artifact), &config)
			Expect(err).NotTo(HaveOccurred())

			_, exists := config["metrics"]
			Expect(exists).To(BeFalse(), "metrics section should NOT exist when namespace is missing")
		})

		It("should NOT include metrics when instance is missing", func() {
			rt := test.Runtime(
				runtime.WithNatsUrl("nats://localhost:4222"),
			)
			rt.Instance = "" // Clear instance

			artifact, err := compiler.Compile(rt, inlet)
			Expect(err).NotTo(HaveOccurred())

			var config map[string]interface{}
			err = yaml.Unmarshal([]byte(artifact), &config)
			Expect(err).NotTo(HaveOccurred())

			_, exists := config["metrics"]
			Expect(exists).To(BeFalse(), "metrics section should NOT exist when instance is missing")
		})
	})
})
