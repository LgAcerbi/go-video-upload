package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/LgAcerbi/go-video-upload/pkg/metrics"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Metrics(w *metrics.Writer, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if w == nil {
				next.ServeHTTP(rw, r)
				return
			}
			start := time.Now()
			rec := &responseRecorder{ResponseWriter: rw, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			durationMs := time.Since(start).Milliseconds()
			path := r.URL.Path
			if path == "" {
				path = "/"
			}
			w.Record("http_request",
				map[string]string{
					"service": serviceName,
					"method":  r.Method,
					"path":    path,
					"status":  strconv.Itoa(rec.status),
				},
				map[string]interface{}{
					"count":       1,
					"duration_ms": durationMs,
				},
			)
		})
	}
}
