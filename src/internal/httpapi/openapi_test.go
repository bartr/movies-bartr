package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRootRedirectsToSwagger(t *testing.T) {
	r := NewRouter("9.9.9", func() bool { return true }, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Fatalf("status: got %d want 301", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/swagger" {
		t.Fatalf("Location: got %q want %q", loc, "/swagger")
	}
}

func TestRobotsTxt(t *testing.T) {
	r := NewRouter("9.9.9", func() bool { return true }, nil)
	rr := do(t, r, "/robots.txt")

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got, want := rr.Header().Get("Content-Type"), plainTextUTF8; got != want {
		t.Fatalf("content-type: got %q want %q", got, want)
	}
	if got, want := body(t, rr), "User-agent: *\nDisallow: /\n"; got != want {
		t.Fatalf("body: got %q want %q", got, want)
	}
}

func TestSwaggerUI(t *testing.T) {
	r := NewRouter("9.9.9", func() bool { return true }, nil)
	for _, path := range []string{"/swagger", "/swagger/"} {
		rr := do(t, r, path)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status: got %d", path, rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Fatalf("%s content-type: %q", path, ct)
		}
		if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("%s X-Content-Type-Options: %q", path, got)
		}
		b := body(t, rr)
		if !strings.Contains(b, "swagger-ui") {
			t.Fatalf("%s body missing swagger-ui marker", path)
		}
		if !strings.Contains(b, "/swagger/v1/swagger.json") {
			t.Fatalf("%s body missing swagger.json url", path)
		}
	}
}

func TestSwaggerJSON(t *testing.T) {
	r := NewRouter("9.9.9", func() bool { return true }, nil)
	rr := do(t, r, "/swagger/v1/swagger.json")

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content-type: %q", got)
	}

	var doc map[string]any
	if err := json.NewDecoder(rr.Result().Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, _ := doc["openapi"].(string); !strings.HasPrefix(got, "3.") {
		t.Fatalf("openapi version: %v", doc["openapi"])
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing or wrong type")
	}
	for _, p := range []string{
		"/api/movies", "/api/movies/{id}",
		"/api/actors", "/api/actors/{id}",
		"/api/genres", "/healthz", "/readyz", "/version",
	} {
		if _, ok := paths[p]; !ok {
			t.Errorf("missing path %q in openapi doc", p)
		}
	}
}

// TestSwaggerJSONIsCompactAndStable verifies the served document is the
// compacted form of the embedded asset and bytes are stable across calls.
func TestSwaggerJSONIsCompactAndStable(t *testing.T) {
	r := NewRouter("9.9.9", func() bool { return true }, nil)
	first := mustGet(t, r, "/swagger/v1/swagger.json")
	second := mustGet(t, r, "/swagger/v1/swagger.json")
	if string(first) != string(second) {
		t.Fatal("swagger.json bytes not stable across calls")
	}
	// Compact JSON has no leading whitespace inside object/array openers.
	if strings.Contains(string(first), "\n  ") {
		t.Fatal("swagger.json should be compacted (no pretty-print whitespace)")
	}
}

func mustGet(t *testing.T, h http.Handler, path string) []byte {
	t.Helper()
	rr := do(t, h, path)
	b, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return b
}
