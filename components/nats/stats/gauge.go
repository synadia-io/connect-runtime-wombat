package stats

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/redpanda-data/benthos/v4/public/service"
)

type Gauge struct {
    ctr prometheus.Gauge
}

func (p *Gauge) Incr(count int64) {
    p.ctr.Add(float64(count))
}

func (p *Gauge) IncrFloat64(count float64) {
    p.ctr.Add(count)
}

func (p *Gauge) Decr(count int64) {
    p.ctr.Add(float64(-count))
}

func (p *Gauge) DecrFloat64(count float64) {
    p.ctr.Add(-count)
}

func (p *Gauge) Set(value int64) {
    p.ctr.Set(float64(value))
}

func (p *Gauge) SetFloat64(value float64) {
    p.ctr.Set(value)
}

func NewGaugeVec(reg *prometheus.Registry, path string, labelNames []string) *GaugeVec {
    ctr := prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Namespace: "connector",
        Name:      path,
        Help:      "Connector Gauge metric",
    }, labelNames)
    reg.MustRegister(ctr)

    return &GaugeVec{
        ctr:   ctr,
        count: len(labelNames),
    }
}

type GaugeVec struct {
    ctr   *prometheus.GaugeVec
    count int
}

func (p *GaugeVec) LabelCount() int {
    return p.count
}

func (p *GaugeVec) With(labelValues ...string) service.MetricsExporterGauge {
    return &Gauge{
        ctr: p.ctr.WithLabelValues(labelValues...),
    }
}
