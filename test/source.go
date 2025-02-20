package test

import (
    "github.com/synadia-io/connect/model"
)

func InvalidSource() model.SourceStep {
    return model.SourceStep{
        Type: "invalid",
    }
}

func GenerateSource() model.SourceStep {
    return model.SourceStep{
        Type: "generate",
        Config: map[string]any{
            "count":   5,
            "mapping": "root = \"hello world\"",
        },
    }
}
