package main

import "gopkg.in/yaml.v3"

type FormatSpec struct {
    Elements []FormatSpecElement
}

type FormatSpecElement struct {
    Name     string
    Children []FormatSpecElement
}

type Formatter struct {
    spec FormatSpec
}

func (f *Formatter) Format(yamlString string) (string, error) {
    var src yaml.Node
    if err := yaml.Unmarshal([]byte(yamlString), &src); err != nil {
        return "", err
    }

    panic("not implemented")
}

func clean(elements []FormatSpecElement, nodes []*yaml.Node) []*yaml.Node {
    result := make([]*yaml.Node, len(nodes))

    for idx, element := range elements {
        for _, n := range nodes {
            if n.Tag == element.Name {
                result[idx] = n
                break
            }
        }

        panic("not implemented")
    }

    return result
}
