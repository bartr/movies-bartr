package config

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("MOVIES_DATA_DIR", "")
	t.Setenv("MOVIES_LOG_LEVEL", "")
	t.Setenv("MOVIES_PORT", "")
	c, err := Load(nil, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DataDir != "/data" || c.LogLevel != "info" || c.Port != 8080 {
		t.Fatalf("defaults wrong: %+v", c)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("MOVIES_DATA_DIR", "/tmp/d")
	t.Setenv("MOVIES_LOG_LEVEL", "warn")
	t.Setenv("MOVIES_PORT", "9090")
	c, err := Load(nil, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DataDir != "/tmp/d" || c.LogLevel != "warn" || c.Port != 9090 {
		t.Fatalf("env not applied: %+v", c)
	}
}

func TestLoad_FlagsBeatEnv(t *testing.T) {
	t.Setenv("MOVIES_DATA_DIR", "/from/env")
	t.Setenv("MOVIES_LOG_LEVEL", "warn")
	t.Setenv("MOVIES_PORT", "9090")
	args := []string{
		"--movies-data-dir=/from/flag",
		"--movies-log-level=DEBUG",
		"--movies-port=7777",
	}
	c, err := Load(args, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DataDir != "/from/flag" || c.LogLevel != "debug" || c.Port != 7777 {
		t.Fatalf("flag override failed: %+v", c)
	}
}

func TestLoad_HelpReturnsErrHelp(t *testing.T) {
	t.Setenv("MOVIES_DATA_DIR", "")
	var buf bytes.Buffer
	_, err := Load([]string{"--help"}, &buf)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("want flag.ErrHelp, got %v", err)
	}
	out := buf.String()
	for _, s := range []string{"movies-api", "--movies-port", "Effective values"} {
		if !strings.Contains(out, s) {
			t.Errorf("help missing %q\nout:\n%s", s, out)
		}
	}
}

func TestLoad_InvalidEnvPort(t *testing.T) {
	t.Setenv("MOVIES_PORT", "not-a-number")
	_, err := Load(nil, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "MOVIES_PORT") {
		t.Fatalf("want env-port error, got %v", err)
	}
}

func TestLoad_Validation(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{
			name: "bad log level",
			args: []string{"--movies-log-level=loud"},
			want: "log-level",
		},
		{
			name: "port too low",
			args: []string{"--movies-port=0"},
			want: "movies-port",
		},
		{
			name: "port too high",
			args: []string{"--movies-port=70000"},
			want: "movies-port",
		},
		{
			name: "empty data dir",
			args: []string{"--movies-data-dir="},
			want: "movies-data-dir",
		},
		{
			name: "positional arg",
			args: []string{"oops"},
			want: "positional",
		},
		{
			name: "unknown flag",
			args: []string{"--nope"},
			want: "flag provided but not defined",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("MOVIES_DATA_DIR", "")
			t.Setenv("MOVIES_LOG_LEVEL", "")
			t.Setenv("MOVIES_PORT", "")
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			_, err := Load(c.args, &bytes.Buffer{})
			if err == nil {
				t.Fatalf("want error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q, got %q", c.want, err)
			}
		})
	}
}

func TestRedacted(t *testing.T) {
	c := Config{DataDir: "/d", LogLevel: "info", Port: 8080}
	got := c.Redacted()
	for _, s := range []string{"data_dir=/d", "log_level=info", "port=8080"} {
		if !strings.Contains(got, s) {
			t.Errorf("Redacted missing %q: %s", s, got)
		}
	}
}
