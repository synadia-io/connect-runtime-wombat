package compiler

import (
	"os"

	"github.com/synadia-io/connect/model"
)

// natsCredsFileEnvVar is the environment variable that, when set to a non-empty
// path, points to a NATS user credentials file (user JWT + NKey seed) provisioned
// by the runtime (e.g. the nex agent). When present, the data-plane nats
// components are configured with auth.user_credentials_file instead of inlining a
// static user_jwt/user_nkey_seed pair.
//
// Why this matters for resilience: the Benthos/Wombat nats components read the
// creds file on every (re)connect and signature challenge (via the nats.UserJWT
// handlers, which re-read the file each time), so a rotated/refreshed credential
// on disk is picked up automatically. Inline user_jwt/user_nkey_seed values are
// captured once at connect time (nats.UserJWTAndSeed), so an expired/rotated JWT
// permanently breaks the data-plane connection after repeated auth errors.
//
// Limitations: the Benthos nats components in this repo's version expose only
// urls/tls/auth config fields and do not expose any reconnect/no-give-up knobs,
// so those cannot be tuned via the generated config; nats.go defaults apply. The
// creds file therefore needs to be refreshed within nats.go's default reconnect
// window for automatic recovery. This swap only takes effect when auth is enabled
// on the connector's NatsConfig; when the env var is unset the previous inline
// behaviour is preserved exactly (backward compatible).
const natsCredsFileEnvVar = "NEX_WORKLOAD_NATS_CREDS_FILE"

func natsBaseFragment(c model.NatsConfig) Fragment {
	cfg := Frag().
		Strings("urls", c.Url)

	if c.AuthEnabled {
		auth := Frag()
		if credsFile := os.Getenv(natsCredsFileEnvVar); credsFile != "" {
			// Reloadable credentials: Benthos re-reads this file on reconnect,
			// so a rotated JWT is picked up without losing the data plane.
			auth.String("user_credentials_file", credsFile)
		} else {
			// Backward-compatible fallback: inline static credentials.
			auth.StringP("user_jwt", c.Jwt).
				StringP("user_nkey_seed", c.Seed)
		}
		cfg.Fragment("auth", auth)
	}

	return cfg
}
