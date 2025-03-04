package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetricsMiddleware struct {
	reqCnt       *prometheus.CounterVec
	reqDur       *prometheus.HistogramVec
	reqSz, resSz prometheus.Summary
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}

func NewPrometheusMetricsMiddleware() *PrometheusMetricsMiddleware {

	m := &PrometheusMetricsMiddleware{}

	m.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "pocket_id",
			Name:      "requests_total",
			Help:      "How many HTTP requests processed, partitioned by status code and HTTP method.",
		},
		[]string{"code", "method", "handler", "host", "url"},
	)

	m.reqDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: "pocket_id",
			Name:      "request_duration_seconds",
			Help:      "The HTTP request latencies in seconds.",
		},
		[]string{"code", "method", "url"},
	)

	m.resSz = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "pocket_id",
			Name:      "response_size_bytes",
			Help:      "The HTTP response sizes in bytes.",
		},
	)

	m.reqSz = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "pocket_id",
			Name:      "request_size_bytes",
			Help:      "The HTTP request sizes in bytes.",
		},
	)

	prometheus.Register(m.reqCnt)
	prometheus.Register(m.reqDur)
	prometheus.Register(m.resSz)
	prometheus.Register(m.reqSz)

	return m
}

func (m *PrometheusMetricsMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip /metrics URL calls, as promhttp already tracks them
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		reqSz := computeApproximateRequestSize(c.Request)

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)
		resSz := float64(c.Writer.Size())

		m.reqDur.WithLabelValues(status, c.Request.Method, c.Request.URL.Path).Observe(elapsed)
		m.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, c.Request.URL.Path).Inc()
		m.reqSz.Observe(float64(reqSz))
		m.resSz.Observe(resSz)
	}
}
