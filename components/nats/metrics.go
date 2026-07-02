// Package nats provides custom NATS components for enhanced integration with Wombat.
// The primary component is a metrics exporter that publishes Prometheus-formatted
// metrics to NATS subjects.
package nats

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/synadia-io/connect-runtime-wombat/components/nats/stats"
)

const (
	metricUrlField           = "url"
	metricSubjectField       = "subject"
	metricFlushIntervalField = "flush_interval"
	metricJwtField           = "jwt"
	metricSeedField          = "seed"
	headersField             = "headers"
)

// MetricsConfigSpec defines the configuration schema for the NATS metrics exporter.
// It specifies fields for NATS connection, authentication, and publishing settings.
var MetricsConfigSpec = service.NewConfigSpec().
	Beta().
	Summary("publish metrics in prometheus format onto a NATS subject").
	Fields(
		service.NewStringField(metricSubjectField).
			Description("The subject to publish metrics.").
			Optional().
			Default("wombat.metrics"),
		service.NewDurationField(metricFlushIntervalField).
			Description("The interval for flushing metrics to the subject").
			Optional().
			Default("5s"),
		service.NewStringField(metricUrlField).Description("The url of the NATS server"),
		service.NewStringField(metricJwtField).Description("The JWT for the NATS server").Optional(),
		service.NewStringField(metricSeedField).Description("The seed for the NATS server").Optional(),
		service.NewStringMapField(headersField).Description("A list of headers to add to the NATS server").Optional(),
	)

// natsCredsFileEnvVar mirrors compiler.natsCredsFileEnvVar. It is duplicated here
// rather than imported because the components package cannot depend on the
// compiler package without an import cycle (the compiler imports components).
//
// When set to a non-empty path it points at a reloadable NATS user credentials
// file (user JWT + NKey seed) provisioned by the runtime (nex agent). The metrics
// connection then authenticates via nats.UserCredentials, whose callbacks re-read
// the file on every (re)connect and signature challenge, so a rotated/refreshed
// JWT is picked up automatically. When unset, the inline jwt/seed captured at
// compile time is used (backward compatible). Compile and stream run in the same
// process/container, so the env var is available at connection time.
const natsCredsFileEnvVar = "NEX_WORKLOAD_NATS_CREDS_FILE"

// metricsAuthConfig is the resolved NATS authentication source for the metrics
// connection. At most one of CredsFile or (Jwt, Seed) is populated.
type metricsAuthConfig struct {
	CredsFile string
	Jwt       string
	Seed      string
}

// resolveMetricsAuth selects the metrics NATS auth source. A non-empty reloadable
// creds file takes precedence over the inline jwt/seed; inline auth is only used
// when both jwt and seed are present (matching the previous behaviour).
func resolveMetricsAuth(credsFile, jwt, seed string) metricsAuthConfig {
	if credsFile != "" {
		return metricsAuthConfig{CredsFile: credsFile}
	}
	if jwt != "" && seed != "" {
		return metricsAuthConfig{Jwt: jwt, Seed: seed}
	}
	return metricsAuthConfig{}
}

// option returns the nats.Option for the resolved auth source, or nil when no
// authentication is configured.
func (a metricsAuthConfig) option() nats.Option {
	switch {
	case a.CredsFile != "":
		return nats.UserCredentials(a.CredsFile)
	case a.Jwt != "" && a.Seed != "":
		return nats.UserJWTAndSeed(a.Jwt, a.Seed)
	default:
		return nil
	}
}

// buildMetricsConnOptions assembles the nats connection options for the metrics
// exporter. The resilience options are always applied so that a rotated or
// expired JWT cannot permanently kill the metrics connection: without them,
// repeated auth errors make nats.go abort the connection for good, leaving the
// connector moving data while silently no longer publishing metrics.
// IgnoreAuthErrorAbort keeps retrying through auth errors and MaxReconnects(-1)
// never stops trying to reconnect.
func buildMetricsConnOptions(credsFile, jwt, seed string) []nats.Option {
	opts := []nats.Option{
		nats.Name("MetricsOps"),
		nats.IgnoreAuthErrorAbort(),
		nats.MaxReconnects(-1),
	}
	if opt := resolveMetricsAuth(credsFile, jwt, seed).option(); opt != nil {
		opts = append(opts, opt)
	}
	return opts
}

// NewMetrics creates a new NATS metrics exporter from the provided configuration.
// It establishes a connection to NATS and sets up periodic publishing of metrics.
//
// Parameters:
//   - conf: Parsed configuration containing NATS connection details
//   - log: Logger for error reporting
//
// Returns:
//   - A configured Metrics instance
//   - An error if configuration is invalid or connection fails
func NewMetrics(conf *service.ParsedConfig, log *service.Logger) (m *Metrics, err error) {
	url, err := conf.FieldString(metricUrlField)
	if err != nil {
		return nil, fmt.Errorf("failed to get nats url field: %w", err)
	}

	var jwt, seed string
	if conf.Contains(metricJwtField) {
		jwt, _ = conf.FieldString(metricJwtField)
	}
	if conf.Contains(metricSeedField) {
		seed, _ = conf.FieldString(metricSeedField)
	}

	opts := buildMetricsConnOptions(os.Getenv(natsCredsFileEnvVar), jwt, seed)

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	m = &Metrics{
		nc:  nc,
		log: log,
		reg: prometheus.NewRegistry(),

		counters:   make(map[string]*stats.CounterVec),
		gauges:     make(map[string]*stats.GaugeVec),
		timers:     make(map[string]*stats.TimingVec),
		timersHist: make(map[string]*stats.TimingHistVec),

		closedChan: make(chan struct{}),
	}

	if headers, _ := conf.FieldStringMap(headersField); headers != nil {
		m.headers = headers
	}

	if err := m.reg.Register(collectors.NewBuildInfoCollector()); err != nil {
		return nil, fmt.Errorf("failed to register build info collector: %w", err)
	}

	if err := m.reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, fmt.Errorf("failed to register process info collector: %w", err)
	}

	m.subject, err = conf.FieldString(metricSubjectField)
	if err != nil {
		return nil, fmt.Errorf("failed to get subject field: %w", err)
	}

	flushInterval, _ := conf.FieldString(metricFlushIntervalField)
	if flushInterval != "" {
		interval, err := time.ParseDuration(flushInterval)
		if err != nil {
			return nil, fmt.Errorf("failed to parse flush interval: %v", err)
		}
		go func() {
			for {
				select {
				case <-m.closedChan:
					return
				case <-time.After(interval):
					mfs, err := m.reg.Gather()
					if err != nil {
						m.log.Errorf("Failed to gather metrics: %v\n", err)
						continue
					}

					b := bytes.NewBuffer(make([]byte, 0))
					for idx, mf := range mfs {
						if idx > 0 {
							b.WriteString("\n")
						}

						if _, err = expfmt.MetricFamilyToText(b, mf); err != nil {
							m.log.Errorf("Failed to convert metrics to text: %v\n", err)
							continue
						}
					}

					msg := nats.NewMsg(m.subject)
					msg.Header.Set("format", "expfmt")

					for k, v := range m.headers {
						msg.Header.Set(k, v)
					}

					msg.Data = b.Bytes()

					if err = m.nc.PublishMsg(msg); err != nil {
						m.log.Errorf("Failed to publish metrics: %v\n", err)
					}
				}
			}
		}()
	}

	return m, nil
}

type Metrics struct {
	log *service.Logger
	mut sync.Mutex

	reg *prometheus.Registry

	subject string
	nc      *nats.Conn

	counters   map[string]*stats.CounterVec
	gauges     map[string]*stats.GaugeVec
	timers     map[string]*stats.TimingVec
	timersHist map[string]*stats.TimingHistVec

	headers map[string]string

	closedChan chan struct{}
}

func (m *Metrics) NewCounterCtor(path string, labelNames ...string) service.MetricsExporterCounterCtor {
	if !model.IsValidMetricName(model.LabelValue(path)) {
		m.log.Errorf("Ignoring metric '%v' due to invalid name", path)
		return func(labelValues ...string) service.MetricsExporterCounter {
			return stats.NoopStat{}
		}
	}

	var pv *stats.CounterVec

	m.mut.Lock()
	var exists bool
	if pv, exists = m.counters[path]; !exists {
		pv = stats.NewCounterVec(m.reg, path, labelNames)
		m.counters[path] = pv
	}
	m.mut.Unlock()

	if pv.LabelCount() != len(labelNames) {
		m.log.Errorf("Metrics label mismatch %v versus %v %v for name '%v', skipping metric", pv.LabelCount(), len(labelNames), labelNames, path)
		return func(labelValues ...string) service.MetricsExporterCounter {
			return stats.NoopStat{}
		}
	}
	return func(labelValues ...string) service.MetricsExporterCounter {
		return pv.With(labelValues...)
	}
}

func (m *Metrics) NewTimerCtor(path string, labelNames ...string) service.MetricsExporterTimerCtor {
	if !model.IsValidMetricName(model.LabelValue(path)) {
		m.log.Errorf("Ignoring metric '%v' due to invalid name", path)
		return func(labelValues ...string) service.MetricsExporterTimer {
			return stats.NoopStat{}
		}
	}

	var pv *stats.TimingVec

	m.mut.Lock()
	var exists bool
	if pv, exists = m.timers[path]; !exists {
		pv = stats.NewTimingVec(m.reg, path, labelNames)
		m.timers[path] = pv
	}
	m.mut.Unlock()

	if pv.LabelCount() != len(labelNames) {
		m.log.Errorf("Metrics label mismatch %v versus %v %v for name '%v', skipping metric", pv.LabelCount(), len(labelNames), labelNames, path)
		return func(labelValues ...string) service.MetricsExporterTimer {
			return stats.NoopStat{}
		}
	}
	return func(labelValues ...string) service.MetricsExporterTimer {
		return pv.With(labelValues...)
	}
}

func (m *Metrics) NewGaugeCtor(path string, labelNames ...string) service.MetricsExporterGaugeCtor {
	if !model.IsValidMetricName(model.LabelValue(path)) {
		m.log.Errorf("Ignoring metric '%v' due to invalid name", path)
		return func(labelValues ...string) service.MetricsExporterGauge {
			return &stats.NoopStat{}
		}
	}

	var pv *stats.GaugeVec

	m.mut.Lock()
	var exists bool
	if pv, exists = m.gauges[path]; !exists {
		pv = stats.NewGaugeVec(m.reg, path, labelNames)
		m.gauges[path] = pv
	}
	m.mut.Unlock()

	if pv.LabelCount() != len(labelNames) {
		m.log.Errorf("Metrics label mismatch %v versus %v %v for name '%v', skipping metric", pv.LabelCount(), len(labelNames), labelNames, path)
		return func(labelValues ...string) service.MetricsExporterGauge {
			return stats.NoopStat{}
		}
	}
	return func(labelValues ...string) service.MetricsExporterGauge {
		return pv.With(labelValues...)
	}
}

func (m *Metrics) Close(ctx context.Context) error {
	close(m.closedChan)

	if m.nc != nil {
		m.nc.Close()
	}

	return nil
}
