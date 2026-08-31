package collection

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"rb/cards"
)

// RunEditor loads the catalog and collection and hands them to the
// interactive editor, which writes each change through as it is made.
//
// With no filters the whole catalog is searched; a set label or a domain
// narrows the search to it, for the common case of entering a stack of cards
// from one box or one pile.
func RunEditor(collectionPath, catalogPath string, filters []string) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	cs, scope, err := narrow(cs, filters)
	if err != nil {
		return err
	}

	if err := newEditor(coll, cs, collectionPath, scope).run(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("failed to run the collection editor: %w", err)
	}
	return nil
}

// narrow keeps only the cards matching every filter, each of which names
// either a set or a domain, in whichever order they were given. It returns
// the surviving cards and a label for the prompt.
//
// A filter matching neither is refused here, before the editor opens, rather
// than leaving the user to wonder why nothing they type matches.
func narrow(cs []cards.Card, filters []string) ([]cards.Card, string, error) {
	sets, domains := map[string]bool{}, map[string]bool{}
	for _, c := range cs {
		sets[strings.ToUpper(c.Set.SetID)] = true
		for _, d := range c.Classification.Domain {
			domains[strings.ToUpper(d)] = true
		}
	}

	var scope []string
	for _, f := range filters {
		f = strings.ToUpper(f)
		switch {
		case sets[f]:
			cs = keep(cs, func(c cards.Card) bool { return strings.EqualFold(c.Set.SetID, f) })
		case domains[f]:
			cs = keep(cs, func(c cards.Card) bool {
				return slices.ContainsFunc(c.Classification.Domain, func(d string) bool {
					return strings.EqualFold(d, f)
				})
			})
		default:
			return nil, "", fmt.Errorf("%q is neither a set nor a domain, known sets: %s, known domains: %s",
				f, labels(sets), labels(domains))
		}
		scope = append(scope, f)
	}

	if len(cs) == 0 {
		return nil, "", fmt.Errorf("no cards match %s", strings.Join(scope, " and "))
	}
	return cs, strings.Join(scope, " "), nil
}

func keep(cs []cards.Card, ok func(cards.Card) bool) []cards.Card {
	var out []cards.Card
	for _, c := range cs {
		if ok(c) {
			out = append(out, c)
		}
	}
	return out
}

func labels(set map[string]bool) string {
	return strings.Join(slices.Sorted(maps.Keys(set)), " ")
}

// collectedCard is one owned card. Cards are keyed by riftbound_id; the name, number
// and set are denormalised so the file stays readable on its own.
type collectedCard struct {
	RiftboundID string    `json:"riftbound_id"`
	Name        string    `json:"name"`
	Number      string    `json:"number"`
	SetID       string    `json:"set_id"`
	Quantity    int       `json:"quantity"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type collection struct {
	Cards []collectedCard `json:"cards"`
}

func load(path string) (*collection, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &collection{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var c collection
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return &c, nil
}

// set records qty copies of a card, dropping the entry entirely if qty is
// zero or less.
func (c *collection) set(card cards.Card, qty int, now time.Time) {
	for i, e := range c.Cards {
		if e.RiftboundID != card.RiftboundID {
			continue
		}
		if qty <= 0 {
			c.Cards = slices.Delete(c.Cards, i, i+1)
			return
		}
		c.Cards[i].Quantity = qty
		c.Cards[i].UpdatedAt = now
		return
	}
	if qty <= 0 {
		return
	}
	c.Cards = append(c.Cards, collectedCard{
		RiftboundID: card.RiftboundID,
		Name:        card.Name,
		Number:      card.Number(),
		SetID:       strings.ToUpper(card.Set.SetID),
		Quantity:    qty,
		UpdatedAt:   now,
	})
}

// save writes the collection out, sorted, so the file stays diffable.
func (c *collection) save(path string) error {
	sorted := slices.Clone(c.Cards)
	slices.SortFunc(sorted, func(a, b collectedCard) int {
		if n := cmp.Compare(a.SetID, b.SetID); n != 0 {
			return n
		}
		return cmp.Compare(a.RiftboundID, b.RiftboundID)
	})

	data, err := json.MarshalIndent(collection{Cards: sorted}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}
	return writeFile(path, append(data, '\n'))
}

// writeFile writes a file atomically, so an interrupted write can't leave a
// truncated one behind.
func writeFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", tmp, path, err)
	}
	return nil
}

// Owned reports how many copies of each card the collection holds, keyed by
// riftbound_id, for callers that only want the counts and not the file.
func Owned(path string) (map[string]int, error) {
	c, err := load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load collection: %w", err)
	}

	owned := make(map[string]int, len(c.Cards))
	for _, e := range c.Cards {
		owned[e.RiftboundID] += e.Quantity
	}
	return owned, nil
}
