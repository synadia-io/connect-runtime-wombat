package compiler

import (
	"testing"

	"github.com/synadia-io/connect-runtime-wombat/utils"
	"github.com/synadia-io/connect/model"
)

// authNatsConfig returns a NatsConfig with inline JWT/seed authentication enabled,
// mirroring what the Connect platform provides today.
func authNatsConfig() model.NatsConfig {
	return model.NatsConfig{
		Url:         "nats://localhost:4222",
		AuthEnabled: true,
		Jwt:         utils.Ptr("test-jwt"),
		Seed:        utils.Ptr("test-seed"),
	}
}

// When NEX_WORKLOAD_NATS_CREDS_FILE is unset, the generated config must keep the
// current inline user_jwt/user_nkey_seed form (backward compatible).
func TestNatsBaseFragmentUsesInlineAuthWhenCredsFileEnvUnset(t *testing.T) {
	// Present-but-empty is treated as unset.
	t.Setenv("NEX_WORKLOAD_NATS_CREDS_FILE", "")

	frag := natsBaseFragment(authNatsConfig())

	exp := map[string]any{
		"urls": []string{"nats://localhost:4222"},
		"auth": map[string]any{
			"user_jwt":       "test-jwt",
			"user_nkey_seed": "test-seed",
		},
	}
	if !frag.EqualsMap(exp) {
		t.Errorf("expected %v, got %v", exp, frag)
	}
}

// When NEX_WORKLOAD_NATS_CREDS_FILE is set, the generated config must use
// auth.user_credentials_file and must NOT inline user_jwt/user_nkey_seed, so that
// the Benthos nats components re-read the (rotatable) creds file on reconnect.
func TestNatsBaseFragmentUsesCredentialsFileWhenCredsFileEnvSet(t *testing.T) {
	const credsFile = "/var/run/nats/user.creds"
	t.Setenv("NEX_WORKLOAD_NATS_CREDS_FILE", credsFile)

	frag := natsBaseFragment(authNatsConfig())

	exp := map[string]any{
		"urls": []string{"nats://localhost:4222"},
		"auth": map[string]any{
			"user_credentials_file": credsFile,
		},
	}
	if !frag.EqualsMap(exp) {
		t.Errorf("expected %v, got %v", exp, frag)
	}

	// Explicitly assert the static inline credentials are absent.
	auth, ok := frag["auth"].(Fragment)
	if !ok {
		t.Fatalf("expected auth fragment, got %T", frag["auth"])
	}
	if v, ok := auth["user_jwt"]; ok {
		t.Errorf("expected no inline user_jwt when creds file is set, got %v", v)
	}
	if v, ok := auth["user_nkey_seed"]; ok {
		t.Errorf("expected no inline user_nkey_seed when creds file is set, got %v", v)
	}
}
