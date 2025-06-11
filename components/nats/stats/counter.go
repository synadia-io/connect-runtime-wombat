package stats

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redpanda-data/benthos/v4/public/service"
)

type Counter struct {
	ctr prometheus.Counter
}

func (p *Counter) Incr(count int64) {
	p.ctr.Add(float64(count))
}

func (p *Counter) IncrFloat64(count float64) {
	p.ctr.Add(count)
}

func NewCounterVec(reg *prometheus.Registry, path string, labelNames []string) *CounterVec {
	ctr := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "connector",
		Name:      path,
		Help:      "Connector Counter metric",
	}, labelNames)
	reg.MustRegister(ctr)

	return &CounterVec{
		ctr:   ctr,
		count: len(labelNames),
	}
}

type CounterVec struct {
	ctr   *prometheus.CounterVec
	count int
}

func (p *CounterVec) LabelCount() int {
	return p.count
}

func (p *CounterVec) With(labelValues ...string) service.MetricsExporterCounter {
	return &Counter{
		ctr: p.ctr.WithLabelValues(labelValues...),
	}
}
