package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadSuites_DefaultsAndOverrides(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "t.yaml", `requests:
- path: /a
- path: /b
  validation:
    statusCode: 404
    contentType: text/plain
    length: 3
`)
	got, err := loadSuites([]string{p})
	if err != nil {
		t.Fatalf("loadSuites: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].expectStatus != 200 || got[0].expectType != "application/json" || got[0].expectLen != 0 {
		t.Errorf("defaults wrong: %+v", got[0])
	}
	if got[1].expectStatus != 404 || got[1].expectType != "text/plain" || got[1].expectLen != 3 {
		t.Errorf("overrides wrong: %+v", got[1])
	}
	if got[0].Method != "GET" {
		t.Errorf("default method = %q, want GET", got[0].Method)
	}
}

func TestLoadSuites_JSONStillWorks(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "t.json", `{"requests":[{"path":"/a"}]}`)
	got, err := loadSuites([]string{p})
	if err != nil {
		t.Fatalf("loadSuites: %v", err)
	}
	if len(got) != 1 || got[0].Path != "/a" {
		t.Errorf("got %+v", got)
	}
}

func TestLoadSuites_FileNotFound(t *testing.T) {
	if _, err := loadSuites([]string{"/no/such/file.yaml"}); err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestLoadSuites_BadYAML(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "bad.yaml", "requests: [unclosed")
	if _, err := loadSuites([]string{p}); err == nil {
		t.Fatal("want error")
	}
}

func TestLoadSuites_PathRequired(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "x.yaml", "requests:\n- validation: {}\n")
	if _, err := loadSuites([]string{p}); err == nil {
		t.Fatal("want error")
	}
}

func TestRunPass_PassAndFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		case "/wrong-code":
			w.WriteHeader(http.StatusInternalServerError)
		case "/wrong-type":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(`{}`))
		case "/wrong-len":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"x":1}`))
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	five := 5
	p := writeFile(t, dir, "t.yaml", `requests:
- path: /ok
- path: /wrong-code
- path: /wrong-type
- path: /wrong-len
  validation:
    length: 99
`)
	_ = five
	reqs, err := loadSuites([]string{p})
	if err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	w := newWriter(&sb, true)
	opts := options{url: srv.URL, threads: 1}
	runUntilDone(context.Background(), opts, reqs, w)

	if got := w.pass.Load(); got != 1 {
		t.Errorf("pass=%d want 1", got)
	}
	if got := w.fail.Load(); got != 3 {
		t.Errorf("fail=%d want 3", got)
	}
	out := sb.String()
	if !strings.Contains(out, "statusCode want=200 got=500") {
		t.Errorf("missing statusCode err in output:\n%s", out)
	}
	if !strings.Contains(out, "contentType want=application/json") {
		t.Errorf("missing contentType err in output:\n%s", out)
	}
	if !strings.Contains(out, "length want=99") {
		t.Errorf("missing length err in output:\n%s", out)
	}
}

func TestRunPass_Threads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	dir := t.TempDir()
	p := writeFile(t, dir, "t.yaml", `requests:
- path: /a
- path: /b
- path: /c
- path: /d
- path: /e
`)
	reqs, _ := loadSuites([]string{p})
	w := newWriter(io.Discard, false)
	opts := options{url: srv.URL, threads: 3, random: true}
	runUntilDone(context.Background(), opts, reqs, w)
	if got := w.pass.Load(); got != 5 {
		t.Errorf("pass=%d want 5", got)
	}
}

func TestRunUntilDone_DurationStopsLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	dir := t.TempDir()
	p := writeFile(t, dir, "t.yaml", "requests:\n- path: /a\n")
	reqs, _ := loadSuites([]string{p})
	w := newWriter(io.Discard, false)
	opts := options{url: srv.URL, threads: 1, loop: true, duration: 50 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	runUntilDone(ctx, opts, reqs, w)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("ran too long: %v", elapsed)
	}
	if w.pass.Load() == 0 {
		t.Errorf("expected at least one pass")
	}
}
