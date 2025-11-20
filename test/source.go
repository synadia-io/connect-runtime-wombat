package test

import (
	"github.com/synadia-io/connect/v2/builders"
)

func InvalidSource() *builders.SourceStepBuilder {
	return builders.SourceStep("invalid")
}

func GenerateSource() *builders.SourceStepBuilder {
	return builders.SourceStep("generate").
		SetString("mapping", "root = \"hello world\"").
		SetInt("count", 5)
}
