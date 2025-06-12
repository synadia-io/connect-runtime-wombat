package corrections

import (
	"github.com/Jeffail/gabs/v2"
)

type Replace struct {
	Path        string `json:"path"`
	Replacement string `json:"with"`
}

func (c Replace) Correct(container *gabs.Container) (*gabs.Container, error) {
	_, err := container.SetP(c.Replacement, c.Path)
	return container, err
}
