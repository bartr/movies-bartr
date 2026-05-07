package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// rawMovie is the on-disk shape of one record in movies.json. We copy only
// the canonical id; key/_id/movieId are duplicates and validated.
type rawMovie struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	UID     string `json:"_id"`
	MovieID string `json:"movieId"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Year    int    `json:"year"`
	Runtime int    `json:"runtime"`
	Genres  []string
	Roles   []Role
}

type rawActor struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	UID        string `json:"_id"`
	ActorID    string `json:"actorId"`
	Type       string `json:"type"`
	Name       string
	BirthYear  int `json:"birthYear"`
	DeathYear  int `json:"deathYear"`
	Profession []string
	Movies     []MovieRef
}

type rawRating struct {
	ID      string  `json:"id"`
	Key     string  `json:"key"`
	UID     string  `json:"_id"`
	MovieID string  `json:"movieId"`
	Type    string  `json:"type"`
	Rating  float64 `json:"rating"`
	Votes   int     `json:"votes"`
}

// Load reads movies.json, actors.json, and ratings.json from dir, validates
// they cross-reference cleanly, and returns a fully indexed Store.
//
// Errors include: missing/unreadable file, malformed JSON, duplicate id,
// inconsistent canonical id (id != key/_id/movieId), or a movie without a
// matching rating record.
func Load(dir string) (*Store, error) {
	movies, err := readMovies(filepath.Join(dir, "movies.json"))
	if err != nil {
		return nil, err
	}
	actors, err := readActors(filepath.Join(dir, "actors.json"))
	if err != nil {
		return nil, err
	}
	ratings, err := readRatings(filepath.Join(dir, "ratings.json"))
	if err != nil {
		return nil, err
	}

	// Merge ratings into movies.
	ratingByID := make(map[string]rawRating, len(ratings))
	for _, r := range ratings {
		if _, dup := ratingByID[r.ID]; dup {
			return nil, fmt.Errorf("ratings.json: duplicate id %q", r.ID)
		}
		ratingByID[r.ID] = r
	}

	s := &Store{
		moviesByID: make(map[string]*Movie, len(movies)),
		actorsByID: make(map[string]*Actor, len(actors)),
	}
	s.movieIDs = make([]string, 0, len(movies))
	for _, rm := range movies {
		if _, dup := s.moviesByID[rm.ID]; dup {
			return nil, fmt.Errorf("movies.json: duplicate id %q", rm.ID)
		}
		r, ok := ratingByID[rm.ID]
		if !ok {
			return nil, fmt.Errorf("movies.json: %q has no matching rating", rm.ID)
		}
		s.moviesByID[rm.ID] = &Movie{
			ID:      rm.ID,
			Title:   rm.Title,
			Year:    rm.Year,
			Runtime: rm.Runtime,
			Genres:  rm.Genres,
			Roles:   rm.Roles,
			Rating:  r.Rating,
			Votes:   r.Votes,
		}
		s.movieIDs = append(s.movieIDs, rm.ID)
	}
	sort.Strings(s.movieIDs)

	s.actorIDs = make([]string, 0, len(actors))
	for _, ra := range actors {
		if _, dup := s.actorsByID[ra.ID]; dup {
			return nil, fmt.Errorf("actors.json: duplicate id %q", ra.ID)
		}
		s.actorsByID[ra.ID] = &Actor{
			ID:         ra.ID,
			Name:       ra.Name,
			BirthYear:  ra.BirthYear,
			DeathYear:  ra.DeathYear,
			Profession: ra.Profession,
			Movies:     ra.Movies,
		}
		s.actorIDs = append(s.actorIDs, ra.ID)
	}
	sort.Strings(s.actorIDs)

	s.buildIndexes()
	return s, nil
}

func readMovies(path string) ([]rawMovie, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []rawMovie
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for i, m := range out {
		if m.ID == "" || m.ID != m.Key || m.ID != m.UID || m.ID != m.MovieID {
			return nil, fmt.Errorf("%s[%d]: inconsistent id (id=%q key=%q _id=%q movieId=%q)", path, i, m.ID, m.Key, m.UID, m.MovieID)
		}
	}
	return out, nil
}

func readActors(path string) ([]rawActor, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []rawActor
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for i, a := range out {
		if a.ID == "" || a.ID != a.Key || a.ID != a.UID || a.ID != a.ActorID {
			return nil, fmt.Errorf("%s[%d]: inconsistent id (id=%q key=%q _id=%q actorId=%q)", path, i, a.ID, a.Key, a.UID, a.ActorID)
		}
	}
	return out, nil
}

func readRatings(path string) ([]rawRating, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var out []rawRating
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for i, r := range out {
		if r.ID == "" || r.ID != r.Key || r.ID != r.UID || r.ID != r.MovieID {
			return nil, fmt.Errorf("%s[%d]: inconsistent id (id=%q key=%q _id=%q movieId=%q)", path, i, r.ID, r.Key, r.UID, r.MovieID)
		}
	}
	return out, nil
}
