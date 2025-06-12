package main

import (
	"gopkg.in/yaml.v3"
)

type FormatSpecElement struct {
	Name     string
	Children []FormatSpecElement
}

type FormatSpec struct {
	Children []FormatSpecElement
}

type Formatter struct {
	// This struct has no functional implementation yet
}

func (f *Formatter) Format(yamlString string) (string, error) {
	var src yaml.Node
	if err := yaml.Unmarshal([]byte(yamlString), &src); err != nil {
		return "", err
	}

	panic("not implemented")
}
