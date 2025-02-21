package compiler

import (
    "github.com/synadia-io/connect/model"
)

func attachNatsConfig(target map[string]any, c model.NatsConfig) {
    target["urls"] = []string{c.Url}

    if c.AuthEnabled {
        target["auth"] = map[string]string{}

        if c.Jwt != nil {
            target["user_jwt"] = *c.Jwt
        }

        if c.Seed != nil {
            target["user_nkey_seed"] = *c.Seed
        }
    }
}

func attachNatsAuth(target map[string]any, c model.NatsConfig) {
    if c.AuthEnabled {
        target["auth"] = map[string]string{}

        if c.Jwt != nil {
            target["user_jwt"] = *c.Jwt
        }

        if c.Seed != nil {
            target["user_nkey_seed"] = *c.Seed
        }
    }
}
