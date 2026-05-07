package httpapi

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Validation bounds. See spec.md §6 and docs/spec.md research note —
// frozen integers, not derived from build time, so behavior is
// deterministic across releases.
const (
	pageNumberMin = 1
	pageNumberMax = 10000
	pageSizeMin   = 1
	pageSizeMax   = 1000
	pageSizeDflt  = 25

	qMinLen     = 2
	qMaxLen     = 20
	genreMinLen = 3
	genreMaxLen = 20

	yearMin = 1874
	yearMax = 2025

	ratingMin = 0.0
	ratingMax = 10.0
)

var (
	movieIDRe = regexp.MustCompile(`^tt[0-9]{5,9}$`)
	actorIDRe = regexp.MustCompile(`^nm[0-9]{5,9}$`)
	zeroDigit = regexp.MustCompile(`^[0]+$`)
)

// validateMovieID returns nil if id is a well-formed, non-trivial movie id.
func validateMovieID(id string) *problem {
	return validateID("movieId", id, movieIDRe)
}

// validateActorID returns nil if id is a well-formed, non-trivial actor id.
func validateActorID(id string) *problem {
	return validateID("actorId", id, actorIDRe)
}

func validateID(name, id string, re *regexp.Regexp) *problem {
	if !re.MatchString(id) {
		return newProblem(400, fmt.Sprintf("%s %q does not match required pattern", name, id))
	}
	// Strip the two-letter prefix and reject all-zero digit ids
	// (tt00000, nm00000, ...). test.json mandates this.
	if zeroDigit.MatchString(id[2:]) {
		return newProblem(400, fmt.Sprintf("%s %q must not be all zeros", name, id))
	}
	return nil
}

// parseInt strictly parses raw as a base-10 integer. Empty raw uses dflt
// without error.
func parseInt(name, raw string, lo, hi, dflt int) (int, *problem) {
	if raw == "" {
		return dflt, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, newProblem(400, fmt.Sprintf("%s must be an integer, got %q", name, raw))
	}
	if n < lo || n > hi {
		return 0, newProblem(400, fmt.Sprintf("%s must be in [%d, %d], got %d", name, lo, hi, n))
	}
	return n, nil
}

// parseFloat strictly parses raw as a float. The empty string is treated as
// "unset" by callers; unset is signalled by returning ok=false.
func parseFloat(name, raw string, lo, hi float64) (val float64, ok bool, p *problem) {
	if raw == "" {
		return 0, false, nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false, newProblem(400, fmt.Sprintf("%s must be a number, got %q", name, raw))
	}
	if f < lo || f > hi {
		return 0, false, newProblem(400, fmt.Sprintf("%s must be in [%g, %g], got %g", name, lo, hi, f))
	}
	return f, true, nil
}

// validateLen enforces a length window when raw is non-empty. ok=false
// means the param is unset.
func validateLen(name, raw string, lo, hi int) (string, bool, *problem) {
	if raw == "" {
		return "", false, nil
	}
	n := len(raw)
	if n < lo || n > hi {
		return "", false, newProblem(400, fmt.Sprintf("%s length must be in [%d, %d], got %d", name, lo, hi, n))
	}
	return raw, true, nil
}

// movieFilters is the parsed/validated bundle for /api/movies.
type movieFilters struct {
	q          string
	hasQ       bool
	genre      string
	hasGenre   bool
	year       int
	hasYear    bool
	rating     float64
	hasRating  bool
	actorID    string
	hasActor   bool
	pageNumber int
	pageSize   int
}

func parseMovieFilters(values map[string]string) (movieFilters, *problem) {
	var f movieFilters
	var p *problem

	if f.q, f.hasQ, p = validateLen("q", strings.TrimSpace(values["q"]), qMinLen, qMaxLen); p != nil {
		return f, p
	}
	if f.genre, f.hasGenre, p = validateLen("genre", strings.TrimSpace(values["genre"]), genreMinLen, genreMaxLen); p != nil {
		return f, p
	}
	if y, perr := parseInt("year", values["year"], yearMin, yearMax, 0); perr != nil {
		return f, perr
	} else if values["year"] != "" {
		f.year, f.hasYear = y, true
	}
	if rt, ok, perr := parseFloat("rating", values["rating"], ratingMin, ratingMax); perr != nil {
		return f, perr
	} else if ok {
		f.rating, f.hasRating = rt, true
	}
	if aid := values["actorId"]; aid != "" {
		if perr := validateActorID(aid); perr != nil {
			return f, perr
		}
		f.actorID, f.hasActor = aid, true
	}

	pn, perr := parseInt("pageNumber", values["pageNumber"], pageNumberMin, pageNumberMax, 1)
	if perr != nil {
		return f, perr
	}
	ps, perr := parseInt("pageSize", values["pageSize"], pageSizeMin, pageSizeMax, pageSizeDflt)
	if perr != nil {
		return f, perr
	}
	f.pageNumber, f.pageSize = pn, ps
	return f, nil
}

// actorFilters is the parsed/validated bundle for /api/actors.
type actorFilters struct {
	q          string
	hasQ       bool
	pageNumber int
	pageSize   int
}

func parseActorFilters(values map[string]string) (actorFilters, *problem) {
	var f actorFilters
	var p *problem

	if f.q, f.hasQ, p = validateLen("q", strings.TrimSpace(values["q"]), qMinLen, qMaxLen); p != nil {
		return f, p
	}
	pn, perr := parseInt("pageNumber", values["pageNumber"], pageNumberMin, pageNumberMax, 1)
	if perr != nil {
		return f, perr
	}
	ps, perr := parseInt("pageSize", values["pageSize"], pageSizeMin, pageSizeMax, pageSizeDflt)
	if perr != nil {
		return f, perr
	}
	f.pageNumber, f.pageSize = pn, ps
	return f, nil
}

// page slices xs to the page window (1-based pageNumber). Out-of-range
// pages return an empty slice; this is not a validation error.
func page[T any](xs []T, pageNumber, pageSize int) []T {
	start := (pageNumber - 1) * pageSize
	if start >= len(xs) {
		return xs[len(xs):]
	}
	end := start + pageSize
	if end > len(xs) {
		end = len(xs)
	}
	return xs[start:end]
}

// firstQueryParam picks the first value for each query key. We do not allow
// repeated params to silently override one another; we just take values.Get.
func firstQueryParam(values map[string][]string) map[string]string {
	out := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}
