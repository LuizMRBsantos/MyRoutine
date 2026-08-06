package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Labels use the chi route pattern (e.g. /api/v1/habits/{habitID}), never the
// raw path — otherwise every habit id would create its own time series and
// blow up cardinality.
var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "myroutine_http_requests_total",
			Help: "Total HTTP requests processed, by method, route and status.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "myroutine_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "myroutine_http_requests_in_flight",
			Help: "HTTP requests currently being served.",
		},
	)
)

// Metrics records request count, latency and in-flight gauge for Prometheus.
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The scrape endpoint measuring itself is noise.
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			httpRequestsInFlight.Inc()
			defer httpRequestsInFlight.Dec()

			next.ServeHTTP(ww, r)

			// Resolved only after the handler runs — chi fills the pattern
			// as it walks the route tree.
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}

			elapsed := time.Since(start).Seconds()
			httpRequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(ww.Status())).Inc()
			httpRequestDuration.WithLabelValues(r.Method, route).Observe(elapsed)
		})
	}
}
