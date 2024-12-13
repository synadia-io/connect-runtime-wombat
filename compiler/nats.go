package compiler

import (
	"github.com/synadia-io/connect/model"
)

func attachNatsConfig(target map[string]any, c model.NatsConfig) {
	target["urls"] = []string{c.Url}

	if c.AuthEnabled {
		target["auth"] = map[string]string{
			"user_jwt":       c.Jwt,
			"user_nkey_seed": c.Seed,
		}
	}
}
