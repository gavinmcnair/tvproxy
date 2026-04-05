package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func RequestLogger(log zerolog.Logger, debugFn func() bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)

			path := r.URL.Path
			if !debugFn() && isNoisyPath(path) {
				return
			}

			event := log.Info().
				Str("method", r.Method).
				Str("path", path).
				Int("status", ww.status).
				Dur("duration", time.Since(start)).
				Str("remote", r.RemoteAddr)
			if ua := r.UserAgent(); ua != "" && isHDHRPath(path) {
				event = event.Str("user_agent", ua)
			}
			event.Msg("request")
		})
	}
}

func isHDHRPath(path string) bool {
	return path == "/discover.json" || path == "/lineup.json" || path == "/lineup_status.json" ||
		path == "/device.xml" || path == "/capability" || strings.HasPrefix(path, "/channel/")
}

func isNoisyPath(path string) bool {
	if strings.HasPrefix(path, "/static/") {
		return true
	}
	if strings.HasSuffix(path, "/status") {
		return true
	}
	if path == "/dlna/device.xml" || path == "/favicon.ico" {
		return true
	}
	return false
}

type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.status = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
