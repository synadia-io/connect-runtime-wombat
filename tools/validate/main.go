package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/synadia-io/connect-runtime-wombat/tools/shared"
	"gopkg.in/yaml.v3"
)

// Field patterns to ignore during validation - vendor-specific and advanced features
var ignoredFieldPatterns = []string{
	// Synadia NATS-specific authentication extensions
	//"auth.nkey_file",
	//"auth.nkey",
	//"auth.user_credentials_file",
	//
	//// Advanced/experimental features outside Connect Runtime scope
	//"inject_tracing_map",
	//"extract_tracing_map",
	//
	//// Enterprise features
	//"batching.processors", // Complex batching processors
	//
	//// TLS configurations - Connect Runtime uses simplified TLS configuration
	//// Many components have their own TLS implementation that differs from Benthos
	//"tls", // General TLS config for NATS, MQTT, Pulsar, etc.
	//"tls.enabled",
	//"tls.skip_cert_verify",
	//"tls.enable_renegotiation",
	//"tls.root_cas",
	//"tls.client_certs",
	//"tls.client_certs[].cert",
	//"tls.client_certs[].key",
	//"tls.client_certs[].password",
	//
	//// Complex batching and database-specific configurations
	//"write_concern", // MongoDB write concern - advanced configuration
	//"write_concern.w",
	//"write_concern.j",
	//"write_concern.w_timeout",
	//"batching", // Complex batching configurations not exposed in Connect
	//"batching.count",
	//"batching.byte_size",
	//"batching.period",
	//"batching.check",
	//
	//// Vendor-specific extensions not in base Benthos schema
	//// AWS-specific SASL extensions
	//"sasl[].aws.region",
	//"sasl[].aws.endpoint",
	//"sasl[].aws.credentials",
	//"region",      // AWS region fields
	//"endpoint",    // AWS endpoint fields
	//"credentials", // AWS credentials fields
	//
	//// Azure-specific configurations
	//"targets_input", // Azure blob storage specific
	//"endpoint",      // Azure endpoint configurations
	//"account_key",   // Azure account key
	//
	//// BigQuery-specific fields
	//"prefix", // GCP BigQuery prefix
	//"suffix", // GCP BigQuery suffix
	//
	//// GCP PubSub specific
	//"create_subscription", // GCP PubSub subscription creation
	//
	//// Enterprise and advanced features not exposed in Connect Runtime
	//"jwt", // JWT authentication - advanced feature
	//"jwt.enabled",
	//"jwt.private_key_file",
	//"jwt.signing_method",
	//"jwt.claims",
	//"rate_limit", // Rate limiting - advanced feature
	//"rate_limit.count",
	//"rate_limit.interval",
	//
	//// Component-specific edge cases
	//// AMQP queue declaration differences between Connect and Benthos
	//"queue.queue_declare.enabled",
	//"queue.queue_declare.durable",
	//"queue.queue_declare.auto_delete",
	//
	//// HTTP client OAuth endpoint parameters not in base schema
	//"oauth.oauth2.endpoint_params",
	//"oauth2.endpoint_params",
	//
	//// Additional AWS and GCP specific advanced configurations
	//"backoff", // AWS backoff configuration - advanced feature
	//"backoff.initial_interval",
	//"backoff.max_interval",
	//"backoff.max_elapsed_time",
	//"csv", // GCP BigQuery CSV format - advanced feature
	//"csv.header",
	//"csv.delimiter",
	//"flow_control", // GCP PubSub flow control - advanced feature
	//"flow_control.max_messages",
	//"flow_control.max_bytes",
}

// ConnectSpec represents our .connect YAML specification
type ConnectSpec struct {
	Name        string         `yaml:"name"`
	Type        string         `yaml:"type"`
	Summary     string         `yaml:"summary,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Fields      []ConnectField `yaml:"fields"`
}

// ConnectField represents a field in our .connect specification
type ConnectField struct {
	Path        string         `yaml:"path"`
	Kind        string         `yaml:"kind,omitempty"`
	Type        string         `yaml:"type,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Default     interface{}    `yaml:"default,omitempty"`
	Fields      []ConnectField `yaml:"fields,omitempty"`
}

// ValidationResult tracks validation issues
type ValidationResult struct {
	ComponentName string
	ComponentType string
	Issues        []ValidationIssue
}

type ValidationIssue struct {
	Severity string // "error", "warning", "info"
	Path     string
	Message  string
}

func main() {
	fmt.Println("Validating .connect specs against extracted Benthos schemas...")

	var allResults []ValidationResult
	hasErrors := false

	// Validate sinks (outputs)
	sinkResults, err := validateComponentType("sinks", "output")
	if err != nil {
		fmt.Printf("Error validating sinks: %v\n", err)
		os.Exit(1)
	}
	allResults = append(allResults, sinkResults...)

	// Validate sources (inputs)
	sourceResults, err := validateComponentType("sources", "input")
	if err != nil {
		fmt.Printf("Error validating sources: %v\n", err)
		os.Exit(1)
	}
	allResults = append(allResults, sourceResults...)

	// Validate processors
	processorResults, err := validateComponentType("processors", "processor")
	if err != nil {
		fmt.Printf("Error validating processors: %v\n", err)
		os.Exit(1)
	}
	allResults = append(allResults, processorResults...)

	// Report results
	fmt.Printf("\n=== VALIDATION RESULTS ===\n\n")

	for _, result := range allResults {
		if len(result.Issues) == 0 {
			fmt.Printf("✅ %s/%s: OK\n", result.ComponentType, result.ComponentName)
			continue
		}

		fmt.Printf("❌ %s/%s: %d issues found\n", result.ComponentType, result.ComponentName, len(result.Issues))

		for _, issue := range result.Issues {
			icon := getIssueIcon(issue.Severity)
			if issue.Severity == "error" {
				hasErrors = true
			}
			fmt.Printf("   %s [%s] %s: %s\n", icon, strings.ToUpper(issue.Severity), issue.Path, issue.Message)
		}
		fmt.Println()
	}

	// Summary
	totalComponents := len(allResults)
	componentsWithIssues := 0
	totalIssues := 0

	for _, result := range allResults {
		if len(result.Issues) > 0 {
			componentsWithIssues++
		}
		totalIssues += len(result.Issues)
	}

	fmt.Printf("=== SUMMARY ===\n")
	fmt.Printf("Total components validated: %d\n", totalComponents)
	fmt.Printf("Components with issues: %d\n", componentsWithIssues)
	fmt.Printf("Total issues found: %d\n", totalIssues)

	if hasErrors {
		fmt.Printf("\n❌ Validation failed with errors\n")
		os.Exit(1)
	} else if totalIssues > 0 {
		fmt.Printf("\n⚠️  Validation completed with warnings\n")
	} else {
		fmt.Printf("\n✅ All validations passed\n")
	}
}

func validateComponentType(connectDir, schemaType string) ([]ValidationResult, error) {
	var results []ValidationResult

	connectPath := filepath.Join(".connect", connectDir)

	// Check if directory exists
	if _, err := os.Stat(connectPath); os.IsNotExist(err) {
		fmt.Printf("Directory %s does not exist, skipping...\n", connectPath)
		return results, nil
	}

	// Read all YAML files in the directory
	files, err := filepath.Glob(filepath.Join(connectPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob YAML files in %s: %w", connectPath, err)
	}

	for _, file := range files {
		result, err := validateSingleComponent(file, schemaType)
		if err != nil {
			fmt.Printf("Warning: failed to validate %s: %v\n", file, err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func validateSingleComponent(connectFile, schemaType string) (ValidationResult, error) {
	// Load .connect spec
	connectSpec, err := loadConnectSpec(connectFile)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("failed to load connect spec: %w", err)
	}

	result := ValidationResult{
		ComponentName: connectSpec.Name,
		ComponentType: schemaType,
		Issues:        []ValidationIssue{},
	}

	// Load corresponding Benthos schema
	schemaFile := filepath.Join("schemas", fmt.Sprintf("%s_%s.json", schemaType, connectSpec.Name))
	benthos, err := loadBenthosSchema(schemaFile)
	if err != nil {
		result.Issues = append(result.Issues, ValidationIssue{
			Severity: "warning",
			Path:     "",
			Message:  fmt.Sprintf("No Benthos schema found at %s", schemaFile),
		})
		return result, nil
	}

	// Validate structure
	result.Issues = append(result.Issues, validateFields(connectSpec.Fields, benthos.Fields, "")...)

	return result, nil
}

func loadConnectSpec(filename string) (*ConnectSpec, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var spec ConnectSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

func loadBenthosSchema(filename string) (*shared.ComponentSchema, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var schema shared.ComponentSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

func validateFields(connectFields []ConnectField, benthosFields []shared.FieldSchema, parentPath string) []ValidationIssue {
	var issues []ValidationIssue

	// Build maps for easier lookup
	connectMap := make(map[string]ConnectField)
	benthosMap := make(map[string]shared.FieldSchema)

	for _, cf := range connectFields {
		connectMap[cf.Path] = cf
	}

	for _, bf := range benthosFields {
		benthosMap[bf.Name] = bf
	}

	// Check for OAuth-specific structural issues
	issues = append(issues, checkOAuthStructure(connectFields, parentPath)...)

	// Check each field in our connect spec
	for _, connectField := range connectFields {
		fieldPath := buildFieldPath(parentPath, connectField.Path)

		// Try multiple matching strategies for better field resolution
		benthosField, exists := findMatchingBenthosField(connectField, benthosFields, parentPath)
		if !exists {
			// Check if this field should be ignored (vendor-specific, advanced features, etc.)
			if shouldIgnoreField(connectField.Path) || shouldIgnoreField(fieldPath) {
				continue
			}

			fieldName := extractLastPathComponent(connectField.Path)
			issues = append(issues, ValidationIssue{
				Severity: "warning",
				Path:     fieldPath,
				Message:  fmt.Sprintf("Field not found in Benthos schema: %s", fieldName),
			})
			continue
		}

		// Validate field type if specified
		if connectField.Type != "" {
			expectedType := mapConnectTypeToBenthos(connectField)
			if expectedType != benthosField.Type {
				issues = append(issues, ValidationIssue{
					Severity: "error",
					Path:     fieldPath,
					Message:  fmt.Sprintf("Type mismatch: connect=%s, benthos=%s", expectedType, benthosField.Type),
				})
			}
		}

		// Recursively validate children
		if len(connectField.Fields) > 0 {
			issues = append(issues, validateFields(connectField.Fields, benthosField.Children, fieldPath)...)
		}
	}

	// Check for missing required fields in our spec
	for _, benthosField := range benthosFields {
		if benthosField.Required {
			// Try to find matching Connect field by checking both field name and full path
			fieldPath := buildFieldPath(parentPath, benthosField.Name)
			found := false

			// Check if field exists in connectMap by name (for root level fields)
			if _, exists := connectMap[benthosField.Name]; exists {
				found = true
			}

			// Check if field exists by full path (for nested fields)
			if !found {
				for connectPath := range connectMap {
					if extractLastPathComponent(connectPath) == benthosField.Name {
						found = true
						break
					}
				}
			}

			if !found {
				// Skip ignored vendor-specific and advanced fields
				if shouldIgnoreField(benthosField.FullName) {
					continue
				}

				issues = append(issues, ValidationIssue{
					Severity: "warning",
					Path:     fieldPath,
					Message:  "Required field missing from connect spec",
				})
			}
		}
	}

	return issues
}

func checkOAuthStructure(connectFields []ConnectField, parentPath string) []ValidationIssue {
	var issues []ValidationIssue

	// Look for OAuth structure issues
	for _, field := range connectFields {
		if field.Path == "oauth" {
			// Check if OAuth2 fields are incorrectly nested under OAuth1
			oauth2Fields := []string{"client_key", "client_secret", "scopes", "token_url"}

			for _, childField := range field.Fields {
				// Check if OAuth2 fields appear as direct children of oauth
				for _, oauth2Field := range oauth2Fields {
					if strings.HasPrefix(childField.Path, "oauth2."+oauth2Field) {
						fieldPath := buildFieldPath(parentPath, field.Path+"."+childField.Path)
						issues = append(issues, ValidationIssue{
							Severity: "error",
							Path:     fieldPath,
							Message:  fmt.Sprintf("OAuth2 field '%s' incorrectly nested under OAuth1 - should be at top level under separate oauth2 section", oauth2Field),
						})
					}
				}
			}
		}
	}

	return issues
}

func buildFieldPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func extractLastPathComponent(path string) string {
	// For paths like "tls.enabled", return "enabled"
	// For simple paths like "oauth", return "oauth"
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}

// findMatchingBenthosField uses multiple strategies to find the best matching Benthos field
// for a given Connect field, handling nested paths and different naming conventions
func findMatchingBenthosField(connectField ConnectField, benthosFields []shared.FieldSchema, parentPath string) (shared.FieldSchema, bool) {
	connectFieldName := extractLastPathComponent(connectField.Path)
	fullConnectPath := buildFieldPath(parentPath, connectField.Path)

	// Strategy 1: Direct name match (most common case)
	for _, bf := range benthosFields {
		if bf.Name == connectFieldName {
			return bf, true
		}
	}

	// Strategy 2: Match against full path for nested fields
	for _, bf := range benthosFields {
		if bf.FullName == fullConnectPath || bf.FullName == connectField.Path {
			return bf, true
		}
	}

	// Strategy 3: Handle array notation differences - Connect uses "field" while Benthos may use "field[]"
	cleanConnectPath := strings.ReplaceAll(connectField.Path, "[]", "")
	for _, bf := range benthosFields {
		cleanBenthosPath := strings.ReplaceAll(bf.FullName, "[]", "")
		if cleanBenthosPath == cleanConnectPath {
			return bf, true
		}
	}

	// Strategy 4: Partial path matching for complex nested structures
	for _, bf := range benthosFields {
		if strings.HasSuffix(bf.FullName, connectField.Path) || strings.HasSuffix(connectField.Path, bf.Name) {
			return bf, true
		}
	}

	return shared.FieldSchema{}, false
}

func mapConnectTypeToBenthos(field ConnectField) string {
	// Handle .connect kind + type combinations and map to Benthos types
	switch {
	case field.Kind == "list":
		// kind: list means array type in Benthos
		return "array"
	case field.Kind == "map":
		// kind: map means object type in Benthos
		return "object"
	case field.Kind == "scalar":
		// kind: scalar uses the type directly
		return field.Type
	case field.Type == "expression":
		// Connect "expression" type maps to Benthos "string" - expressions are stored as strings
		// This is a semantic enhancement: Connect explicitly labels expression-capable fields
		return "string"
	default:
		// No kind specified, use type directly
		return field.Type
	}
}

// shouldIgnoreField checks if a field should be ignored during validation
// Returns true for vendor-specific extensions and advanced features outside Connect Runtime scope
func shouldIgnoreField(fullName string) bool {
	for _, pattern := range ignoredFieldPatterns {
		if fullName == pattern {
			return true
		}
	}
	return false
}

func getIssueIcon(severity string) string {
	switch severity {
	case "error":
		return "🔴"
	case "warning":
		return "🟡"
	case "info":
		return "🔵"
	default:
		return "⚪"
	}
}
