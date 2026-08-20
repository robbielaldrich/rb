package collection

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"rb/cards"
)

// Run loads the catalog and the collection file, then hands control to the
// terminal UI until the user quits.
func Run(catalogPath, collectionPath string) error {
	catalog, err := loadCatalog(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	m := NewModel(catalog, coll, collectionPath)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return fmt.Errorf("failed to run collection UI: %w", err)
	}
	return nil
}

// entry is one owned card. Cards are keyed by riftbound_id; the name, number
// and set are denormalised so the file stays readable on its own.
type entry struct {
	RiftboundID string    `json:"riftbound_id"`
	Name        string    `json:"name"`
	Number      string    `json:"number"`
	SetID       string    `json:"set_id"`
	Quantity    int       `json:"quantity"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type collection struct {
	Cards []entry `json:"cards"`
}

func load(path string) (*collection, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &collection{}, nil
	}
	if err != nil {
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
	c.Cards = append(c.Cards, entry{
		RiftboundID: card.RiftboundID,
		Name:        card.Name,
		Number:      cardNumber(card),
		SetID:       strings.ToUpper(card.Set.SetID),
		Quantity:    qty,
		UpdatedAt:   now,
	})
}

func (c *collection) total() int {
	var n int
	for _, e := range c.Cards {
		n += e.Quantity
	}
	return n
}

// save writes the collection atomically so an interrupted write can't leave a
// truncated file behind.
func (c *collection) save(path string) error {
	sorted := slices.Clone(c.Cards)
	slices.SortFunc(sorted, func(a, b entry) int {
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
