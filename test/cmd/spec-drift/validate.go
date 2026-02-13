package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func validate(connectPath, schemasDir string) error {
	fmt.Println("Validating .connect specs against extracted Benthos schemas...")

	var allResults []ValidationResult
	hasErrors := false

	// Validate sinks (outputs)
	sinkResults, err := validateComponentType(connectPath, "sinks", schemasDir, "output")
	if err != nil {
		return fmt.Errorf("error validating sinks: %w", err)
	}
	allResults = append(allResults, sinkResults...)

	// Validate sources (inputs)
	sourceResults, err := validateComponentType(connectPath, "sources", schemasDir, "input")
	if err != nil {
		return fmt.Errorf("error validating sources: %w", err)
	}
	allResults = append(allResults, sourceResults...)

	// Validate processors
	processorResults, err := validateComponentType(connectPath, "processors", schemasDir, "processor")
	if err != nil {
		return fmt.Errorf("error validating processors: %w", err)
	}
	allResults = append(allResults, processorResults...)

	hasErrors = outputText(allResults)
	if hasErrors {
		return fmt.Errorf("validation failed with errors")
	}
	return nil
}

func validateComponentType(connectBasePath, connectDir, schemasDir, schemaType string) ([]ValidationResult, error) {
	var results []ValidationResult

	fullConnectPath := filepath.Join(connectBasePath, connectDir)

	// Check if directory exists
	if _, err := os.Stat(fullConnectPath); os.IsNotExist(err) {
		fmt.Printf("Directory %s does not exist, skipping...\n", fullConnectPath)
		return results, nil
	}

	// Read all YAML files in the directory
	files, err := filepath.Glob(filepath.Join(fullConnectPath, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob YAML files in %s: %w", fullConnectPath, err)
	}

	for _, file := range files {
		result, err := validateSingleComponent(file, schemasDir, schemaType)
		if err != nil {
			fmt.Printf("Warning: failed to validate %s: %v\n", file, err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

func validateSingleComponent(connectFile, schemasDir, schemaType string) (ValidationResult, error) {
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
	schemaFile := filepath.Join(schemasDir, fmt.Sprintf("%s_%s.json", schemaType, connectSpec.Name))
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

func loadBenthosSchema(filename string) (*ComponentSchema, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var schema ComponentSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

func validateFields(connectFields []ConnectField, benthosFields []FieldSchema, parentPath string) []ValidationIssue {
	var issues []ValidationIssue

	// Build maps for easier lookup
	connectMap := make(map[string]ConnectField)
	benthosMap := make(map[string]FieldSchema)

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
func findMatchingBenthosField(connectField ConnectField, benthosFields []FieldSchema, parentPath string) (FieldSchema, bool) {
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

	return FieldSchema{}, false
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

func outputText(allResults []ValidationResult) bool {
	hasErrors := false

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
	} else if totalIssues > 0 {
		fmt.Printf("\n⚠️  Validation completed with warnings\n")
	} else {
		fmt.Printf("\n✅ All validations passed\n")
	}

	return hasErrors
}
