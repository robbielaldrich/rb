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
// An empty setID searches the whole catalog; naming a set narrows the search
// to it, for the common case of entering a stack of cards from one box.
func RunEditor(collectionPath, catalogPath, setID string) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	if setID != "" {
		if cs, err = filterSet(cs, setID); err != nil {
			return err
		}
	}

	if err := newEditor(coll, cs, collectionPath, setID).run(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("failed to run the collection editor: %w", err)
	}
	return nil
}

// filterSet keeps only the cards printed in the named set. An unknown label
// is refused here, before the editor opens, rather than leaving the user to
// wonder why nothing they type matches.
func filterSet(cs []cards.Card, setID string) ([]cards.Card, error) {
	var out []cards.Card
	known := map[string]bool{}
	for _, c := range cs {
		known[strings.ToUpper(c.Set.SetID)] = true
		if strings.EqualFold(c.Set.SetID, setID) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no cards in set %q, known sets: %s",
			setID, strings.Join(slices.Sorted(maps.Keys(known)), " "))
	}
	return out, nil
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

// save writes the collection atomically so an interrupted write can't leave a
// truncated file behind.
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
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", tmp, path, err)
	}
	return nil
}
