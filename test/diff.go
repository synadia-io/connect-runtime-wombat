package test

import (
	"fmt"
	"github.com/r3labs/diff/v3"
	"gopkg.in/yaml.v3"
)

func DiffYaml(expected, actual string) (diff.Changelog, error) {
	var am map[string]interface{}
	var bm map[string]interface{}

	if err := yaml.Unmarshal([]byte(expected), &am); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the expected yaml: %w", err)
	}

	if err := yaml.Unmarshal([]byte(actual), &bm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the actual yaml: %w", err)
	}

	return diff.Diff(am, bm)
}
