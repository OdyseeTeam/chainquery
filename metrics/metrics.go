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

	// ProcessingSchedulerEvents tracks dependency-aware block scheduler events.
	ProcessingSchedulerEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "chainquery",
		Subsystem: "processing",
		Name:      "scheduler_events",
		Help:      "dependency-aware block scheduler events",
	}, []string{"event"})

	// ProcessingSchedulerDependencyEdges tracks same-block dependency edges by reason.
	ProcessingSchedulerDependencyEdges = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "chainquery",
		Subsystem: "processing",
		Name:      "scheduler_dependency_edges",
		Help:      "same-block dependency edges by reason",
	}, []string{"reason"})

	processing = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "chainquery",
		Subsystem: "processing",
		Name:      "duration",
		Help:      "The durations of the individual processing by type",
	}, []string{"type"})

	// LBRYcrdRPCInflight tracks active lbrycrd JSON-RPC wrapper calls by method.
	LBRYcrdRPCInflight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "chainquery",
		Subsystem: "lbrycrd",
		Name:      "rpc_inflight",
		Help:      "Number of in-flight lbrycrd JSON-RPC calls by method",
	}, []string{"method"})

	// LBRYcrdRPCLatency tracks lbrycrd JSON-RPC wrapper latency by method and result.
	LBRYcrdRPCLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "chainquery",
		Subsystem: "lbrycrd",
		Name:      "rpc_duration",
		Help:      "The durations of lbrycrd JSON-RPC calls by method and result",
	}, []string{"method", "result"})

	// SocketyNotifications metric for processing failure count by type
	SocketyNotifications = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "chainquery",
		Subsystem: "sockety",
		Name:      "notifications",
		Help:      "counter for sending sockety notifications as the blockchain sourcex",
	}, []string{"type", "category", "subcategory"})
)

// Job helper function to make tracking metric one line deferral
func Job(start time.Time, name string) {
	duration := time.Since(start).Seconds()
	jobs.WithLabelValues(name).Observe(duration)
}

// Processing helper function to make tracking metric one line deferral
func Processing(start time.Time, name string) {
	duration := time.Since(start).Seconds()
	processing.WithLabelValues(name).Observe(duration)
}
