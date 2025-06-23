package compiler

import (
	"bytes"
	"encoding/json"
)

// Frag creates a new Fragment for building configuration structures.
// This is the entry point for the Fragment builder pattern.
func Frag() Fragment {
	return make(Fragment)
}

// Fragment is a builder type for constructing nested configuration structures
// that can be serialized to YAML. It provides a fluent API for adding
// different types of configuration values.
//
// Example usage:
//
//	cfg := Frag().
//	    String("type", "http_client").
//	    Fragment("config", Frag().
//	        String("url", "https://example.com").
//	        Int("timeout", 30))
type Fragment map[string]any

// Fragment adds a nested Fragment to the configuration.
// This is used for creating hierarchical configuration structures.
func (f Fragment) Fragment(key string, fragment Fragment) Fragment {
	f[key] = fragment
	return f
}

// StringMap adds a map of string key-value pairs to the configuration.
// Useful for headers, labels, and other string-to-string mappings.
func (f Fragment) StringMap(key string, m map[string]string) Fragment {
	f[key] = m
	return f
}

// Map adds a map of arbitrary key-value pairs to the configuration.
// The values can be of any type that is YAML-serializable.
func (f Fragment) Map(key string, m map[string]any) Fragment {
	f[key] = m
	return f
}

// Fragments adds an array of Fragments to the configuration.
// This is typically used for lists of processors or multiple configurations.
func (f Fragment) Fragments(key string, fragments ...Fragment) Fragment {
	f[key] = fragments
	return f
}

// Strings adds an array of strings to the configuration.
// Useful for lists like topics, subjects, or allowed values.
func (f Fragment) Strings(key string, values ...string) Fragment {
	f[key] = values
	return f
}

// String adds a string value to the configuration.
// This is the most common method for simple string configuration values.
func (f Fragment) String(key string, value string) Fragment {
	f[key] = value
	return f
}

// StringP adds a string pointer to the configuration.
// If the pointer is nil, the key is not added, making this useful for optional fields.
func (f Fragment) StringP(key string, value *string) Fragment {
	if value != nil {
		f[key] = *value
	}
	return f
}

// Int adds an integer value to the configuration.
// Used for numeric configuration like ports, timeouts, or counts.
func (f Fragment) Int(key string, value int) Fragment {
	f[key] = value
	return f
}

// IntP adds an integer pointer to the configuration.
// If the pointer is nil, the key is not added, making this useful for optional numeric fields.
func (f Fragment) IntP(key string, value *int) Fragment {
	if value != nil {
		f[key] = *value
	}
	return f
}

// Bool adds a boolean value to the configuration.
// Used for flags and boolean configuration options.
func (f Fragment) Bool(key string, value bool) Fragment {
	f[key] = value
	return f
}

// BoolP adds a boolean pointer to the configuration.
// If the pointer is nil, the key is not added, making this useful for optional boolean fields.
func (f Fragment) BoolP(key string, value *bool) Fragment {
	if value != nil {
		f[key] = *value
	}
	return f
}

// EqualsMap compares the Fragment with a map[string]any for deep equality.
// This method uses JSON marshaling for comparison to ensure consistent results
// across different type representations.
func (f Fragment) EqualsMap(exp map[string]any) bool {
	b1, err1 := json.Marshal(f)
	b2, err2 := json.Marshal(exp)
	if err1 != nil || err2 != nil {
		return false
	}
	return bytes.Equal(b1, b2)
}
