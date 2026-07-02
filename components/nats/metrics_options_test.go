package nats

import (
	"os"
	"testing"

	"github.com/nats-io/nats.go"
)

// When NEX_WORKLOAD_NATS_CREDS_FILE is set, the metrics connection must
// authenticate from the reloadable creds file and must NOT fall back to the
// inline jwt/seed, so a rotated JWT is picked up on reconnect.
func TestMetricsAuthHonorsCredsFileEnvVar(t *testing.T) {
	const credsFile = "/var/run/nats/user.creds"
	t.Setenv(natsCredsFileEnvVar, credsFile)

	got := resolveMetricsAuth(os.Getenv(natsCredsFileEnvVar), "inline-jwt", "inline-seed")

	if got.CredsFile != credsFile {
		t.Errorf("expected creds file %q from env, got %+v", credsFile, got)
	}
	if got.Jwt != "" || got.Seed != "" {
		t.Errorf("expected inline auth suppressed when creds file set, got %+v", got)
	}
}

// When NEX_WORKLOAD_NATS_CREDS_FILE is unset (present-but-empty counts as unset),
// the metrics connection keeps the previous inline jwt/seed behaviour.
func TestMetricsAuthUsesInlineWhenEnvUnset(t *testing.T) {
	t.Setenv(natsCredsFileEnvVar, "")

	got := resolveMetricsAuth(os.Getenv(natsCredsFileEnvVar), "inline-jwt", "inline-seed")

	want := metricsAuthConfig{Jwt: "inline-jwt", Seed: "inline-seed"}
	if got != want {
		t.Errorf("expected inline auth %+v when env unset, got %+v", want, got)
	}
}

// Inline auth requires both jwt and seed; a partial pair yields no auth, exactly
// as the original guard behaved.
func TestResolveMetricsAuthIgnoresIncompleteInline(t *testing.T) {
	if got := resolveMetricsAuth("", "inline-jwt", ""); got != (metricsAuthConfig{}) {
		t.Errorf("expected no auth when seed missing, got %+v", got)
	}
	if got := resolveMetricsAuth("", "", "inline-seed"); got != (metricsAuthConfig{}) {
		t.Errorf("expected no auth when jwt missing, got %+v", got)
	}
}

// A creds file must produce an auth option appended after the three base options.
// The option is not applied here because nats.UserCredentials reads the file at
// apply time; selection is verified at the resolver level instead.
func TestBuildMetricsConnOptionsAppendsCredsFileAuth(t *testing.T) {
	if resolveMetricsAuth("/var/run/nats/user.creds", "", "").option() == nil {
		t.Fatal("expected a nats auth option for a creds file")
	}
	if got := len(buildMetricsConnOptions("/var/run/nats/user.creds", "", "")); got != 4 {
		t.Errorf("expected 4 options (name, ignore-auth-abort, max-reconnects, auth), got %d", got)
	}
}

// The resilience options must always be applied, independent of the auth source,
// and inline jwt/seed must configure authentication. These cases use no-I/O
// options so they can be applied to a nats.Options for inspection.
func TestBuildMetricsConnOptionsAppliesResilience(t *testing.T) {
	cases := map[string]struct {
		jwt, seed string
		wantAuth  bool
	}{
		"no auth":             {wantAuth: false},
		"inline jwt and seed": {jwt: "j", seed: "s", wantAuth: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var o nats.Options
			for _, opt := range buildMetricsConnOptions("", tc.jwt, tc.seed) {
				if err := opt(&o); err != nil {
					t.Fatalf("applying option failed: %v", err)
				}
			}

			if !o.IgnoreAuthErrorAbort {
				t.Error("expected IgnoreAuthErrorAbort to be set")
			}
			if o.MaxReconnect != -1 {
				t.Errorf("expected MaxReconnect -1, got %d", o.MaxReconnect)
			}
			if o.Name != "MetricsOps" {
				t.Errorf("expected connection name MetricsOps, got %q", o.Name)
			}
			if gotAuth := o.UserJWT != nil; gotAuth != tc.wantAuth {
				t.Errorf("expected auth configured=%v, got %v", tc.wantAuth, gotAuth)
			}
		})
	}
}
