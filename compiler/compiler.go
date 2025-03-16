package compiler

import (
    "fmt"
    _ "github.com/synadia-io/connect-runtime-wombat/components"
    "github.com/synadia-io/connect/model"
    "github.com/synadia-io/connect/runtime"
    "gopkg.in/yaml.v3"
)

func Compile(rt *runtime.Runtime, steps model.Steps) (string, error) {
    mainCfg := Frag()

    if rt.NatsUrl != "" && rt.Namespace != "" && rt.Instance != "" {
        natsCfg := Frag().
            String("url", rt.NatsUrl).
            String("subject", fmt.Sprintf("$NEX.logs.%s.%s.metrics", rt.Namespace, rt.Instance))

        if rt.NatsJwt != "" && rt.NatsSeed != "" {
            natsCfg.
                String("jwt", rt.NatsJwt).
                String("seed", rt.NatsSeed)
        }

        mainCfg.
            Fragment("metrics", Frag().
                Fragment("nats", natsCfg))
    }

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
