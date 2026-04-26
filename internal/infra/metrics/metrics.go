package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counts the number of received HTTP requests (Ingest/Search)
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "morphic_http_requests_total",
			Help: "Total number of HTTP requests by endpoint",
		},
		[]string{"endpoint", "method", "status"},
	)

	// Measures the time it takes workers (Embedder, Engine) to process messages
	EventProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "morphic_event_processing_duration_seconds",
			Help:    "Time spent processing an event from the bus",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"module", "event_type"},
	)

	// Counts the number of vectorized chunks
	ChunksProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "morphic_chunks_processed_total",
			Help: "Total number of document chunks vectorized",
		},
	)
)
