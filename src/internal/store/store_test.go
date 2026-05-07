package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func mustLoad(t *testing.T, dir string) *Store {
	t.Helper()
	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load(%s): %v", dir, err)
	}
	return s
}

func ids(ms []*Movie) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.ID
	}
	return out
}

func actorIDs(as []*Actor) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.ID
	}
	return out
}

func eqStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLoad_Good(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	if got := s.Stats(); got.Movies != 3 || got.Actors != 3 || got.Genres != 5 {
		t.Fatalf("Stats=%+v want 3/3/5", got)
	}
	want := []string{"Adventure", "Drama", "Fantasy", "Mystery", "Sci-Fi"}
	if !eqStrings(s.Genres(), want) {
		t.Fatalf("Genres=%v want %v", s.Genres(), want)
	}
}

func TestLoad_Errors(t *testing.T) {
	cases := []struct {
		dir, want string
	}{
		{"testdata/no_rating", "no matching rating"},
		{"testdata/dup_movie", "duplicate id"},
		{"testdata/bad_id", "inconsistent id"},
		{"testdata/bad_json", "parse"},
		{"testdata/missing", "read"},
	}
	for _, c := range cases {
		t.Run(filepath.Base(c.dir), func(t *testing.T) {
			_, err := Load(c.dir)
			if err == nil {
				t.Fatalf("want error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %q", c.want, err)
			}
		})
	}
}

func TestMovieByID(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	m, ok := s.MovieByID("tt0000001")
	if !ok || m == nil {
		t.Fatalf("MovieByID miss")
	}
	if m.Title != "The Alpha File" || m.Rating != 8.5 || m.Votes != 1000 {
		t.Fatalf("unexpected movie: %+v", m)
	}
	if _, ok := s.MovieByID("tt9999999"); ok {
		t.Fatal("expected miss for unknown id")
	}
}

func TestActorByID(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	a, ok := s.ActorByID("nm0000002")
	if !ok || a.Name != "Bob Brown" || a.DeathYear != 2020 {
		t.Fatalf("ActorByID got %+v ok=%v", a, ok)
	}
	if _, ok := s.ActorByID("nm9999999"); ok {
		t.Fatal("expected miss")
	}
}

func TestMovies_Sorted(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	got := ids(s.Movies())
	want := []string{"tt0000001", "tt0000002", "tt0000003"}
	if !eqStrings(got, want) {
		t.Fatalf("Movies ids=%v want %v", got, want)
	}
}

func TestActors_Sorted(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	got := actorIDs(s.Actors())
	want := []string{"nm0000001", "nm0000002", "nm0000003"}
	if !eqStrings(got, want) {
		t.Fatalf("Actors ids=%v want %v", got, want)
	}
}

func TestMoviesByGenre(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	cases := []struct {
		genre string
		want  []string
	}{
		{"Drama", []string{"tt0000001", "tt0000003"}},
		{"drama", []string{"tt0000001", "tt0000003"}}, // case-insensitive
		{"Sci-Fi", []string{"tt0000003"}},
		{"Adventure", []string{"tt0000002"}},
		{"Western", nil}, // unknown returns empty
	}
	for _, c := range cases {
		t.Run(c.genre, func(t *testing.T) {
			got := ids(s.MoviesByGenre(c.genre))
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !eqStrings(got, c.want) {
				t.Fatalf("MoviesByGenre(%q)=%v want %v", c.genre, got, c.want)
			}
		})
	}
}

func TestMoviesByYear(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	if got := ids(s.MoviesByYear(1999)); !eqStrings(got, []string{"tt0000001"}) {
		t.Fatalf("year 1999=%v", got)
	}
	if got := ids(s.MoviesByYear(2009)); !eqStrings(got, []string{"tt0000003"}) {
		t.Fatalf("year 2009=%v", got)
	}
	if got := s.MoviesByYear(1900); len(got) != 0 {
		t.Fatalf("year 1900 should be empty, got %v", got)
	}
}

func TestMoviesByRatingBucket(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	// ratings: 8.5, 7.2, 9.1
	if got := ids(s.MoviesByRatingBucket(8)); !eqStrings(got, []string{"tt0000001"}) {
		t.Fatalf("bucket 8=%v", got)
	}
	if got := ids(s.MoviesByRatingBucket(7)); !eqStrings(got, []string{"tt0000002"}) {
		t.Fatalf("bucket 7=%v", got)
	}
	if got := ids(s.MoviesByRatingBucket(9)); !eqStrings(got, []string{"tt0000003"}) {
		t.Fatalf("bucket 9=%v", got)
	}
	if got := s.MoviesByRatingBucket(0); len(got) != 0 {
		t.Fatalf("bucket 0 should be empty, got %v", got)
	}
}

func TestMoviesByActor(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	cases := map[string][]string{
		"nm0000001": {"tt0000001", "tt0000002"},
		"nm0000002": {"tt0000001", "tt0000003"},
		"nm0000003": {"tt0000001", "tt0000002"},
		"nm9999999": nil,
	}
	for actor, want := range cases {
		got := ids(s.MoviesByActor(actor))
		if len(got) == 0 && len(want) == 0 {
			continue
		}
		if !eqStrings(got, want) {
			t.Fatalf("MoviesByActor(%s)=%v want %v", actor, got, want)
		}
	}
}

func TestRolesByMovie(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	r := s.RolesByMovie("tt0000001")
	if len(r) != 3 {
		t.Fatalf("want 3 roles, got %d", len(r))
	}
	if r[0].ActorID != "nm0000001" || r[0].Category != "actress" {
		t.Fatalf("unexpected first role: %+v", r[0])
	}
	if r := s.RolesByMovie("tt9999999"); r != nil {
		t.Fatalf("want nil for unknown movie, got %v", r)
	}
}

func TestSearchMovies(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	cases := []struct {
		q    string
		want []string
	}{
		{"alpha", []string{"tt0000001"}},                                       // title
		{"ALPHA", []string{"tt0000001"}},                                       // case-insensitive
		{"drama", []string{"tt0000001", "tt0000003"}},                          // genre
		{"2005", []string{"tt0000002"}},                                        // year
		{"alice", []string{"tt0000001", "tt0000002"}},                          // actor name
		{"detective pine", []string{"tt0000001"}},                              // character
		{"  ", []string{"tt0000001", "tt0000002", "tt0000003"}},                // whitespace = all
		{"", []string{"tt0000001", "tt0000002", "tt0000003"}},                  // empty = all
		{"zzznomatch", nil},
	}
	for _, c := range cases {
		t.Run(c.q, func(t *testing.T) {
			got := ids(s.SearchMovies(c.q))
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !eqStrings(got, c.want) {
				t.Fatalf("SearchMovies(%q)=%v want %v", c.q, got, c.want)
			}
		})
	}
}

func TestSearchActors(t *testing.T) {
	s := mustLoad(t, "testdata/good")
	cases := []struct {
		q    string
		want []string
	}{
		{"alice", []string{"nm0000001"}},
		{"director", []string{"nm0000003"}},                                // profession
		{"alpha", []string{"nm0000001", "nm0000002", "nm0000003"}},         // movie title (all 3 in Alpha File)
		{"", []string{"nm0000001", "nm0000002", "nm0000003"}},
		{"zzznomatch", nil},
	}
	for _, c := range cases {
		t.Run(c.q, func(t *testing.T) {
			got := actorIDs(s.SearchActors(c.q))
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !eqStrings(got, c.want) {
				t.Fatalf("SearchActors(%q)=%v want %v", c.q, got, c.want)
			}
		})
	}
}

// TestLoad_RealData is a smoke test against the committed dataset under
// src/data/ to make sure the schema inference matches the real files.
func TestLoad_RealData(t *testing.T) {
	s, err := Load("../../data")
	if err != nil {
		t.Fatalf("Load real data: %v", err)
	}
	c := s.Stats()
	if c.Movies != 100 {
		t.Errorf("movies=%d want 100", c.Movies)
	}
	if c.Actors != 553 {
		t.Errorf("actors=%d want 553", c.Actors)
	}
	if c.Genres < 10 {
		t.Errorf("genres=%d want >=10", c.Genres)
	}
	// Spot check: every movie has a non-zero rating.
	for _, m := range s.Movies() {
		if m.Rating <= 0 {
			t.Fatalf("movie %s has zero rating", m.ID)
		}
	}
}
