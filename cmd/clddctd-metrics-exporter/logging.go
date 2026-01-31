package main

import (
	"net/http"
	"time"

	exporter "clddctd-metrics-exporter/internal/exporter"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.status = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lrw, r)
		sourceIP := r.RemoteAddr
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			sourceIP = ip
		}
		exporter.Logf("info", "msg=\"http request\" method=%s path=%s status=%d dur_ms=%d src=%s", r.Method, r.URL.Path, lrw.status, time.Since(start).Milliseconds(), sourceIP)
	})
}
