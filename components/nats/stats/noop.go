package stats

type NoopStat struct{}

func (n NoopStat) Incr(count int64)          {}
func (n NoopStat) Decr(count int64)          {}
func (n NoopStat) Timing(delta int64)        {}
func (n NoopStat) Set(value int64)           {}
func (n NoopStat) SetFloat64(value float64)  {}
func (n NoopStat) IncrFloat64(count float64) {}
func (n NoopStat) DecrFloat64(count float64) {}
