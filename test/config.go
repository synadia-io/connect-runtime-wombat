package test

import "github.com/synadia-labs/vent/public/control"

func TestConfig(steps control.Steps) control.ConnectorConfig {
	return control.ConnectorConfig{
		Steps: &steps,
	}
}
