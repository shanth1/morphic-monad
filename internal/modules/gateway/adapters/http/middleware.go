package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/shanth1/morphic-monad/internal/infra/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next(rw, r)

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(rw.status)

		metrics.HTTPRequestsTotal.WithLabelValues(endpoint, r.Method, statusStr).Inc()
		metrics.HTTPResponseTime.WithLabelValues(endpoint).Observe(duration)
	}
}
