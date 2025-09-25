package full_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redpanda-data/benthos/v4/public/service"
	"gopkg.in/yaml.v3"

	// Import components to register them
	_ "github.com/synadia-io/connect-runtime-wombat/components"
)

var _ = Describe("Component Validation", func() {
	var (
		componentsDir string
		testConfigDir string
		validTypes    = []string{"bool", "int", "object", "scanner", "string", "expression", "condition"}
		validKinds    = []string{"scalar", "map", "list"}
	)

	BeforeEach(func() {
		// Get the project root (two levels up from test/full)
		projectRoot := filepath.Join("..", "..")
		componentsDir = filepath.Join(projectRoot, ".connect")
		testConfigDir = filepath.Join(projectRoot, "test", "component_validation", "test_configs")

		// Create test configs directory
		Expect(os.MkdirAll(testConfigDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		// Clean up test configs
		if err := os.RemoveAll(testConfigDir); err != nil {
			GinkgoLogr.Error(err, "failed to remove test config directory", "dir", testConfigDir)
		}
	})

	Describe("Scanner Components", func() {
		It("should validate all scanner components", func() {
			results := testComponents(filepath.Join(componentsDir, "scanners"), "scanner", testConfigDir)

			for _, result := range results {
				if !result.Success {
					Fail(fmt.Sprintf("Scanner %s failed validation: %s", result.Component, result.Error))
				}
			}

			Expect(len(results)).To(Equal(5), "Expected 5 scanner components")
		})
	})

	Describe("Source Components", func() {
		It("should validate all source components", func() {
			results := testComponents(filepath.Join(componentsDir, "sources"), "input", testConfigDir)

			for _, result := range results {
				if !result.Success {
					Fail(fmt.Sprintf("Source %s failed validation: %s", result.Component, result.Error))
				}
			}

			Expect(len(results)).To(Equal(33), "Expected 33 source components")
		})
	})

	Describe("Sink Components", func() {
		It("should validate all sink components", func() {
			results := testComponents(filepath.Join(componentsDir, "sinks"), "output", testConfigDir)

			for _, result := range results {
				if !result.Success {
					Fail(fmt.Sprintf("Sink %s failed validation: %s", result.Component, result.Error))
				}
			}

			Expect(len(results)).To(Equal(41), "Expected 41 sink components")
		})
	})

	Describe("Field Type and Kind Validation", func() {
		It("should reject invalid type values in all component specs", func() {
			// Test all component directories
			componentDirs := []string{
				filepath.Join(componentsDir, "scanners"),
				filepath.Join(componentsDir, "sources"),
				filepath.Join(componentsDir, "sinks"),
			}

			var allErrors []string

			for _, dir := range componentDirs {
				files, err := os.ReadDir(dir)
				if err != nil {
					continue
				}

				for _, file := range files {
					if !strings.HasSuffix(file.Name(), ".yml") && !strings.HasSuffix(file.Name(), ".yaml") {
						continue
					}

					componentPath := filepath.Join(dir, file.Name())
					errors := validateComponentSpec(componentPath, validTypes, validKinds)
					allErrors = append(allErrors, errors...)
				}
			}

			if len(allErrors) > 0 {
				Fail(fmt.Sprintf("Found %d validation errors:\n%s", len(allErrors), strings.Join(allErrors, "\n")))
			}
		})
	})

	It("should generate a comprehensive test report", func() {
		allResults := []TestResult{}

		// Test all component types
		scannerResults := testComponents(filepath.Join(componentsDir, "scanners"), "scanner", testConfigDir)
		allResults = append(allResults, scannerResults...)

		sourceResults := testComponents(filepath.Join(componentsDir, "sources"), "input", testConfigDir)
		allResults = append(allResults, sourceResults...)

		sinkResults := testComponents(filepath.Join(componentsDir, "sinks"), "output", testConfigDir)
		allResults = append(allResults, sinkResults...)

		// Generate summary
		successCount := 0
		for _, result := range allResults {
			if result.Success {
				successCount++
			}
		}

		// Save results to JSON
		resultsFile := filepath.Join(testConfigDir, "..", "test_results.json")
		file, err := os.Create(resultsFile)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			if err := file.Close(); err != nil {
				GinkgoLogr.Error(err, "failed to close results file")
			}
		}()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		Expect(encoder.Encode(allResults)).To(Succeed())

		// All components should pass
		Expect(successCount).To(Equal(len(allResults)), "All components should pass validation")
		Expect(len(allResults)).To(Equal(79), "Expected 79 total components")
	})
})

type ComponentSpec struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Kind        string `yaml:"kind"`
	Description string `yaml:"description"`
}

type TestResult struct {
	Component  string        `json:"component"`
	Type       string        `json:"type"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	ConfigPath string        `json:"config_path"`
	Duration   time.Duration `json:"duration"`
}

type ComponentSpecValidation struct {
	Fields []FieldSpecValidation `yaml:"fields"`
}

type FieldSpecValidation struct {
	Name   string                `yaml:"name"`
	Type   string                `yaml:"type"`
	Kind   string                `yaml:"kind"`
	Fields []FieldSpecValidation `yaml:"fields,omitempty"`
}

func testComponents(dir string, wombatType string, testConfigDir string) []TestResult {
	results := []TestResult{}

	files, err := os.ReadDir(dir)
	if err != nil {
		return results
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		componentPath := filepath.Join(dir, file.Name())
		start := time.Now()

		// Read component spec
		spec, err := readComponentSpec(componentPath)
		if err != nil {
			results = append(results, TestResult{
				Component:  file.Name(),
				Type:       wombatType,
				Success:    false,
				Error:      fmt.Sprintf("Failed to read spec: %v", err),
				ConfigPath: componentPath,
				Duration:   time.Since(start),
			})
			continue
		}

		// Generate wombat config
		configPath := filepath.Join(testConfigDir, fmt.Sprintf("%s_%s.yaml", wombatType, strings.TrimSuffix(file.Name(), ".yml")))
		if err := generateWombatConfig(spec.Name, wombatType, configPath); err != nil {
			results = append(results, TestResult{
				Component:  spec.Name,
				Type:       wombatType,
				Success:    false,
				Error:      fmt.Sprintf("Failed to generate config: %v", err),
				ConfigPath: configPath,
				Duration:   time.Since(start),
			})
			continue
		}

		// Test with wombat
		if err := testWombatConfig(configPath); err != nil {
			results = append(results, TestResult{
				Component:  spec.Name,
				Type:       wombatType,
				Success:    false,
				Error:      fmt.Sprintf("Wombat test failed: %v", err),
				ConfigPath: configPath,
				Duration:   time.Since(start),
			})
		} else {
			results = append(results, TestResult{
				Component:  spec.Name,
				Type:       wombatType,
				Success:    true,
				ConfigPath: configPath,
				Duration:   time.Since(start),
			})
		}
	}

	return results
}

func readComponentSpec(path string) (*ComponentSpec, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close() // Ignore close error in defer
	}()

	var spec ComponentSpec
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

func generateWombatConfig(componentName string, wombatType string, outputPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// Generate minimal valid config based on type
	var config string

	switch wombatType {
	case "input":
		minimalConfig := getMinimalConfig(componentName, wombatType)
		config = fmt.Sprintf(`input:
  %s:%s

output:
  drop: {}
`, componentName, minimalConfig)

	case "output":
		minimalConfig := getMinimalConfig(componentName, wombatType)
		config = fmt.Sprintf(`input:
  generate:
    count: 1
    mapping: |
      root = {"test": "data"}

output:
  %s:%s
`, componentName, minimalConfig)

	case "scanner":
		scannerConfig := getScannerConfig(componentName)
		config = fmt.Sprintf(`input:
  stdin:%s

output:
  drop: {}
`, scannerConfig)
	}

	return os.WriteFile(outputPath, []byte(config), 0644)
}

func getScannerConfig(scannerName string) string {
	switch scannerName {
	case "decompress":
		return `
    scanner:
      decompress:
        algorithm: gzip`
	case "json_documents":
		return `
    scanner:
      json_documents: {}`
	case "lines":
		return `
    scanner:
      lines: {}`
	case "tar":
		return `
    scanner:
      tar: {}`
	case "to_the_end":
		return `
    scanner:
      to_the_end: {}`
	default:
		return fmt.Sprintf(`
    scanner:
      %s: {}`, scannerName)
	}
}

func getMinimalConfig(componentName string, wombatType string) string {
	// Return minimal required configuration for known components
	switch componentName {
	// NATS family
	case "nats", "nats_jetstream":
		return `
    urls: ["nats://localhost:4222"]
    subject: "test.subject"`
	case "nats_kv":
		if wombatType == "input" {
			return `
    urls: ["nats://localhost:4222"]
    bucket: "test-bucket"`
		}
		return `
    urls: ["nats://localhost:4222"]
    bucket: "test-bucket"
    key: "test-key"`

	// Generate
	case "generate":
		return `
    count: 1
    mapping: |
      root = {"test": "data"}`

	// HTTP
	case "http_client":
		return `
    url: "http://localhost:8080"`

	// Kafka
	case "kafka_franz":
		if wombatType == "input" {
			return `
    seed_brokers: ["localhost:9092"]
    topics: ["test-topic"]
    consumer_group: "test-group"`
		}
		return `
    seed_brokers: ["localhost:9092"]
    topic: "test-topic"`

	// Redis family
	case "redis_list":
		return `
    url: "redis://localhost:6379"
    key: "test-list"`
	case "redis_pubsub":
		if wombatType == "input" {
			return `
    url: "redis://localhost:6379"
    channels: ["test-channel"]`
		}
		return `
    url: "redis://localhost:6379"
    channel: "test-channel"`
	case "redis_scan":
		return `
    url: "redis://localhost:6379"`
	case "redis_streams":
		if wombatType == "input" {
			return `
    url: "redis://localhost:6379"
    streams: ["test-stream"]`
		}
		return `
    url: "redis://localhost:6379"
    stream: "test-stream"`
	case "redis_hash":
		return `
    url: "redis://localhost:6379"
    key: "test-hash"`

	// Elasticsearch/OpenSearch
	case "elasticsearch_v8":
		return `
    urls: ["http://localhost:9200"]
    action: "index"
    id: ${!counter()}-${!timestamp_unix()}
    index: "test-index"`
	case "opensearch":
		return `
    urls: ["http://localhost:9200"]
    index: "test-index"
    action: "index"
    id: "${!json()}_id"`

	// MongoDB
	case "mongodb":
		if wombatType == "input" {
			return `
    url: "mongodb://localhost:27017"
    database: "test"
    collection: "test"
    query: |
      root = {}
    `
		}
		return `
    url: "mongodb://localhost:27017"
    database: "test"
    collection: "test"`
	case "mongodb_change_stream":
		return `
    uri: "mongodb://localhost:27017"
    database: "test"
    collection: "test"`

	// AWS services
	case "aws_s3":
		return `
    bucket: "test-bucket"
    region: "us-east-1"`
	case "aws_sqs":
		return `
    url: "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
    region: "us-east-1"`
	case "aws_kinesis":
		if wombatType == "input" {
			return `
    streams: ["test-stream"]
    region: "us-east-1"`
		}
		return `
    stream: "test-stream"
    region: "us-east-1"
    partition_key: "test-key"`
	case "aws_kinesis_firehose":
		return `
    stream: "test-stream"
    region: "us-east-1"`
	case "aws_sns":
		return `
    topic_arn: "arn:aws:sns:us-east-1:123456789012:test-topic"
    region: "us-east-1"`
	case "aws_dynamodb":
		return `
    table: "test-table"
    region: "us-east-1"`

	// AMQP
	case "amqp_0_9":
		if wombatType == "input" {
			return `
    urls: ["amqp://localhost:5672"]
    queue: "test-queue"`
		}
		return `
    urls: ["amqp://localhost:5672"]
    exchange: "test-exchange"`
	case "amqp_1":
		if wombatType == "input" {
			return `
    url: "amqp://localhost:5672"
    source_address: "test-queue"`
		}
		return `
    url: "amqp://localhost:5672"
    target_address: "test-queue"`

	// MQTT
	case "mqtt":
		return `
    urls: ["tcp://localhost:1883"]
    topics: ["test/topic"]`

	// Pulsar
	case "pulsar":
		if wombatType == "input" {
			return `
    url: "pulsar://localhost:6650"
    topics: ["test-topic"]
    subscription_name: "test-sub"`
		}
		return `
    url: "pulsar://localhost:6650"
    topic: "test-topic"`

	// NSQ
	case "nsq":
		if wombatType == "input" {
			return `
    nsqd_tcp_addresses: ["localhost:4150"]
    topic: "test"
    channel: "test-channel"`
		}
		return `
    nsqd_tcp_address: "localhost:4150"
    topic: "test"`

	// Azure services
	case "azure_blob_storage":
		return `
    storage_account: "test"
    storage_access_key: "test"
    container: "test"`
	case "azure_cosmosdb":
		if wombatType == "input" {
			return `
    endpoint: "https://test.documents.azure.com:443/"
    account_key: "test-key"
    database: "test"
    container: "test"
    query: "SELECT * FROM c"
    partition_keys_map: |
      root = {"id": this.id}`
		}
		return `
    endpoint: "https://test.documents.azure.com:443/"
    account_key: "test-key"
    database: "test"
    container: "test"
    partition_keys_map: |
      root = {"id": this.id}`
	case "azure_queue_storage":
		return `
    storage_account: "test"
    storage_access_key: "test"
    queue_name: "test"`
	case "azure_table_storage":
		return `
    storage_account: "test"
    storage_access_key: "test"
    table_name: "test"`
	case "azure_data_lake_gen2":
		return `
    storage_account: "test"
    storage_access_key: "test"
    filesystem: "test"`

	// GCP services
	case "gcp_cloud_storage":
		return `
    bucket: "test-bucket"`
	case "gcp_pubsub":
		if wombatType == "input" {
			return `
    project: "test-project"
    subscription: "test-subscription"`
		}
		return `
    project: "test-project"
    topic: "test-topic"`
	case "gcp_bigquery":
		return `
    project: "test-project"
    dataset: "test-dataset"
    table: "test-table"`
	case "gcp_bigquery_select":
		return `
    project: "test-project"
    table: "test-dataset.test-table"`
	case "gcp_bigtable":
		return `
    project: "test-project"
    instance: "test-instance"
    table: "test-table"`

	// Other databases
	case "cassandra":
		return `
    addresses: ["localhost:9042"]
    query: "INSERT INTO test.table (id, data) VALUES (?, ?)"`
	case "couchbase":
		return `
    url: "couchbase://localhost"
    bucket: "test-bucket"
    id: "${!json()}_id"
    content: |
      root = this`
	case "cypher":
		return `
    uri: "neo4j://localhost:7687"
    cypher: "CREATE (n:Test {data: $data})"`

	// Other services
	case "hdfs":
		return `
    hosts: ["localhost:9000"]
    directory: "/test"`
	case "socket":
		return `
    network: "tcp"
    address: "localhost:9999"`
	case "discord":
		return `
    channel_id: "123456789"
    bot_token: "test-token"`
	case "pusher":
		return `
    appId: "test-app"
    key: "test-key"
    secret: "test-secret"
    cluster: "us-east-1"
    channel: "test-channel"
    event: "test-event"`
	case "sftp":
		return `
    address: "localhost:22"
    path: "/test/file.txt"
    credentials:
      username: "test"
      password: "test"`
	case "timeplus":
		if wombatType == "input" {
			return `
    url: "http://localhost:8000"
    apikey: "test-key"
    query: "SELECT * FROM test"`
		}
		return `
    url: "http://localhost:8000"
    apikey: "test-key"
    stream: "test"`
	case "ww_mqtt_3":
		if wombatType == "input" {
			return `
    urls: ["tcp://localhost:1883"]
    filters: {"test/topic": 1}
    client_id: "test-client"`
		}
		return `
    urls: ["tcp://localhost:1883"]
    topic: "test/topic"
    client_id: "test-client"`
	case "sql_raw":
		if wombatType == "input" {
			return `
    driver: "clickhouse"
    dsn: clickhouse://username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60
    query: SELECT * FROM footable;`
		}
		return `
    driver: "clickhouse"
    dsn: clickhouse://username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60
    max_in_flight: 64`
	case "snowflake_put":
		return `
    account: wombat
    user: foobar
    role: test_role
    database: test_db
    warehouse: test_warehouse
    schema: test_schema
    stage: "@%MY_TABLE"
    path: "foo/bar/baz"`
	default:
		// Return empty config for components we don't have defaults for
		return " {}"
	}
}

func testWombatConfig(configPath string) error {
	// Read the config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Use the service.Builder to validate the config
	builder := service.NewStreamBuilder()

	// Try to build from the config - this will validate the configuration
	if err := builder.SetYAML(string(configData)); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// The config is valid if we reach here
	return nil
}

func validateComponentSpec(path string, validTypes, validKinds []string) []string {
	var errors []string

	file, err := os.Open(path)
	if err != nil {
		return []string{fmt.Sprintf("Failed to open %s: %v", path, err)}
	}
	defer func() {
		if err := file.Close(); err != nil {
			GinkgoLogr.Error(err, "failed to close file", "path", path)
		}
	}()

	var spec ComponentSpecValidation
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&spec); err != nil {
		return []string{fmt.Sprintf("Failed to parse %s: %v", path, err)}
	}

	fieldErrors := validateFields(spec.Fields, path, validTypes, validKinds, "")
	errors = append(errors, fieldErrors...)

	return errors
}

func validateFields(fields []FieldSpecValidation, filePath string, validTypes, validKinds []string, parentPath string) []string {
	var errors []string

	for _, field := range fields {
		fieldPath := field.Name
		if parentPath != "" {
			fieldPath = parentPath + "." + field.Name
		}

		// Validate type
		if field.Type != "" && !contains(validTypes, field.Type) {
			errors = append(errors, fmt.Sprintf("%s: Field '%s' has invalid type '%s'. Valid types are: %v",
				filepath.Base(filePath), fieldPath, field.Type, validTypes))
		}

		// Validate kind
		if field.Kind != "" && !contains(validKinds, field.Kind) {
			errors = append(errors, fmt.Sprintf("%s: Field '%s' has invalid kind '%s'. Valid kinds are: %v",
				filepath.Base(filePath), fieldPath, field.Kind, validKinds))
		}

		// Recursively validate nested fields
		if len(field.Fields) > 0 {
			nestedErrors := validateFields(field.Fields, filePath, validTypes, validKinds, fieldPath)
			errors = append(errors, nestedErrors...)
		}
	}

	return errors
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
