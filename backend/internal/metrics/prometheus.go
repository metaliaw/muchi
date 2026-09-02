package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	metrics := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "muchi_http_requests_total", Help: "HTTP requests by method and status."}, []string{"method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "muchi_http_request_duration_seconds", Help: "HTTP request duration."}, []string{"method"}),
	}
	prometheus.MustRegister(metrics.requests, metrics.duration)
	return metrics
}

func (m *Metrics) MeasureRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		m.requests.WithLabelValues(r.Method, strconv.Itoa(writer.status)).Inc()
		m.duration.WithLabelValues(r.Method).Observe(time.Since(started).Seconds())
	})
}

func Handler() http.Handler {
	return promhttp.Handler()
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
