package test

import (
    "github.com/synadia-io/connect/builders"
    "github.com/synadia-io/connect/model"
)

func InvalidSource() model.SourceStep {
    return model.SourceStep{
        Type: "invalid",
    }
}

func GenerateSource() *builders.SourceStepBuilder {
    return builders.SourceStep("generate").
        SetString("mapping", "root = \"hello world\"").
        SetInt("count", 5)
}
