package test

import (
	"github.com/synadia-labs/vent/public/control"
)

func Inlet(source control.Source, producer control.Producer) control.Steps {
	return InletWithTransformer(source, nil, producer)
}

func InletWithTransformer(source control.Source, transformer *control.Transformer, producer control.Producer) control.Steps {
	return control.Steps{
		Source:      &source,
		Transformer: transformer,
		Producer:    &producer,
	}
}
