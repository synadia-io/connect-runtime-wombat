package corrections

import (
    "fmt"
    "github.com/Jeffail/gabs/v2"
)

type Correction struct {
    Replace      *Replace      `yaml:"replace,omitempty"`
    AsciidocToMd *AsciidocToMd `yaml:"asciidoc_to_md,omitempty"`
}

func (c Correction) Correct(doc *gabs.Container) (*gabs.Container, error) {
    if c.Replace != nil {
        return c.Replace.Correct(doc)
    } else if c.AsciidocToMd != nil {
        return c.AsciidocToMd.Correct(doc)
    } else {
        return nil, fmt.Errorf("invalid correction")
    }
}
