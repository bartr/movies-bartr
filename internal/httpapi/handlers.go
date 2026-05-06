package httpapi

import (
	"io"
	"net/http"
)

const plainTextUTF8 = "text/plain; charset=utf-8"

// versionHandler returns the build's semver per spec §6.1: 200 OK,
// text/plain; charset=utf-8, body = "<semver>\n".
func versionHandler(version string) http.HandlerFunc {
	body := version + "\n"
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", plainTextUTF8)
		_, _ = io.WriteString(w, body)
	}
}

// healthzHandler returns plaintext liveness state. Session 1 always reports
// "pass"; richer warn/fail semantics are deferred.
func healthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", plainTextUTF8)
		_, _ = io.WriteString(w, "pass\n")
	}
}

// readyzHandler returns 200 "pass\n" only after ready() is true; otherwise
// 503 "not ready\n". Per spec §6, /readyz gates on dataset load.
func readyzHandler(ready ReadyFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", plainTextUTF8)
		if ready != nil && ready() {
			_, _ = io.WriteString(w, "pass\n")
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "not ready\n")
	}
}
