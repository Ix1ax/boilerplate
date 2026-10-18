package httpapi

import (
	"crypto/subtle"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registerMetrics добавляет middleware и GET /metrics. Реестр свой на каждый роутер,
// метки — шаблон маршрута (FullPath), а не URL, чтобы число серий было ограничено.
func registerMetrics(r *gin.Engine, token string) {
	reg := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total", Help: "Запросы gateway"}, []string{"method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "Длительность запросов gateway", Buckets: prometheus.DefBuckets}, []string{"route"})
	reg.MustRegister(requests, duration, prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		requests.WithLabelValues(c.Request.Method, route, strconv.Itoa(c.Writer.Status())).Inc()
		duration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	r.GET("/metrics", func(c *gin.Context) {
		if token != "" {
			got := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				c.AbortWithStatus(401)
				return
			}
		}
		h.ServeHTTP(c.Writer, c.Request)
	})
}
