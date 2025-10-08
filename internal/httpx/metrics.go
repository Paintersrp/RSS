package httpx

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	service         string
	gatherer        prometheus.Gatherer
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	externalLatency *prometheus.HistogramVec
}

func NewMetrics(service string) *Metrics {
	return &Metrics{
		service:  service,
		gatherer: prometheus.DefaultGatherer,
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "courier",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed.",
		}, []string{"service", "method", "path", "status"}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "courier",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Histogram of HTTP request durations in seconds.",
		}, []string{"service", "method", "path", "status"}),
		externalLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "courier",
			Subsystem: "external",
			Name:      "operation_duration_seconds",
			Help:      "Duration of external operations in seconds.",
		}, []string{"service", "component", "method", "status"}),
	}
}

func (m *Metrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{m.requestsTotal, m.requestDuration, m.externalLatency}
}

func (m *Metrics) Gatherer() prometheus.Gatherer {
	if m == nil {
		return prometheus.DefaultGatherer
	}
	return m.gatherer
}

func (m *Metrics) Middleware() echo.MiddlewareFunc {
	if m == nil {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			status := c.Response().Status
			if status == 0 {
				status = http.StatusOK
				if err != nil {
					if httpErr, ok := err.(*echo.HTTPError); ok {
						status = httpErr.Code
					} else {
						status = http.StatusInternalServerError
					}
				}
			}

			path := c.Path()
			if path == "" {
				path = c.Request().URL.Path
			}

			labels := []string{m.service, c.Request().Method, path, strconv.Itoa(status)}
			m.requestsTotal.WithLabelValues(labels...).Inc()
			m.requestDuration.WithLabelValues(labels...).Observe(time.Since(start).Seconds())

			return err
		}
	}
}

func (m *Metrics) ObserveDB(method string, err error, duration time.Duration) {
	m.observeExternal("db", method, err, duration)
}

func (m *Metrics) ObserveSearch(method string, err error, duration time.Duration) {
	m.observeExternal("meilisearch", method, err, duration)
}

func (m *Metrics) observeExternal(component, method string, err error, duration time.Duration) {
	if m == nil {
		return
	}
	status := "success"
	if err != nil {
		status = "error"
	}
	m.externalLatency.WithLabelValues(m.service, component, method, status).Observe(duration.Seconds())
}
