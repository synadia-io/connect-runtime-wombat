package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redpanda-data/benthos/v4/public/service"
	_ "github.com/wombatwisdom/wombat/public/components/all"
)

func extract() (string, error) {
	fmt.Println("Extracting schemas from Benthos components...")

	env := service.GlobalEnvironment()

	schemasDir, err := os.MkdirTemp("", "wombat-schema-")
	if err != nil {
		fmt.Printf("❌Error creating temp dir for schemas: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Using dir: %v\n", schemasDir)

	}

	var outputCount int
	env.WalkOutputs(func(name string, config *service.ConfigView) {
		outputCount++
		if err := extractAndSaveComponentSchema(name, "output", config, schemasDir); err != nil {
			fmt.Printf("Error extracting schema for output %s: %v\n", name, err)
			return
		}
		fmt.Printf("✓ Extracted schema for output: %s\n", name)
	})
	fmt.Printf("Found %d output components\n", outputCount)

	var inputCount int
	env.WalkInputs(func(name string, config *service.ConfigView) {
		inputCount++
		if err := extractAndSaveComponentSchema(name, "input", config, schemasDir); err != nil {
			fmt.Printf("Error extracting schema for input %s: %v\n", name, err)
			return
		}
		fmt.Printf("✓ Extracted schema for input: %s\n", name)
	})
	fmt.Printf("Found %d input components\n", inputCount)

	// Extract processor specs
	var processorCount int
	env.WalkProcessors(func(name string, config *service.ConfigView) {
		processorCount++
		if err := extractAndSaveComponentSchema(name, "processor", config, schemasDir); err != nil {
			fmt.Printf("Error extracting schema for processor %s: %v\n", name, err)
			return
		}
		fmt.Printf("✓ Extracted schema for processor: %s\n", name)
	})
	fmt.Printf("Found %d processor components\n", processorCount)

	fmt.Printf("\nSchema extraction completed. Files saved to %s/\n", schemasDir)
	return schemasDir, nil
}

func extractAndSaveComponentSchema(name, componentType string, spec *service.ConfigView, schemasDir string) error {
	// Get template data which contains the structured field information
	templateData, err := spec.TemplateData()
	if err != nil {
		return fmt.Errorf("failed to get template data: %w", err)
	}

	// Convert template data to our schema format
	schema := ComponentSchema{
		Name:   name,
		Type:   componentType,
		Fields: convertTemplateFieldsToSchema(templateData.Fields),
	}

	// Save to JSON file
	filename := filepath.Join(schemasDir, fmt.Sprintf("%s_%s.json", componentType, name))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filename, err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		return fmt.Errorf("failed to encode schema: %w", err)
	}

	return nil
}

func convertTemplateFieldsToSchema(templateFields []service.TemplateDataPluginField) []FieldSchema {
	var fields []FieldSchema

	for _, tf := range templateFields {
		field := FieldSchema{
			Name:        extractFieldName(tf.FullName),
			FullName:    tf.FullName,
			Type:        tf.Type,
			Description: tf.Description,
			Required:    tf.DefaultMarshalled == "", // Simple heuristic
		}

		// Parse default value if available
		if tf.DefaultMarshalled != "" {
			var defaultVal interface{}
			if err := json.Unmarshal([]byte(tf.DefaultMarshalled), &defaultVal); err == nil {
				field.Default = defaultVal
			}
		}

		fields = append(fields, field)
	}

	// Build hierarchical structure from flat list
	return buildFieldHierarchy(fields)
}

func extractFieldName(fullName string) string {
	// Handle array notation: "tls.client_certs[].cert" -> "cert"
	// Remove array notation first, then extract field name
	cleanPath := strings.ReplaceAll(fullName, "[]", "")

	parts := []rune(cleanPath)
	lastDot := -1

	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot == -1 {
		return cleanPath
	}

	return string(parts[lastDot+1:])
}

func buildFieldHierarchy(flatFields []FieldSchema) []FieldSchema {
	// Group fields by their parent path
	fieldMap := make(map[string][]FieldSchema)
	rootFields := []FieldSchema{}

	for _, field := range flatFields {
		// Determine parent path
		parentPath := getParentPath(field.FullName)

		if parentPath == "" {
			// Root level field
			rootFields = append(rootFields, field)
		} else {
			// Child field
			fieldMap[parentPath] = append(fieldMap[parentPath], field)
		}
	}

	// Recursively build hierarchy
	return attachChildren(rootFields, fieldMap)
}

func getParentPath(fullName string) string {
	// Handle array notation: "tls.client_certs[].cert" -> "tls.client_certs"
	// Remove array notation first, then find parent
	cleanPath := strings.ReplaceAll(fullName, "[]", "")

	parts := []rune(cleanPath)
	lastDot := -1

	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot == -1 {
		return ""
	}

	return string(parts[:lastDot])
}

func attachChildren(fields []FieldSchema, fieldMap map[string][]FieldSchema) []FieldSchema {
	for i := range fields {
		if children, exists := fieldMap[fields[i].FullName]; exists {
			fields[i].Children = attachChildren(children, fieldMap)
		}
	}
	return fields
}
