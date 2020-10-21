package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	jobs = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "chainquery",
		Subsystem: "jobs",
		Name:      "duration",
		Help:      "The durations of the individual job processing",
	}, []string{"job"})

	// JobLoad metric for number of active calls by job
	JobLoad = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "chainquery",
		Subsystem: "jobs",
		Name:      "job_load",
		Help:      "Number of active calls by job",
	}, []string{"job"})

	// Notifications metric for total sent notifications by type
	Notifications = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "chainquery",
		Subsystem: "notifications",
		Name:      "total_sent",
		Help:      "sent notifications by type",
	}, []string{"type"})

	// ProcessingFailures metric for processing failure count by type
	ProcessingFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "chainquery",
		Subsystem: "processing",
		Name:      "failures",
		Help:      "processing failure count by type",
	}, []string{"type"})

	processing = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "chainquery",
		Subsystem: "processing",
		Name:      "duration",
		Help:      "The durations of the individual processing by type",
	}, []string{"type"})
)

//Job helper function to make tracking metric one line deferral
func Job(start time.Time, name string) {
	duration := time.Since(start).Seconds()
	jobs.WithLabelValues(name).Observe(duration)
}

//Processing helper function to make tracking metric one line deferral
func Processing(start time.Time, name string) {
	duration := time.Since(start).Seconds()
	processing.WithLabelValues(name).Observe(duration)
}
