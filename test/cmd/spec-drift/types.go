package main

// ComponentSchema represents the extracted schema structure
type ComponentSchema struct {
	Name   string        `json:"name"`
	Type   string        `json:"type"`
	Fields []FieldSchema `json:"fields"`
}

// FieldSchema represents a field in the component configuration
type FieldSchema struct {
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default,omitempty"`
	Required    bool          `json:"required"`
	Children    []FieldSchema `json:"children,omitempty"`
	FullName    string        `json:"full_name"`
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
