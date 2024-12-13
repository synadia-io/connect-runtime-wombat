package compiler

import (
	"fmt"
	"github.com/synadia-io/connect/model"
	"gopkg.in/yaml.v3"

	_ "github.com/synadia-io/connect-runtime-wombat/components"
)

func Compile(steps model.Steps) (string, error) {
	mainCfg := map[string]any{
		"input":  nil,
		"output": nil,
		"metrics": map[string]any{
			"prometheus": map[string]any{},
		},
	}

	var err error
	switch steps.Kind() {
	case model.Inlet:
		mainCfg["input"], err = compileSource(steps)
		if err != nil {
			return "", fmt.Errorf("input: %w", err)
		}

		mainCfg["output"], err = compileProducer(steps)
		if err != nil {
			return "", fmt.Errorf("target: %w", err)
		}
	case model.Outlet:
		mainCfg["input"], err = compileConsumer(steps)
		if err != nil {
			return "", fmt.Errorf("source: %w", err)
		}

		mainCfg["output"], err = compileSink(steps)
		if err != nil {
			return "", fmt.Errorf("output: %w", err)
		}
	default:
		return "", fmt.Errorf("unknown connector kind %s", steps.Kind())
	}

	b, err := yaml.Marshal(mainCfg)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	return string(b), nil

}
