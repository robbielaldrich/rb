package cards

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Set is one card set, as returned by the Riftcodex API (GET /sets) and
// stored in sets.json.
type Set struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SetID       string `json:"set_id"`
	CardCount   int    `json:"card_count"`
	PublishedOn string `json:"published_on"`
}

// publishedLayout is how sets.json spells a release date, e.g.
// "2026-07-31T00:00:00" — a local timestamp with no zone.
const publishedLayout = "2006-01-02T15:04:05"

// Published reads the date the set was released.
func (s Set) Published() (time.Time, error) {
	t, err := time.Parse(publishedLayout, s.PublishedOn)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read the release date of %s (%q): %w", s.SetID, s.PublishedOn, err)
	}
	return t, nil
}

// isPromo reports whether a set is a pile of promotional printings rather
// than a set that moves the format on. They are released alongside the real
// sets and would otherwise be read as the newest one.
func (s Set) isPromo() bool {
	return strings.Contains(strings.ToLower(s.Name), "promotional")
}

// LoadSets reads sets.json.
func LoadSets(path string) ([]Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found, run `rb download-cards` first: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var sets []Set
	if err := json.Unmarshal(data, &sets); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return sets, nil
}

// Latest returns the newest set on shelves as of the given day: the set that
// dates everything played under it. Sets announced but not yet released are
// left out, since a deck can't be built from one, and where two sets share a
// release date the larger is the one the format is named after — "Origins:
// Proving Grounds" ships with Origins but isn't what anyone calls the format.
func Latest(sets []Set, asOf time.Time) (Set, error) {
	var best Set
	var bestAt time.Time
	found := false
	for _, s := range sets {
		if s.isPromo() {
			continue
		}
		at, err := s.Published()
		if err != nil {
			return Set{}, err
		}
		if at.After(asOf) {
			continue
		}
		if found && (at.Before(bestAt) || at.Equal(bestAt) && s.CardCount <= best.CardCount) {
			continue
		}
		best, bestAt, found = s, at, true
	}
	if !found {
		return Set{}, fmt.Errorf("no set has been released as of %s", asOf.Format(time.DateOnly))
	}
	return best, nil
}
