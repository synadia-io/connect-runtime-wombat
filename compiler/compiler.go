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
    if steps.Producer != nil && steps.Source != nil {
        // -- an inlet has a producer and a source
        mainCfg["input"], err = compileSource(steps)
        if err != nil {
            return "", fmt.Errorf("input: %w", err)
        }

        mainCfg["output"], err = compileProducer(steps)
        if err != nil {
            return "", fmt.Errorf("target: %w", err)
        }
    } else if steps.Consumer != nil && steps.Sink != nil {
        // -- an outlet has a consumer and a sink
        mainCfg["input"], err = compileConsumer(steps)
        if err != nil {
            return "", fmt.Errorf("source: %w", err)
        }

        mainCfg["output"], err = compileSink(steps)
        if err != nil {
            return "", fmt.Errorf("output: %w", err)
        }
    } else {
        return "", fmt.Errorf("invalid steps")
    }

    b, err := yaml.Marshal(mainCfg)
    if err != nil {
        return "", fmt.Errorf("marshal: %w", err)
    }

    return string(b), nil

}
