package nats

import (
    "bytes"
    "context"
    "fmt"
    "github.com/nats-io/nats.go"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/collectors"
    "github.com/prometheus/common/expfmt"
    "github.com/prometheus/common/model"
    "github.com/redpanda-data/benthos/v4/public/service"
    "github.com/synadia-io/connect-runtime-wombat/components/nats/stats"
    "sync"
    "time"
)

const (
    metricUrlField           = "url"
    metricSubjectField       = "subject"
    metricFlushIntervalField = "flush_interval"
    metricJwtField           = "jwt"
    metricSeedField          = "seed"
)

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
    )

func NewMetrics(conf *service.ParsedConfig, log *service.Logger) (m *Metrics, err error) {
    url, err := conf.FieldString(metricUrlField)
    if err != nil {
        return nil, fmt.Errorf("failed to get nats url field: %w", err)
    }

    opts := []nats.Option{
        nats.Name("MetricsOps"),
    }

    if conf.Contains(metricJwtField) && conf.Contains(metricSeedField) {
        jwt, _ := conf.FieldString(metricJwtField)
        seed, _ := conf.FieldString(metricSeedField)
        if jwt != "" && seed != "" {
            opts = append(opts, nats.UserJWTAndSeed(jwt, seed))
        }
    }

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

                    if err = m.nc.Publish(m.subject, b.Bytes()); err != nil {
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
