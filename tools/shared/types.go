package shared

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
