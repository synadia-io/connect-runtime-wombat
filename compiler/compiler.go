package compiler

import (
    "fmt"
    "github.com/synadia-io/connect/model"
    "gopkg.in/yaml.v3"

    _ "github.com/synadia-io/connect-runtime-wombat/components"
)

func Compile(steps model.Steps) (string, error) {
    mainCfg := Frag().
        Fragment("metrics", Frag().
            Fragment("prometheus", Frag()))

    var err error
    if steps.Producer != nil && steps.Source != nil {
        producer, err := compileProducer(*steps.Producer)
        if err != nil {
            return "", fmt.Errorf("output: %w", err)
        }

        mainCfg.Fragment("input", compileSource(*steps.Source, steps.Transformer))
        mainCfg.Fragment("output", producer)
    } else if steps.Consumer != nil && steps.Sink != nil {
        consumer, err := compileConsumer(*steps.Consumer, steps.Transformer)
        if err != nil {
            return "", fmt.Errorf("source: %w", err)
        }

        mainCfg.Fragment("input", consumer)
        mainCfg.Fragment("output", compileSink(*steps.Sink))
    } else {
        return "", fmt.Errorf("invalid steps")
    }

    b, err := yaml.Marshal(mainCfg)
    if err != nil {
        return "", fmt.Errorf("marshal: %w", err)
    }

    return string(b), nil

}
