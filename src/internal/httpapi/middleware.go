package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

// requestLogger emits one structured JSON log line per request via the
// default slog logger. It records method, path, raw query, status, response
// bytes, duration in milliseconds, remote address, and user agent.
//
// Per spec §7.2 the request body is never logged — the API is GET-only —
// and the query string is logged as-is.
func requestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Don't log Prometheus scrapes — they hit /metrics every 30s and
			// would drown out the signal in the request log.
			if r.URL.Path == "/metrics" {
				return
			}

			lvl := slog.LevelInfo
			switch {
			case rec.status >= 500:
				lvl = slog.LevelError
			case rec.status >= 400:
				lvl = slog.LevelWarn
			}

			slog.LogAttrs(r.Context(), lvl, "http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("remote", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// statusRecorder captures the response status and byte count for logging.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}
	s.status = code
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}
