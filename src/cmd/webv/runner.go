package main

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// result is one record emitted to the tab-delimited log.
type result struct {
	when     time.Time
	pass     bool
	code     int
	dur      time.Duration
	method   string
	path     string
	bytes    int
	contentT string
	errs     string
}

// writer serializes log lines and tallies pass/fail counters.
type writer struct {
	mu      sync.Mutex
	out     io.Writer
	verbose bool
	pass    atomic.Int64
	fail    atomic.Int64
}

func newWriter(out io.Writer, verbose bool) *writer {
	return &writer{out: out, verbose: verbose}
}

func (w *writer) emit(r result) {
	if r.pass {
		w.pass.Add(1)
		if !w.verbose {
			return
		}
	} else {
		w.fail.Add(1)
	}
	status := "PASS"
	if !r.pass {
		status = "FAIL"
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.out, "%s\t%s\t%d\t%dms\t%s\t%s\t%d\t%s\t%s\n",
		r.when.UTC().Format(time.RFC3339),
		status,
		r.code,
		r.dur.Milliseconds(),
		r.method,
		r.path,
		r.bytes,
		r.contentT,
		r.errs,
	)
}

func (w *writer) summary() {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.out, "# summary pass=%d fail=%d\n", w.pass.Load(), w.fail.Load())
}

// passComplete is called at the end of every pass; emits a heartbeat
// line so a long-running --loop run leaves visible signal in the log
// even when every request passes.
func (w *writer) passComplete() {
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.out, "# pass complete\tpass=%d\tfail=%d\n", w.pass.Load(), w.fail.Load())
}

// runUntilDone runs passes over `reqs` until ctx is done. Each pass either
// runs sequentially (single thread) or fans out across `opts.threads`
// workers. Random shuffles the order at the start of every pass.
func runUntilDone(ctx context.Context, opts options, reqs []Request, w *writer) {
	client := &http.Client{Timeout: 30 * time.Second}
	loop := opts.loop || opts.duration > 0

	for {
		order := reqs
		if opts.random {
			order = make([]Request, len(reqs))
			copy(order, reqs)
			rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		}
		runPass(ctx, client, opts, order, w)
		w.passComplete()
		if !loop || ctx.Err() != nil {
			return
		}
	}
}

func runPass(ctx context.Context, client *http.Client, opts options, reqs []Request, w *writer) {
	if opts.threads <= 1 {
		for i := range reqs {
			if ctx.Err() != nil {
				return
			}
			doOne(ctx, client, opts.url, &reqs[i], w)
			if opts.sleep > 0 && i < len(reqs)-1 {
				if !sleepCtx(ctx, opts.sleep) {
					return
				}
			}
		}
		return
	}
	ch := make(chan *Request)
	var wg sync.WaitGroup
	for i := 0; i < opts.threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			first := true
			for r := range ch {
				if ctx.Err() != nil {
					return
				}
				if !first && opts.sleep > 0 {
					if !sleepCtx(ctx, opts.sleep) {
						return
					}
				}
				first = false
				doOne(ctx, client, opts.url, r, w)
			}
		}()
	}
	for i := range reqs {
		select {
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			return
		case ch <- &reqs[i]:
		}
	}
	close(ch)
	wg.Wait()
}

// sleepCtx blocks for d or until ctx is cancelled. Returns true if the
// full sleep elapsed, false if ctx was cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func doOne(ctx context.Context, client *http.Client, base string, r *Request, w *writer) {
	url := base + r.Path
	start := time.Now()
	res := result{when: start, method: r.Method, path: r.Path}

	req, err := http.NewRequestWithContext(ctx, r.Method, url, nil)
	if err != nil {
		res.errs = "new-request: " + err.Error()
		res.dur = time.Since(start)
		w.emit(res)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		res.errs = "do: " + err.Error()
		res.dur = time.Since(start)
		w.emit(res)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	res.dur = time.Since(start)
	res.code = resp.StatusCode
	res.bytes = len(body)
	res.contentT = resp.Header.Get("Content-Type")

	var errs []string
	if resp.StatusCode != r.expectStatus {
		errs = append(errs, fmt.Sprintf("statusCode want=%d got=%d", r.expectStatus, resp.StatusCode))
	}
	// Skip content-type validation for 404s: the framework that produces
	// the response (chi default 404, RFC 7807 problem, plaintext, etc.)
	// varies by route shape and is not a contract we want to pin.
	if r.expectType != "" && r.expectStatus != 404 && !strings.HasPrefix(strings.ToLower(res.contentT), strings.ToLower(r.expectType)) {
		errs = append(errs, fmt.Sprintf("contentType want=%s got=%s", r.expectType, res.contentT))
	}
	if r.expectLen > 0 && r.expectLen != len(body) {
		errs = append(errs, fmt.Sprintf("length want=%d got=%d", r.expectLen, len(body)))
	}
	if len(errs) == 0 {
		res.pass = true
	} else {
		res.errs = strings.Join(errs, "; ")
	}
	w.emit(res)
}
