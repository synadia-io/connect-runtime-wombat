package stats

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redpanda-data/benthos/v4/public/service"
)

type Timing struct {
	sum       prometheus.Observer
	asSeconds bool
}

func (p *Timing) Timing(val int64) {
	vFloat := float64(val)
	if p.asSeconds {
		vFloat /= 1_000_000_000
	}
	p.sum.Observe(vFloat)
}

func NewTimingVec(reg *prometheus.Registry, path string, labelNames []string) *TimingVec {
	tmr := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "connector",
		Name:      path,
		Help:      "Connector Timing metric",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	}, labelNames)
	reg.MustRegister(tmr)

	return &TimingVec{
		sum:   tmr,
		count: len(labelNames),
	}
}

type TimingVec struct {
	sum   *prometheus.SummaryVec
	count int
}

func (p *TimingVec) LabelCount() int {
	return p.count
}

func (p *TimingVec) With(labelValues ...string) service.MetricsExporterTimer {
	return &Timing{
		sum: p.sum.WithLabelValues(labelValues...),
	}
}

type TimingHistVec struct {
	sum   *prometheus.HistogramVec
	count int
}

func (p *TimingHistVec) LabelCount() int {
	return p.count
}

func (p *TimingHistVec) With(labelValues ...string) service.MetricsExporterTimer {
	return &Timing{
		asSeconds: true,
		sum:       p.sum.WithLabelValues(labelValues...),
	}
}
