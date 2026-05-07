package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bartr/bartr-movies/internal/store"
)

// loadTestStore loads the canonical good fixture under
// internal/store/testdata/good. The path is resolved from this test file
// so the test works no matter where `go test` is invoked from.
func loadTestStore(t *testing.T) *store.Store {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "store", "testdata", "good")
	s, err := store.Load(dir)
	if err != nil {
		t.Fatalf("load test store: %v", err)
	}
	return s
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	s := loadTestStore(t)
	return NewRouter("9.9.9", func() bool { return true }, func() *store.Store { return s })
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rr.Result().Body).Decode(v); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func TestListMovies_HappyPath(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200, body=%s", rr.Code, body(t, rr))
	}
	if ct := rr.Header().Get("Content-Type"); ct != jsonContentType {
		t.Fatalf("content-type: %q", ct)
	}
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 3 {
		t.Fatalf("len: got %d want 3", len(got))
	}
	// Sorted by id.
	if got[0].ID != "tt0000001" || got[2].ID != "tt0000003" {
		t.Fatalf("order: %+v", got)
	}
}

func TestListMovies_FilterByGenre(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies?genre=Drama")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, body(t, rr))
	}
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 2 {
		t.Fatalf("drama count: got %d want 2", len(got))
	}
}

func TestListMovies_FilterByYear(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies?year=1999")
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 1 || got[0].ID != "tt0000001" {
		t.Fatalf("year filter: %+v", got)
	}
}

func TestListMovies_FilterByRating(t *testing.T) {
	r := newTestRouter(t)
	// fixture rating 8.5 → bucket 8 → tt0000001
	rr := do(t, r, "/api/movies?rating=8.5")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 1 || got[0].ID != "tt0000001" {
		t.Fatalf("rating filter: %+v", got)
	}
}

func TestListMovies_FilterByActor(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies?actorId=nm0000001")
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 2 {
		t.Fatalf("actor filter count: got %d want 2", len(got))
	}
}

func TestListMovies_Q(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies?q=alpha")
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 1 || got[0].ID != "tt0000001" {
		t.Fatalf("q filter: %+v", got)
	}
}

func TestListMovies_Pagination(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies?pageSize=2&pageNumber=2")
	var got []store.Movie
	decodeJSON(t, rr, &got)
	if len(got) != 1 || got[0].ID != "tt0000003" {
		t.Fatalf("page2: %+v", got)
	}
}

func TestGetMovie_HappyPath(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies/tt0000001")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, body(t, rr))
	}
	var got store.Movie
	decodeJSON(t, rr, &got)
	if got.ID != "tt0000001" || got.Title == "" {
		t.Fatalf("movie: %+v", got)
	}
}

func TestGetMovie_NotFound(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/movies/tt99999")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: got %d want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != problemContentType {
		t.Fatalf("content-type: %q", ct)
	}
}

func TestListActors_HappyPath(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/actors")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var got []store.Actor
	decodeJSON(t, rr, &got)
	if len(got) != 3 {
		t.Fatalf("len: %d", len(got))
	}
}

func TestListActors_Q(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/actors?q=alice")
	var got []store.Actor
	decodeJSON(t, rr, &got)
	if len(got) != 1 || got[0].Name != "Alice Anderson" {
		t.Fatalf("actor q: %+v", got)
	}
}

func TestGetActor_HappyPath(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/actors/nm0000001")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var got store.Actor
	decodeJSON(t, rr, &got)
	if got.ID != "nm0000001" {
		t.Fatalf("actor: %+v", got)
	}
}

func TestGetActor_NotFound(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/actors/nm99999")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: %d", rr.Code)
	}
}

func TestGenres_HappyPath(t *testing.T) {
	r := newTestRouter(t)
	rr := do(t, r, "/api/genres")
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	var got []string
	decodeJSON(t, rr, &got)
	want := []string{"Adventure", "Drama", "Fantasy", "Mystery", "Sci-Fi"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("genres: got %v want %v", got, want)
	}
}

// TestValidationNegatives is the table mirror of the negative cases in
// repo-root test.json. One row per validation rule; each must return 400
// with application/problem+json.
func TestValidationNegatives(t *testing.T) {
	r := newTestRouter(t)
	cases := []struct {
		name string
		path string
	}{
		// q
		{"q_too_short", "/api/movies?q=a"},
		{"q_too_long", "/api/movies?q=" + strings.Repeat("a", 21)},
		{"q_too_short_actors", "/api/actors?q=a"},
		{"q_too_long_actors", "/api/actors?q=" + strings.Repeat("a", 21)},

		// pageSize
		{"pagesize_zero", "/api/movies?pageSize=0"},
		{"pagesize_neg", "/api/movies?pageSize=-1"},
		{"pagesize_over", "/api/movies?pageSize=1001"},
		{"pagesize_decimal", "/api/movies?pageSize=10.1"},
		{"pagesize_alpha", "/api/movies?pageSize=foo"},

		// pageNumber
		{"pagenum_zero", "/api/movies?pageNumber=0"},
		{"pagenum_neg", "/api/movies?pageNumber=-1"},
		{"pagenum_over", "/api/movies?pageNumber=10001"},
		{"pagenum_decimal", "/api/movies?pageNumber=10.1"},
		{"pagenum_alpha", "/api/movies?pageNumber=foo"},

		// year
		{"year_neg", "/api/movies?year=-1"},
		{"year_one", "/api/movies?year=1"},
		{"year_too_old", "/api/movies?year=1873"},
		{"year_future", "/api/movies?year=2026"},
		{"year_decimal", "/api/movies?year=2020.1"},
		{"year_alpha", "/api/movies?year=foo"},

		// rating
		{"rating_neg", "/api/movies?rating=-1"},
		{"rating_over", "/api/movies?rating=10.1"},
		{"rating_alpha", "/api/movies?rating=foo"},

		// genre
		{"genre_short", "/api/movies?genre=ab"},
		{"genre_long", "/api/movies?genre=" + strings.Repeat("a", 21)},

		// actorId query
		{"actorid_short", "/api/movies?actorId=nm123"},
		{"actorid_wrong_prefix1", "/api/movies?actorId=ab12345"},
		{"actorid_wrong_prefix2", "/api/movies?actorId=tt12345"},
		{"actorid_uppercase", "/api/movies?actorId=NM12345"},
		{"actorid_zero", "/api/movies?actorId=nm00000"},

		// path movieId
		{"movieid_wrong_prefix1", "/api/movies/ab12345"},
		{"movieid_wrong_prefix2", "/api/movies/nm12345"},
		{"movieid_uppercase", "/api/movies/TT12345"},
		{"movieid_short", "/api/movies/tt123"},
		{"movieid_alpha_digits", "/api/movies/ttabcde"},
		{"movieid_zero", "/api/movies/tt00000"},

		// path actorId
		{"actorid_path_short", "/api/actors/ab12345"},
		{"actorid_path_wrong_prefix", "/api/actors/tt12345"},
		{"actorid_path_short_digits", "/api/actors/nm123"},
		{"actorid_path_alpha", "/api/actors/nmabcde"},
		{"actorid_path_zero", "/api/actors/nm00000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := do(t, r, tc.path)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status: got %d want 400 (body=%s)", rr.Code, body(t, rr))
			}
			if ct := rr.Header().Get("Content-Type"); ct != problemContentType {
				t.Fatalf("content-type: got %q want %q", ct, problemContentType)
			}
			var p problem
			decodeJSON(t, rr, &p)
			if p.Status != 400 || p.Detail == "" {
				t.Fatalf("problem body: %+v", p)
			}
		})
	}
}

// TestNotFoundPaths covers 404 path cases listed in test.json.
func TestNotFoundPaths(t *testing.T) {
	r := newTestRouter(t)
	cases := []string{
		"/api/actors/nm12345",        // valid id, no record
		"/api/movies/tt12345",        // valid id, no record
		"/api/actors/nm0000001/foo",  // nested path
		"/api/movies/tt0000001/foo",  // nested path
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			rr := do(t, r, path)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("status: got %d want 404 (body=%s)", rr.Code, body(t, rr))
			}
		})
	}
}

// TestStoreNotReady covers the 503 problem path when the store accessor
// returns nil (dataset still loading).
func TestStoreNotReady(t *testing.T) {
	r := NewRouter("0.0.0", func() bool { return false }, func() *store.Store { return nil })
	for _, path := range []string{
		"/api/movies",
		"/api/movies/tt0000001",
		"/api/actors",
		"/api/actors/nm0000001",
		"/api/genres",
	} {
		t.Run(path, func(t *testing.T) {
			rr := do(t, r, path)
			if rr.Code != http.StatusServiceUnavailable {
				t.Fatalf("status: got %d want 503", rr.Code)
			}
			if ct := rr.Header().Get("Content-Type"); ct != problemContentType {
				t.Fatalf("content-type: %q", ct)
			}
		})
	}
}

// TestNilStoreFunc covers the defensive nil-StoreFunc branch.
func TestNilStoreFunc(t *testing.T) {
	r := NewRouter("0.0.0", func() bool { return false }, nil)
	rr := do(t, r, "/api/movies")
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: %d", rr.Code)
	}
}
