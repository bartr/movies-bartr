// Package config loads runtime configuration from defaults < env < flags.
//
// Precedence (lowest to highest, per spec §11):
//  1. Built-in defaults
//  2. Environment variables (MOVIES_*)
//  3. Command-line flags (kebab-case of the env var with MOVIES_ prefix dropped)
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Config is the effective runtime configuration.
type Config struct {
	DataDir  string
	LogLevel string
	Port     int
}

// Defaults per spec §11.
func defaults() Config {
	return Config{
		DataDir:  "/data",
		LogLevel: "info",
		Port:     8080,
	}
}

// envOr returns the env var value if set & non-empty, otherwise fallback.
func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// envInt returns the env var parsed as int if set, else fallback. Returns an
// error if the env var is set but not a valid integer.
func envInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %q", key, v)
	}
	return n, nil
}

// Load parses configuration from defaults, env, and the provided args (which
// should NOT include the program name). It writes --help output to out.
//
// Returns the resolved Config. If the user passed -h/--help, returns
// (Config{}, flag.ErrHelp). On invalid input, returns a non-nil error.
func Load(args []string, out io.Writer) (Config, error) {
	d := defaults()

	// Env layer.
	envDataDir := envOr("MOVIES_DATA_DIR", d.DataDir)
	envLogLevel := envOr("MOVIES_LOG_LEVEL", d.LogLevel)
	envPort, err := envInt("MOVIES_PORT", d.Port)
	if err != nil {
		return Config{}, err
	}

	// Flag layer.
	fs := flag.NewFlagSet("movies-api", flag.ContinueOnError)
	fs.SetOutput(out)
	dataDir := fs.String("movies-data-dir", envDataDir, "directory containing data files (env: MOVIES_DATA_DIR, default: "+d.DataDir+")")
	logLevel := fs.String("movies-log-level", envLogLevel, "minimum log level: debug|info|warn|error (env: MOVIES_LOG_LEVEL, default: "+d.LogLevel+")")
	port := fs.Int("movies-port", envPort, "HTTP listen port (env: MOVIES_PORT, default: "+strconv.Itoa(d.Port)+")")

	fs.Usage = func() {
		fmt.Fprintln(out, "movies-api — read-only catalog HTTP API")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Flags (each maps to an env var; CLI flags override env, env overrides defaults):")
		fs.PrintDefaults()
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Effective values (after env, before flags):")
		fmt.Fprintf(out, "  --movies-data-dir   %s\n", envDataDir)
		fmt.Fprintf(out, "  --movies-log-level  %s\n", envLogLevel)
		fmt.Fprintf(out, "  --movies-port       %d\n", envPort)
	}

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}

	c := Config{
		DataDir:  *dataDir,
		LogLevel: strings.ToLower(*logLevel),
		Port:     *port,
	}

	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) validate() error {
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid --movies-log-level %q (want debug|info|warn|error)", c.LogLevel)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid --movies-port %d (want 1..65535)", c.Port)
	}
	if c.DataDir == "" {
		return fmt.Errorf("--movies-data-dir must not be empty")
	}
	return nil
}

// Redacted returns a string suitable for one-line startup logging. No secrets
// today, but this is the seam for future redaction.
func (c Config) Redacted() string {
	return fmt.Sprintf("data_dir=%s log_level=%s port=%d", c.DataDir, c.LogLevel, c.Port)
}
