// webv — minimal Web Validate-compatible HTTP request runner.
//
// Reads JSON test suites in the shape used by `test.json` at the repo root
// and issues HTTP requests against a base URL, validating the response
// status code, content type, and (optionally) body length. Designed for
// inner-loop smoke testing and running as a continuous load generator
// inside the cluster.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bartr/bartr-movies/internal/version"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "webv:", err)
		os.Exit(1)
	}
}

// fileList is a flag.Value that accepts repeated `-f` flags AND
// comma-separated values, e.g. `-f a.json -f b.json,c.json`.
type fileList []string

func (f *fileList) String() string { return strings.Join(*f, ",") }
func (f *fileList) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			*f = append(*f, p)
		}
	}
	return nil
}

type options struct {
	url      string
	files    fileList
	loop     bool
	threads  int
	random   bool
	duration time.Duration
	sleep    time.Duration
	verbose  bool
}

func run(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("webv", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts options
	var showVersion bool
	var sleepMs int

	fs.StringVar(&opts.url, "url", "", "base URL to test (http or https)")
	fs.StringVar(&opts.url, "u", "", "alias for --url")
	fs.Var(&opts.files, "files", "test file(s); repeatable or comma-separated")
	fs.Var(&opts.files, "f", "alias for --files")
	fs.BoolVar(&opts.loop, "loop", false, "run forever (overridden by --duration if set)")
	fs.BoolVar(&opts.loop, "l", false, "alias for --loop")
	fs.IntVar(&opts.threads, "threads", 1, "concurrent worker goroutines")
	fs.IntVar(&opts.threads, "t", 1, "alias for --threads")
	fs.BoolVar(&opts.random, "random", false, "execute requests in random order each pass")
	fs.BoolVar(&opts.random, "r", false, "alias for --random")
	fs.DurationVar(&opts.duration, "duration", 0, "total run time, e.g. 30s, 5m, 24h (takes precedence over --loop)")
	fs.DurationVar(&opts.duration, "d", 0, "alias for --duration")
	fs.IntVar(&sleepMs, "sleep", 0, "sleep N milliseconds between calls on each thread (0 = no sleep)")
	fs.IntVar(&sleepMs, "s", 0, "alias for --sleep")
	fs.BoolVar(&opts.verbose, "verbose", false, "log successes too")
	fs.BoolVar(&opts.verbose, "v", false, "alias for --verbose")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "webv — Web Validate-compatible HTTP request runner")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Usage: webv --url <base> --files <file>[,file...] [flags]")
		fmt.Fprintln(stderr, "")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if showVersion {
		fmt.Fprintln(stdout, version.Version)
		return nil
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if opts.url == "" {
		return errors.New("--url is required")
	}
	if !strings.HasPrefix(opts.url, "http://") && !strings.HasPrefix(opts.url, "https://") {
		return fmt.Errorf("--url must start with http:// or https://, got %q", opts.url)
	}
	opts.url = strings.TrimRight(opts.url, "/")
	if len(opts.files) == 0 {
		return errors.New("--files is required (one or more test files)")
	}
	if opts.threads < 1 {
		return fmt.Errorf("--threads must be >= 1, got %d", opts.threads)
	}
	if sleepMs < 0 {
		return fmt.Errorf("--sleep must be >= 0, got %d", sleepMs)
	}
	opts.sleep = time.Duration(sleepMs) * time.Millisecond

	suite, err := loadSuites(opts.files)
	if err != nil {
		return err
	}
	if len(suite) == 0 {
		return errors.New("no requests in test files")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if opts.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.duration)
		defer cancel()
	}

	w := newWriter(stdout, opts.verbose)
	defer w.summary()

	runUntilDone(ctx, opts, suite, w)
	return nil
}
