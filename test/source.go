package test

import (
	"github.com/synadia-io/connect/model"
)

func InvalidSource() model.Source {
	return model.Source{
		Type: "invalid",
	}
}

func GenerateSource() model.Source {
	return model.Source{
		Type: "generate",
		Config: map[string]any{
			"count":   5,
			"mapping": "root = \"hello world\"",
		},
	}
}
