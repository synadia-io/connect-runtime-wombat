package test

import (
	"github.com/synadia-io/connect/model"
)

func Inlet(source model.Source, producer model.Producer) model.Steps {
	return InletWithTransformer(source, nil, producer)
}

func InletWithTransformer(source model.Source, transformer *model.Transformer, producer model.Producer) model.Steps {
	return model.Steps{
		Source:      &source,
		Transformer: transformer,
		Producer:    &producer,
	}
}
