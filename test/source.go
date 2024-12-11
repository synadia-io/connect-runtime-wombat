package test

import (
	"github.com/synadia-labs/vent/public/control"
)

func InvalidSource() control.Source {
	return control.Source{
		Type: "invalid",
	}
}

func GenerateSource() control.Source {
	return control.Source{
		Type: "generate",
		Config: map[string]any{
			"count":   5,
			"mapping": "root = \"hello world\"",
		},
	}
}
