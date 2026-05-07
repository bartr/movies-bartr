package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestMetricsEndpoint_Exposes(t *testing.T) {
	r := newTestRouter(t)
	// Drive the middleware with one happy and one error response so the
	// counter has at least two label sets registered before /metrics renders.
	if rr := do(t, r, "/api/genres"); rr.Code != http.StatusOK {
		t.Fatalf("seed /api/genres: %d", rr.Code)
	}
	if rr := do(t, r, "/api/movies/tt12345"); rr.Code != http.StatusNotFound {
		t.Fatalf("seed /api/movies/{id}: %d", rr.Code)
	}

	rr := do(t, r, "/metrics")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, body(t, rr))
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content-type: %q", ct)
	}
	got := body(t, rr)

	for _, want := range []string{
		// Application instrumentation
		`http_requests_total`,
		`http_request_duration_seconds_bucket`,
		`http_requests_in_flight`,
		// Two-level route labels — the {id} segment is dropped so
		// /api/movies/{id} and /api/movies both record as /api/movies.
		`route="/api/movies"`,
		`route="/api/genres"`,
		`code="200"`,
		`code="404"`,
		// Standard Go runtime + process collectors.
		`go_goroutines`,
		`process_cpu_seconds_total`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in /metrics output", want)
		}
	}
	// Cardinality guards: raw ids must never become labels, and the
	// templated `{id}` segment must not leak through either.
	for _, banned := range []string{`tt12345`, `{id}`} {
		if strings.Contains(got, banned) {
			t.Errorf("/metrics leaked %q into labels (cardinality bug)", banned)
		}
	}
}

func TestMetricsEndpoint_DoesNotInstrumentItself(t *testing.T) {
	r := newTestRouter(t)
	// Seed at least one /api/* observation so the counter has a series
	// (HELP/TYPE only emit for vectors with an observed label set).
	if rr := do(t, r, "/api/genres"); rr.Code != http.StatusOK {
		t.Fatalf("seed /api/genres: %d", rr.Code)
	}
	// First call: middleware ignores the /metrics scrape itself (only
	// /api/* is measured), so the metrics output stays stable.
	_ = do(t, r, "/metrics")
	rr := do(t, r, "/metrics")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	got := body(t, rr)
	if !strings.Contains(got, "# HELP http_requests_total") {
		t.Errorf("missing HELP for http_requests_total")
	}
	if !strings.Contains(got, "# TYPE http_requests_total counter") {
		t.Errorf("missing TYPE counter for http_requests_total")
	}
	// /metrics itself must NOT show up as a measured route now that
	// the middleware filters to /api/*.
	if strings.Contains(got, `route="/metrics"`) {
		t.Errorf("/metrics scrape unexpectedly recorded itself")
	}
}

func TestMetrics_NonAPIRoutesNotMeasured(t *testing.T) {
	r := newTestRouter(t)
	// Non-/api/ traffic should never produce http_requests_total series.
	if rr := do(t, r, "/healthz"); rr.Code != http.StatusOK {
		t.Fatalf("seed /healthz: %d", rr.Code)
	}
	if rr := do(t, r, "/version"); rr.Code != http.StatusOK {
		t.Fatalf("seed /version: %d", rr.Code)
	}
	if rr := do(t, r, "/this/does/not/exist"); rr.Code != http.StatusNotFound {
		t.Fatalf("seed unrouted: %d", rr.Code)
	}
	rr := do(t, r, "/metrics")
	got := body(t, rr)
	for _, banned := range []string{
		`route="/healthz"`,
		`route="/version"`,
		`route="unmatched"`,
	} {
		if strings.Contains(got, banned) {
			t.Errorf("/metrics unexpectedly recorded %q", banned)
		}
	}
}

func TestAPIRouteLabel(t *testing.T) {
	cases := []struct {
		path    string
		want    string
		wantOK  bool
	}{
		{"/api/movies", "/api/movies", true},
		{"/api/movies/tt0133093", "/api/movies", true},
		{"/api/movies/", "/api/movies", true},
		{"/api/actors/nm0000206", "/api/actors", true},
		{"/api/genres", "/api/genres", true},
		{"/api/", "", false},
		{"/api", "", false},
		{"/healthz", "", false},
		{"/", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := apiRouteLabel(tc.path)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("apiRouteLabel(%q) = (%q, %v), want (%q, %v)",
				tc.path, got, ok, tc.want, tc.wantOK)
		}
	}
}
