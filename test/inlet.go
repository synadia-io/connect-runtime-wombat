package test

import (
    "github.com/synadia-io/connect/model"
)

func Inlet(source model.SourceStep, producer model.ProducerStep) model.Steps {
    return InletWithTransformer(source, nil, producer)
}

func InletWithTransformer(source model.SourceStep, transformer *model.TransformerStep, producer model.ProducerStep) model.Steps {
    return model.Steps{
        Source:      &source,
        Transformer: transformer,
        Producer:    &producer,
    }
}
