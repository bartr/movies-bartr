package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func do(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func body(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	b, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestVersionHandler(t *testing.T) {
	r := NewRouter("9.9.9", func() bool { return true })
	rr := do(t, r, "/version")
	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d want %d", got, want)
	}
	if got, want := rr.Header().Get("Content-Type"), plainTextUTF8; got != want {
		t.Fatalf("content-type: got %q want %q", got, want)
	}
	if got, want := body(t, rr), "9.9.9\n"; got != want {
		t.Fatalf("body: got %q want %q", got, want)
	}
}

func TestVersionIndependentOfReady(t *testing.T) {
	// /version must respond 200 even before /readyz is true (spec §6.1).
	r := NewRouter("0.1.0", func() bool { return false })
	rr := do(t, r, "/version")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rr.Code)
	}
	if body(t, rr) != "0.1.0\n" {
		t.Fatalf("body: got %q", body(t, rr))
	}
}

func TestHealthz(t *testing.T) {
	r := NewRouter("0.1.0", func() bool { return false })
	rr := do(t, r, "/healthz")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rr.Code)
	}
	if got, want := rr.Header().Get("Content-Type"), plainTextUTF8; got != want {
		t.Fatalf("content-type: %q", got)
	}
	if body(t, rr) != "pass\n" {
		t.Fatalf("body: got %q", body(t, rr))
	}
}

func TestReadyz(t *testing.T) {
	cases := []struct {
		name      string
		ready     bool
		wantCode  int
		wantBody  string
	}{
		{"not_ready", false, http.StatusServiceUnavailable, "not ready\n"},
		{"ready", true, http.StatusOK, "pass\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready := tc.ready
			r := NewRouter("0.1.0", func() bool { return ready })
			rr := do(t, r, "/readyz")
			if rr.Code != tc.wantCode {
				t.Fatalf("status: got %d want %d", rr.Code, tc.wantCode)
			}
			if got, want := rr.Header().Get("Content-Type"), plainTextUTF8; got != want {
				t.Fatalf("content-type: %q", got)
			}
			if got := body(t, rr); got != tc.wantBody {
				t.Fatalf("body: got %q want %q", got, tc.wantBody)
			}
		})
	}
}
