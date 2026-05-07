package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Request mirrors a single entry of the test.json `requests` array.
// Defaults: Method=GET, Validation.StatusCode=200, Validation.ContentType=application/json.
type Request struct {
	Path       string      `json:"path"`
	Method     string      `json:"method,omitempty"`
	Validation *Validation `json:"validation,omitempty"`

	// Resolved (non-pointer, post-defaults) — populated by loadSuites.
	expectStatus int
	expectType   string
	expectLen    int // 0 = no length check
	source       string
}

// Validation is the per-request expectations block in the JSON file.
type Validation struct {
	StatusCode  *int    `json:"statusCode,omitempty"`
	ContentType *string `json:"contentType,omitempty"`
	Length      *int    `json:"length,omitempty"`
}

type suite struct {
	Requests []Request `json:"requests"`
}

func loadSuites(files []string) ([]Request, error) {
	var all []Request
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}
		var s suite
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&s); err != nil {
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
