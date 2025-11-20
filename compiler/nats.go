package compiler

import (
	"github.com/synadia-io/connect/v2/model"
)

func natsBaseFragment(c model.NatsConfig) Fragment {
	cfg := Frag().
		Strings("urls", c.Url)

	if c.AuthEnabled {
		cfg.Fragment("auth", Frag().
			StringP("user_jwt", c.Jwt).
			StringP("user_nkey_seed", c.Seed))
	}

	return cfg
}
