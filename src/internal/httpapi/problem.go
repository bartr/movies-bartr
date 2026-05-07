package httpapi

import (
	"encoding/json"
	"net/http"
)

// problemContentType is the RFC 7807 media type used for every error
// response on /api/*.
const problemContentType = "application/problem+json"

// problem is the RFC 7807 body. Type defaults to "about:blank" and Title
// is derived from Status when omitted (helpers below).
type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

func newProblem(status int, detail string) *problem {
	return &problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
}

// writeProblem serializes p with Instance set from r.RequestURI and writes
// the response with the application/problem+json content type.
func writeProblem(w http.ResponseWriter, r *http.Request, p *problem) {
	if p.Instance == "" {
		p.Instance = r.RequestURI
	}
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}
