package integration_test

// This suite answers, end-to-end, the open question behind the creds-file fix
// (commit 7119fe6): does a compiled Wombat/Benthos stream that authenticates via
// auth.user_credentials_file actually SURVIVE a NATS credential rotation that
// happens mid-run, when the original user JWT expires?
//
// The path under test is the real one:
//   - a real nats-server is started in OPERATOR (JWT auth) mode;
//   - an auth-enabled connector is compiled through the REAL compiler
//     (compiler.Compile) with NEX_WORKLOAD_NATS_CREDS_FILE set, so
//     natsBaseFragment emits auth.user_credentials_file (verified below);
//   - the compiled YAML is executed with service.NewStreamBuilder().Run(), so the
//     Benthos nats output connects with nats.UserJWT handlers that re-read the
//     creds file on every (re)connect and signature challenge
//     (redpanda-connect internal/impl/nats/auth.go).
//
// Empirical finding (see the two specs): the stream SURVIVES rotation. When a
// fresh, valid creds file is written before the short-lived JWT expires, the
// server disconnects the connection at expiry, nats.go auto-reconnects, re-reads
// the rotated file, and data flow resumes on its own. The negative control (no
// rotation) shows the contrast: after the JWT expires the connection is torn
// down, reconnect attempts fail against the still-expired creds, and data flow
// does NOT resume. That contrast is what proves the rotation (not something else)
// is what heals the data plane.

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redpanda-data/benthos/v4/public/service"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"

	"github.com/synadia-io/connect-runtime-wombat/compiler"
	"github.com/synadia-io/connect-runtime-wombat/test"
	"github.com/synadia-io/connect/builders"

	// Import all components so the nats output is registered.
	_ "github.com/synadia-io/connect-runtime-wombat/components"
)

const (
	// credsEnvVar is the runtime-provisioned reloadable creds-file pointer that
	// the compiler consults to decide between auth.user_credentials_file and the
	// legacy inline user_jwt/user_nkey_seed. Mirrors compiler.natsCredsFileEnvVar.
	credsEnvVar = "NEX_WORKLOAD_NATS_CREDS_FILE"

	// rotationSubject is the core-NATS subject the compiled inlet publishes to.
	rotationSubject = "rotation.data"

	// initialCredTTL is the lifetime of the FIRST (short-lived) user JWT. It must
	// be long enough for baseline flow to establish, short enough to keep the test
	// fast. The operator server enforces expiry at 1s granularity, so keep it a
	// few whole seconds.
	initialCredTTL = 6 * time.Second
)

var _ = Describe("NATS creds-file rotation (mid-run JWT expiry)", func() {
	var (
		srv        *server.Server
		accKP      nkeys.KeyPair
		dataUserKP nkeys.KeyPair // stable identity across rotation (only the JWT is re-minted)
		credsPath  string
		verifierNC *nats.Conn
		received   *atomic.Int64
	)

	BeforeEach(func() {
		// -- Operator + account, entirely in-memory.
		opKP, err := nkeys.CreateOperator()
		Expect(err).NotTo(HaveOccurred())
		opPub, err := opKP.PublicKey()
		Expect(err).NotTo(HaveOccurred())

		accKP, err = nkeys.CreateAccount()
		Expect(err).NotTo(HaveOccurred())
		accPub, err := accKP.PublicKey()
		Expect(err).NotTo(HaveOccurred())

		nac := jwt.NewAccountClaims(accPub)
		nac.Name = "ROTATION_ACC"
		accJWT, err := nac.Encode(opKP)
		Expect(err).NotTo(HaveOccurred())

		// -- Auth-enabled server: trust the operator key, resolve the account JWT
		// from an in-memory resolver.
		opts := &server.Options{
			Host:                  "127.0.0.1",
			Port:                  -1,
			NoLog:                 true,
			NoSigs:                true,
			MaxControlLine:        4096,
			DisableShortFirstPing: true,
			TrustedKeys:           []string{opPub},
			AccountResolver:       &server.MemAccResolver{},
		}
		srv, err = server.NewServer(opts)
		Expect(err).NotTo(HaveOccurred())
		go srv.Start()
		Expect(srv.ReadyForConnections(10 * time.Second)).To(BeTrue())
		Expect(srv.AccountResolver().Store(accPub, accJWT)).To(Succeed())
		DeferCleanup(srv.Shutdown)

		// -- The rotating data-plane identity used by the compiled connector.
		dataUserKP, err = nkeys.CreateUser()
		Expect(err).NotTo(HaveOccurred())

		// -- An independent, long-lived verifier connection that counts what the
		// data plane actually delivers. It authenticates in-memory (no file), so
		// it is immune to the rotation under test.
		verifierUserKP, err := nkeys.CreateUser()
		Expect(err).NotTo(HaveOccurred())
		verifierNC, err = nats.Connect(srv.ClientURL(),
			inMemoryUserOption(accKP, verifierUserKP, time.Hour))
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(verifierNC.Close)

		received = &atomic.Int64{}
		_, err = verifierNC.Subscribe(rotationSubject, func(*nats.Msg) {
			received.Add(1)
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(verifierNC.Flush()).To(Succeed())

		// -- Creds file + env var must be in place BEFORE compilation, because the
		// compiler reads the env var at compile time to choose the creds-file seam.
		credsPath = filepath.Join(GinkgoT().TempDir(), "user.creds")
		Expect(os.Setenv(credsEnvVar, credsPath)).To(Succeed())
		DeferCleanup(func() { _ = os.Unsetenv(credsEnvVar) })
	})

	It("resumes data flow after the JWT expires when the creds file is rotated", func() {
		// Write the short-lived credentials the connector will start with.
		mintTime := time.Now()
		writeCredsAtomic(credsPath, mintUserCreds(accKP, dataUserKP, initialCredTTL))

		stream := buildRotationStream(srv.ClientURL())
		runStream(stream)

		// Baseline: the data plane connects with the short-lived creds and flows.
		Eventually(received.Load, 4*time.Second, 50*time.Millisecond).
			Should(BeNumerically(">=", int64(3)), "expected baseline data flow with the initial creds")

		// Rotate: same identity, fresh long-lived JWT, written atomically to the
		// same path while the stream is running and BEFORE the initial JWT expires.
		writeCredsAtomic(credsPath, mintUserCreds(accKP, dataUserKP, time.Hour))

		// Let the initial JWT expire (~initialCredTTL) and the client auto-reconnect
		// (nats.go default reconnect wait is 2s). By now any pre-expiry buffered
		// messages have flushed, so growth past this point is genuinely post-reconnect.
		time.Sleep(time.Until(mintTime.Add(10 * time.Second)))
		afterReconnect := received.Load()

		// THE ASSERTION: the connection healed itself off the rotated creds file and
		// fresh messages keep arriving.
		Eventually(received.Load, 20*time.Second, 200*time.Millisecond).
			Should(BeNumerically(">", afterReconnect+2),
				"data flow did not resume after credential rotation")
	})

	It("does NOT resume data flow after the JWT expires when creds are not rotated (negative control)", func() {
		// Same short-lived creds, but this time we never refresh them.
		mintTime := time.Now()
		writeCredsAtomic(credsPath, mintUserCreds(accKP, dataUserKP, initialCredTTL))

		stream := buildRotationStream(srv.ClientURL())
		runStream(stream)

		// Baseline flow proves the initial connect worked.
		Eventually(received.Load, 4*time.Second, 50*time.Millisecond).
			Should(BeNumerically(">=", int64(3)), "expected baseline data flow with the initial creds")

		// Wait past expiry (~initialCredTTL) and the two auth-failing reconnect
		// attempts after which nats.go gives up (same auth error twice). The stale
		// creds can never satisfy the server, so nothing should be delivered.
		time.Sleep(time.Until(mintTime.Add(13 * time.Second)))
		plateau := received.Load()

		Consistently(received.Load, 3*time.Second, 250*time.Millisecond).
			Should(BeNumerically("<=", plateau),
				"data flow unexpectedly resumed without a credential rotation")
	})
})

// buildRotationStream compiles an auth-enabled generate -> core-NATS inlet through
// the real compiler and builds a runnable Benthos stream from the result. It also
// asserts that the fix's creds-file seam is actually exercised.
func buildRotationStream(url string) *service.Stream {
	// Auth(...) flips AuthEnabled so natsBaseFragment takes the auth branch; the
	// inline jwt/seed values are deliberately bogus because, with the env var set,
	// the compiler must ignore them in favour of user_credentials_file.
	natsCfg := builders.NatsConfig().Url(url).Auth("unused-inline-jwt", "unused-inline-seed")

	steps := builders.Steps().
		Source(builders.SourceStep("generate").
			SetString("mapping", "root.seq = counter()").
			SetString("interval", "50ms").
			SetInt("count", 0)).
		Producer(builders.ProducerStep(natsCfg).Core(builders.ProducerStepCore(rotationSubject))).
		Build()

	yamlCfg, err := compiler.Compile(test.Runtime(), steps)
	Expect(err).NotTo(HaveOccurred())
	Expect(yamlCfg).To(ContainSubstring("user_credentials_file"),
		"expected the compiler to emit the reloadable creds-file auth seam")
	Expect(yamlCfg).NotTo(ContainSubstring("user_jwt"),
		"expected inline jwt auth to be suppressed when the creds file is set")

	builder := service.NewStreamBuilder()
	Expect(builder.SetYAML(yamlCfg)).To(Succeed())
	stream, err := builder.Build()
	Expect(err).NotTo(HaveOccurred())
	return stream
}

// runStream runs the stream in the background and, at spec teardown, cancels it
// and waits for it to exit before the temp creds file is removed (so a lagging
// reconnect attempt can't race the cleanup).
func runStream(stream *service.Stream) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(done)
		// Run returns nil on ctx cancellation; any other error is incidental to the
		// rotation behaviour being measured (the verifier counter is the oracle),
		// so we only surface it to the log.
		if err := stream.Run(ctx); err != nil && ctx.Err() == nil {
			GinkgoWriter.Printf("stream.Run returned: %v\n", err)
		}
	}()
	DeferCleanup(func() {
		cancel()
		Eventually(done, 10*time.Second).Should(BeClosed(), "stream did not shut down")
	})
}

// mintUserCreds encodes a decorated NATS creds file (JWT + NKey seed) for userKP,
// signed by the account key, expiring after ttl. Re-minting for the same userKP is
// exactly how a real credential rotation refreshes an identity's JWT.
func mintUserCreds(accKP, userKP nkeys.KeyPair, ttl time.Duration) []byte {
	GinkgoHelper()
	pub, err := userKP.PublicKey()
	Expect(err).NotTo(HaveOccurred())

	uc := jwt.NewUserClaims(pub)
	uc.Name = "rotation-user"
	if ttl > 0 {
		uc.Expires = time.Now().Add(ttl).Unix()
	}
	ujwt, err := uc.Encode(accKP)
	Expect(err).NotTo(HaveOccurred())

	seed, err := userKP.Seed()
	Expect(err).NotTo(HaveOccurred())

	creds, err := jwt.FormatUserConfig(ujwt, seed)
	Expect(err).NotTo(HaveOccurred())
	return creds
}

// inMemoryUserOption builds a nats.Option authenticating as userKP without touching
// disk, used by the verifier connection so it is unaffected by the file rotation.
func inMemoryUserOption(accKP, userKP nkeys.KeyPair, ttl time.Duration) nats.Option {
	GinkgoHelper()
	pub, err := userKP.PublicKey()
	Expect(err).NotTo(HaveOccurred())

	uc := jwt.NewUserClaims(pub)
	if ttl > 0 {
		uc.Expires = time.Now().Add(ttl).Unix()
	}
	ujwt, err := uc.Encode(accKP)
	Expect(err).NotTo(HaveOccurred())

	return nats.UserJWT(
		func() (string, error) { return ujwt, nil },
		func(nonce []byte) ([]byte, error) { return userKP.Sign(nonce) },
	)
}

// writeCredsAtomic writes creds via a temp file + rename so a concurrent reader
// (the nats.go auth handlers, which read the file on every reconnect) never sees a
// partially written file.
func writeCredsAtomic(path string, data []byte) {
	GinkgoHelper()
	tmp := path + ".tmp"
	Expect(os.WriteFile(tmp, data, 0o600)).To(Succeed())
	Expect(os.Rename(tmp, path)).To(Succeed())
}
