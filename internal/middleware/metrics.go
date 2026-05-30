package middleware

import (
	"strconv"
	"time"

	"warehouse-controller/internal/metrics"

	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		start := time.Now()
		c.Next()
		elapsed := time.Since(start).Seconds()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(elapsed)
		if c.Writer.Status() >= 500 {
			metrics.HTTPRequestsErrorsTotal.WithLabelValues(method, path, status).Inc()
		}
	}
}
