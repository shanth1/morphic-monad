package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Traffic (Rate & Errors)
	// API throughput (RPS) and error rate
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "morphic_http_requests_total",
			Help: "Total number of HTTP requests by endpoint and status",
		},
		[]string{"endpoint", "method", "status"},
	)

	// HTTP Latency (Duration)
	// User Response Time (Synchronous Ingest vs. Asynchronous Search)
	HTTPResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "morphic_http_response_time_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}, // From 10 ms to 10 sec
		},
		[]string{"endpoint"},
	)

	// Event Bus Traffic (EDA Observability)
	// Message Broker Load (Events Per Second - EPS)
	EventsPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "morphic_events_published_total",
			Help: "Total number of events published to NATS",
		},
		[]string{"topic", "event_type"},
	)

	// 4. Worker Processing Time (Compute & I/O)
	// Vectorization time (GPU/CPU) and VectorDB write time (I/O)
	WorkerProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "morphic_worker_processing_time_seconds",
			Help:    "Time spent processing an event by background workers",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"module", "operation"},
	)

	// Business Metrics (System State)
	// Knowledge base growth (Number of vectors/documents)
	VectorsUpsertedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "morphic_vectors_upserted_total",
			Help: "Total number of document chunks successfully saved to VectorDB",
		},
	)
)
