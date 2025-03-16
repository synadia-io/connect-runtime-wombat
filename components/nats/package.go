package nats

import "github.com/redpanda-data/benthos/v4/public/service"

func init() {
    err := service.RegisterMetricsExporter(
        "nats", MetricsConfigSpec,
        func(conf *service.ParsedConfig, log *service.Logger) (service.MetricsExporter, error) {
            return NewMetrics(conf, log)
        })
    if err != nil {
        panic(err)
    }
}
