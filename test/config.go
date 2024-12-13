package test

import "github.com/synadia-io/connect/model"

func TestConfig(steps model.Steps) model.ConnectorConfig {
	return model.ConnectorConfig{
		Steps: &steps,
	}
}
