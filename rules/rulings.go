package rules

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
)

// Ruling is one entry of rulings.json. That file is the source of truth for
// the Rules tab, so the field names have to match what the tab reads.
type Ruling struct {
	Category string   `json:"category"`
	Topic    string   `json:"topic"`
	Slug     string   `json:"slug"`
	Anchor   string   `json:"anchor"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Rules    []string `json:"rules"`
	Source   string   `json:"source"`
	Note     string   `json:"note"`
}

// Key identifies a ruling from one run to the next, so that a decision made
// about it can be filed and found again. Rewording a ruling keeps its key;
// only moving it under another anchor breaks the link.
func (r Ruling) Key() string {
	if r.Anchor != "" {
		return r.Slug + "#" + r.Anchor
	}
	// The hand-written Reddit notes all share the "reddit" slug and carry no
	// anchor, so their identity has to come from the question itself.
	h := fnv.New32a()
	h.Write([]byte(r.Question))
	return fmt.Sprintf("%s#%08x", r.Slug, h.Sum32())
}

// Load reads the ruling dataset in the order the file holds, which the Rules
// tab depends on and which makes a review walk the topics in turn.
func Load(path string) ([]Ruling, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var rs []Ruling
	if err := json.Unmarshal(data, &rs); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("%s holds no rulings", path)
	}
	return rs, nil
}
