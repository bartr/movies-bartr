package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestLoggerEmitsJSON wires the router with a captured slog handler
// and asserts that one well-formed JSON request log line is emitted per
// request, with the expected fields and level.
func TestRequestLoggerEmitsJSON(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	r := NewRouter("9.9.9", func() bool { return true }, nil)

	cases := []struct {
		path     string
		wantCode int
		wantLvl  string
	}{
		{"/version", http.StatusOK, "INFO"},
		{"/api/movies", http.StatusServiceUnavailable, "ERROR"}, // store nil → 503
	}
	for _, tc := range cases {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != tc.wantCode {
			t.Fatalf("%s status: got %d want %d", tc.path, rr.Code, tc.wantCode)
		}

		// Expect at least one line with msg=http_request.
		var line map[string]any
		for _, raw := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("log line not JSON: %q", raw)
			}
			if m["msg"] == "http_request" {
				line = m
			}
		}
		if line == nil {
			t.Fatalf("%s: no http_request log line found in: %s", tc.path, buf.String())
		}
		if got, _ := line["level"].(string); got != tc.wantLvl {
			t.Errorf("%s level: got %q want %q", tc.path, got, tc.wantLvl)
		}
		for _, key := range []string{"method", "path", "status", "duration_ms", "remote", "user_agent"} {
			if _, ok := line[key]; !ok {
				t.Errorf("%s log missing key %q", tc.path, key)
			}
		}
		if got, _ := line["path"].(string); !strings.HasPrefix(got, tc.path) {
			t.Errorf("%s log path: got %q", tc.path, got)
		}
	}
}
