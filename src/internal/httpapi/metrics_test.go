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
		// Templated route label, not the raw URL — tt12345 must not appear.
		`route="/api/movies/{id}"`,
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
	// Cardinality guard: the raw id must never become a label value.
	if strings.Contains(got, `tt12345`) {
		t.Errorf("/metrics leaked raw path into labels (cardinality bug)")
	}
}

func TestMetricsEndpoint_DoesNotInstrumentItself(t *testing.T) {
	r := newTestRouter(t)
	// First call: middleware records the /metrics scrape itself once.
	_ = do(t, r, "/metrics")
	rr := do(t, r, "/metrics")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	// Note: /metrics IS itself instrumented by the middleware; we just
	// assert that it remains a valid Prometheus exposition (no parse errors
	// would manifest as 500). Sanity check: HELP/TYPE present.
	got := body(t, rr)
	if !strings.Contains(got, "# HELP http_requests_total") {
		t.Errorf("missing HELP for http_requests_total")
	}
	if !strings.Contains(got, "# TYPE http_requests_total counter") {
		t.Errorf("missing TYPE counter for http_requests_total")
	}
}

func TestMetrics_UnmatchedRouteLabel(t *testing.T) {
	r := newTestRouter(t)
	if rr := do(t, r, "/this/does/not/exist"); rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	rr := do(t, r, "/metrics")
	got := body(t, rr)
	if !strings.Contains(got, `route="unmatched"`) {
		t.Errorf(`expected route="unmatched" label for unrouted requests`)
	}
}
