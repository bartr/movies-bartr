package httpapi

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"

	"github.com/bartr/bartr-movies/internal/store"
	"github.com/go-chi/chi/v5"
)

// StoreFunc returns the loaded store, or nil if the dataset is not yet
// loaded. /api/* responds 503 problem+json while nil.
type StoreFunc func() *store.Store

const jsonContentType = "application/json; charset=utf-8"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// requireStore returns the store or writes a 503 problem and returns nil.
func requireStore(w http.ResponseWriter, r *http.Request, sf StoreFunc) *store.Store {
	if sf == nil {
		writeProblem(w, r, newProblem(http.StatusServiceUnavailable, "service not ready"))
		return nil
	}
	s := sf()
	if s == nil {
		writeProblem(w, r, newProblem(http.StatusServiceUnavailable, "service not ready"))
		return nil
	}
	return s
}

// listMoviesHandler implements GET /api/movies with full validation.
func listMoviesHandler(sf StoreFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := requireStore(w, r, sf)
		if s == nil {
			return
		}
		f, p := parseMovieFilters(firstQueryParam(r.URL.Query()))
		if p != nil {
			writeProblem(w, r, p)
			return
		}

		// Pick the cheapest seed list, then filter the rest in memory.
		var seed []*store.Movie
		switch {
		case f.hasActor:
			seed = s.MoviesByActor(f.actorID)
		case f.hasGenre:
			seed = s.MoviesByGenre(f.genre)
		case f.hasYear:
			seed = s.MoviesByYear(f.year)
		case f.hasQ:
			seed = s.SearchMovies(f.q)
		default:
			seed = s.Movies()
		}

		out := make([]*store.Movie, 0, len(seed))
		for _, m := range seed {
			if f.hasActor && !movieHasActor(m, f.actorID) {
				continue
			}
			if f.hasGenre && !movieHasGenre(m, f.genre) {
				continue
			}
			if f.hasYear && m.Year != f.year {
				continue
			}
			if f.hasRating && int(math.Floor(m.Rating)) != int(math.Floor(f.rating)) {
				continue
			}
			if f.hasQ && !movieMatchesQ(m, f.q) {
				continue
			}
			out = append(out, m)
		}
		writeJSON(w, http.StatusOK, page(out, f.pageNumber, f.pageSize))
	}
}

func movieHasActor(m *store.Movie, actorID string) bool {
	for _, r := range m.Roles {
		if r.ActorID == actorID {
			return true
		}
	}
	return false
}

func movieHasGenre(m *store.Movie, genre string) bool {
	g := strings.ToLower(strings.TrimSpace(genre))
	for _, mg := range m.Genres {
		if strings.ToLower(mg) == g {
			return true
		}
	}
	return false
}

func movieMatchesQ(m *store.Movie, q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if strings.Contains(strings.ToLower(m.Title), q) {
		return true
	}
	for _, g := range m.Genres {
		if strings.Contains(strings.ToLower(g), q) {
			return true
		}
	}
	for _, r := range m.Roles {
		if strings.Contains(strings.ToLower(r.Name), q) {
			return true
		}
		for _, c := range r.Characters {
			if strings.Contains(strings.ToLower(c), q) {
				return true
			}
		}
	}
	return false
}

// getMovieHandler implements GET /api/movies/{id}.
func getMovieHandler(sf StoreFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := requireStore(w, r, sf)
		if s == nil {
			return
		}
		id := chi.URLParam(r, "id")
		if p := validateMovieID(id); p != nil {
			writeProblem(w, r, p)
			return
		}
		m, ok := s.MovieByID(id)
		if !ok {
			writeProblem(w, r, newProblem(http.StatusNotFound, "movie not found"))
			return
		}
		writeJSON(w, http.StatusOK, m)
	}
}

// listActorsHandler implements GET /api/actors.
func listActorsHandler(sf StoreFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := requireStore(w, r, sf)
		if s == nil {
			return
		}
		f, p := parseActorFilters(firstQueryParam(r.URL.Query()))
		if p != nil {
			writeProblem(w, r, p)
			return
		}
		var seed []*store.Actor
		if f.hasQ {
			seed = s.SearchActors(f.q)
		} else {
			seed = s.Actors()
		}
		writeJSON(w, http.StatusOK, page(seed, f.pageNumber, f.pageSize))
	}
}

// getActorHandler implements GET /api/actors/{id}.
func getActorHandler(sf StoreFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := requireStore(w, r, sf)
		if s == nil {
			return
		}
		id := chi.URLParam(r, "id")
		if p := validateActorID(id); p != nil {
			writeProblem(w, r, p)
			return
		}
		a, ok := s.ActorByID(id)
		if !ok {
			writeProblem(w, r, newProblem(http.StatusNotFound, "actor not found"))
			return
		}
		writeJSON(w, http.StatusOK, a)
	}
}

// genresHandler implements GET /api/genres.
func genresHandler(sf StoreFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := requireStore(w, r, sf)
		if s == nil {
			return
		}
		writeJSON(w, http.StatusOK, s.Genres())
	}
}
