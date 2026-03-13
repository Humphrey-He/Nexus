package api

import "time"

// Metrics records ORM performance measurements.
type Metrics interface {
	ObserveChangeTracking(duration time.Duration, entries int)
	ObserveSQL(operation string, duration time.Duration)
}

// NopMetrics is a no-op metrics recorder.
type NopMetrics struct{}

func (NopMetrics) ObserveChangeTracking(time.Duration, int) {}
func (NopMetrics) ObserveSQL(string, time.Duration)         {}
