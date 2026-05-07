// Package store loads the movies/actors/ratings JSON catalog into memory and
// exposes read-only, indexed accessors plus a substring search over `q`.
//
// Schemas are inferred from src/data/{movies,actors,ratings}.json — see
// .copilot-tracking/2026-05-07-data-layer-research.md. Spec references in
// docs/spec.md §5–6.
//
// The store is fully populated by Load and is safe for concurrent reads.
// There are no mutating methods.
package store

import (
	"sort"
	"strconv"
	"strings"
)

// Role is one cast/crew entry inside a Movie. category is one of:
// actor, actress, director, producer, self.
type Role struct {
	Order      int      `json:"order"`
	ActorID    string   `json:"actorId"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Characters []string `json:"characters,omitempty"`
	Job        string   `json:"job,omitempty"`
}

// Movie is one record from movies.json with rating/votes merged in from
// ratings.json.
type Movie struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Year    int      `json:"year"`
	Runtime int      `json:"runtime"`
	Genres  []string `json:"genres"`
	Roles   []Role   `json:"roles"`
	Rating  float64  `json:"rating"`
	Votes   int      `json:"votes"`
}

// MovieRef is the trimmed cross-reference embedded in actors.json.
type MovieRef struct {
	MovieID string `json:"movieId"`
	Title   string `json:"title"`
}

// Actor is one record from actors.json. DeathYear == 0 means living/unknown.
type Actor struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	BirthYear  int        `json:"birthYear"`
	DeathYear  int        `json:"deathYear"`
	Profession []string   `json:"profession"`
	Movies     []MovieRef `json:"movies"`
}

// Store is the in-memory catalog with all indexes precomputed. Build it with
// Load.
type Store struct {
	moviesByID map[string]*Movie
	actorsByID map[string]*Actor

	movieIDs []string // sorted, all movie ids
	actorIDs []string // sorted, all actor ids
	genres   []string // sorted, distinct

	moviesByGenre        map[string][]string // genre -> sorted movie ids
	moviesByYear         map[int][]string    // year -> sorted movie ids
	moviesByRatingBucket map[int][]string    // floor(rating) -> sorted movie ids
	moviesByActor        map[string][]string // actorId -> sorted movie ids

	movieHaystack map[string]string // movie id -> lowercased searchable blob
	actorHaystack map[string]string // actor id -> lowercased searchable blob
}

// MovieByID returns the movie with the given id and whether it was found.
func (s *Store) MovieByID(id string) (*Movie, bool) {
	m, ok := s.moviesByID[id]
	return m, ok
}

// ActorByID returns the actor with the given id and whether it was found.
func (s *Store) ActorByID(id string) (*Actor, bool) {
	a, ok := s.actorsByID[id]
	return a, ok
}

// Movies returns every movie, sorted by id.
func (s *Store) Movies() []*Movie { return s.materialize(s.movieIDs) }

// Actors returns every actor, sorted by id.
func (s *Store) Actors() []*Actor { return s.materializeActors(s.actorIDs) }

// Genres returns the sorted, distinct genre list across all movies.
func (s *Store) Genres() []string {
	out := make([]string, len(s.genres))
	copy(out, s.genres)
	return out
}

// MoviesByGenre returns all movies with the given genre (case-insensitive),
// sorted by id. Empty slice if no match.
func (s *Store) MoviesByGenre(genre string) []*Movie {
	return s.materialize(s.moviesByGenre[normGenre(genre)])
}

// MoviesByYear returns all movies released in the given year, sorted by id.
func (s *Store) MoviesByYear(year int) []*Movie {
	return s.materialize(s.moviesByYear[year])
}

// MoviesByRatingBucket returns all movies whose floor(rating) equals bucket
// (e.g. bucket=8 returns ratings in [8.0, 9.0)).
func (s *Store) MoviesByRatingBucket(bucket int) []*Movie {
	return s.materialize(s.moviesByRatingBucket[bucket])
}

// MoviesByActor returns all movies the actor appears in (any role category),
// sorted by id.
func (s *Store) MoviesByActor(actorID string) []*Movie {
	return s.materialize(s.moviesByActor[actorID])
}

// RolesByMovie returns the role list for the given movie id. nil if unknown.
func (s *Store) RolesByMovie(movieID string) []Role {
	m, ok := s.moviesByID[movieID]
	if !ok {
		return nil
	}
	return m.Roles
}

// SearchMovies returns movies whose searchable blob contains q
// (case-insensitive substring). Empty q returns all movies.
func (s *Store) SearchMovies(q string) []*Movie {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return s.Movies()
	}
	ids := make([]string, 0, 16)
	for _, id := range s.movieIDs {
		if strings.Contains(s.movieHaystack[id], q) {
			ids = append(ids, id)
		}
	}
	return s.materialize(ids)
}

// SearchActors returns actors whose searchable blob contains q
// (case-insensitive substring). Empty q returns all actors.
func (s *Store) SearchActors(q string) []*Actor {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return s.Actors()
	}
	ids := make([]string, 0, 16)
	for _, id := range s.actorIDs {
		if strings.Contains(s.actorHaystack[id], q) {
			ids = append(ids, id)
		}
	}
	return s.materializeActors(ids)
}

// Counts is the small struct returned by Stats for startup logging.
type Counts struct {
	Movies int
	Actors int
	Genres int
}

// Stats returns record counts for one-shot startup logging.
func (s *Store) Stats() Counts {
	return Counts{Movies: len(s.movieIDs), Actors: len(s.actorIDs), Genres: len(s.genres)}
}

// --- internal helpers ---

func (s *Store) materialize(ids []string) []*Movie {
	out := make([]*Movie, 0, len(ids))
	for _, id := range ids {
		if m, ok := s.moviesByID[id]; ok {
			out = append(out, m)
		}
	}
	return out
}

func (s *Store) materializeActors(ids []string) []*Actor {
	out := make([]*Actor, 0, len(ids))
	for _, id := range ids {
		if a, ok := s.actorsByID[id]; ok {
			out = append(out, a)
		}
	}
	return out
}

func normGenre(g string) string { return strings.ToLower(strings.TrimSpace(g)) }

// buildIndexes populates every secondary index from moviesByID + actorsByID.
// movieIDs and actorIDs must already be set and sorted.
func (s *Store) buildIndexes() {
	s.moviesByGenre = map[string][]string{}
	s.moviesByYear = map[int][]string{}
	s.moviesByRatingBucket = map[int][]string{}
	s.moviesByActor = map[string][]string{}
	s.movieHaystack = map[string]string{}
	s.actorHaystack = map[string]string{}

	genreSet := map[string]struct{}{}

	for _, id := range s.movieIDs {
		m := s.moviesByID[id]
		for _, g := range m.Genres {
			key := normGenre(g)
			s.moviesByGenre[key] = append(s.moviesByGenre[key], id)
			genreSet[g] = struct{}{}
		}
		s.moviesByYear[m.Year] = append(s.moviesByYear[m.Year], id)
		s.moviesByRatingBucket[int(m.Rating)] = append(s.moviesByRatingBucket[int(m.Rating)], id)

		seenActors := map[string]bool{}
		var sb strings.Builder
		sb.WriteString(strings.ToLower(m.Title))
		sb.WriteByte(' ')
		sb.WriteString(strconv.Itoa(m.Year))
		sb.WriteByte(' ')
		for _, g := range m.Genres {
			sb.WriteString(strings.ToLower(g))
			sb.WriteByte(' ')
		}
		for _, r := range m.Roles {
			if r.ActorID != "" && !seenActors[r.ActorID] {
				s.moviesByActor[r.ActorID] = append(s.moviesByActor[r.ActorID], id)
				seenActors[r.ActorID] = true
			}
			sb.WriteString(strings.ToLower(r.Name))
			sb.WriteByte(' ')
			for _, c := range r.Characters {
				sb.WriteString(strings.ToLower(c))
				sb.WriteByte(' ')
			}
		}
		s.movieHaystack[id] = sb.String()
	}

	for _, id := range s.actorIDs {
		a := s.actorsByID[id]
		var sb strings.Builder
		sb.WriteString(strings.ToLower(a.Name))
		sb.WriteByte(' ')
		for _, p := range a.Profession {
			sb.WriteString(strings.ToLower(p))
			sb.WriteByte(' ')
		}
		for _, mr := range a.Movies {
			sb.WriteString(strings.ToLower(mr.Title))
			sb.WriteByte(' ')
		}
		s.actorHaystack[id] = sb.String()
	}

	// Sort every per-key slice by id for stable iteration. movieIDs is
	// already sorted, so per-key slices are appended in id order — but be
	// explicit to survive any future loader change.
	for k := range s.moviesByGenre {
		sort.Strings(s.moviesByGenre[k])
	}
	for k := range s.moviesByYear {
		sort.Strings(s.moviesByYear[k])
	}
	for k := range s.moviesByRatingBucket {
		sort.Strings(s.moviesByRatingBucket[k])
	}
	for k := range s.moviesByActor {
		sort.Strings(s.moviesByActor[k])
	}

	s.genres = make([]string, 0, len(genreSet))
	for g := range genreSet {
		s.genres = append(s.genres, g)
	}
	sort.Strings(s.genres)
}
