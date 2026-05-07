package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Request mirrors a single entry of the test.yaml/test.json `requests` array.
// Defaults: Method=GET, Validation.StatusCode=200, Validation.ContentType=application/json.
type Request struct {
	Path       string      `json:"path"       yaml:"path"`
	Method     string      `json:"method,omitempty"     yaml:"method,omitempty"`
	Validation *Validation `json:"validation,omitempty" yaml:"validation,omitempty"`

	// Resolved (non-pointer, post-defaults) — populated by loadSuites.
	expectStatus int
	expectType   string
	expectLen    int // 0 = no length check
	source       string
}

// Validation is the per-request expectations block in the suite file.
type Validation struct {
	StatusCode  *int    `json:"statusCode,omitempty"  yaml:"statusCode,omitempty"`
	ContentType *string `json:"contentType,omitempty" yaml:"contentType,omitempty"`
	Length      *int    `json:"length,omitempty"      yaml:"length,omitempty"`
}

type suite struct {
	Requests []Request `json:"requests" yaml:"requests"`
}

// decodeSuite picks the decoder by file extension. .yaml/.yml use YAML;
// everything else (and .json) uses strict JSON.
func decodeSuite(path string, raw []byte) (suite, error) {
	var s suite
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		dec := yaml.NewDecoder(strings.NewReader(string(raw)))
		dec.KnownFields(true)
		if err := dec.Decode(&s); err != nil {
			return s, err
		}
	default:
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&s); err != nil {
			return s, err
		}
	}
	return s, nil
}

func loadSuites(files []string) ([]Request, error) {
	var all []Request
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		s, err := decodeSuite(f, raw)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}
		for i := range s.Requests {
			r := &s.Requests[i]
			if r.Path == "" {
				return nil, fmt.Errorf("%s: request[%d]: path is required", f, i)
			}
			if !strings.HasPrefix(r.Path, "/") {
				return nil, fmt.Errorf("%s: request[%d]: path must start with /", f, i)
			}
			if r.Method == "" {
				r.Method = "GET"
			}
			r.expectStatus = 200
			r.expectType = "application/json"
			if r.Validation != nil {
				if r.Validation.StatusCode != nil {
					r.expectStatus = *r.Validation.StatusCode
				}
				if r.Validation.ContentType != nil {
					r.expectType = *r.Validation.ContentType
				}
				if r.Validation.Length != nil {
					r.expectLen = *r.Validation.Length
				}
			}
			r.source = f
		}
		all = append(all, s.Requests...)
	}
	return all, nil
}
