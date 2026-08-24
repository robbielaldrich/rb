package decks

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Entry is one line of a decklist: how many copies to play, and the card name
// exactly as the list spelled it.
type Entry struct {
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

// Section is one headed block of a decklist, e.g. "MainDeck" or "Rune Pool".
// Headings are kept as they were pasted rather than mapped onto a fixed set,
// so a list that carries a block we've never seen still records in full; only
// "Sideboard" is given a meaning, since it is the one block a deck can be
// played without.
type Section struct {
	Name  string  `json:"name"`
	Cards []Entry `json:"cards"`
}

// Deck is one registered decklist.
type Deck struct {
	Name     string    `json:"name"`
	AddedAt  time.Time `json:"added_at"`
	Sections []Section `json:"sections"`
}

func isSideboard(section string) bool {
	return strings.EqualFold(strings.TrimSpace(section), "sideboard")
}

// isRunePool reports whether a heading names the runes a deck brings.
//
// Runes aren't collected the way cards are — every product ships a pile of
// them and no list is held up for want of one — so they are recorded as
// pasted but left out of what the collection has to cover.
func isRunePool(section string) bool {
	return strings.Contains(strings.ToLower(section), "rune")
}

// section returns the block under the given heading, creating it if the deck
// doesn't carry one yet. Headings are matched case-insensitively so a list
// that writes "Maindeck" doesn't open a second block.
func (d *Deck) section(name string) *Section {
	for i := range d.Sections {
		if strings.EqualFold(d.Sections[i].Name, name) {
			return &d.Sections[i]
		}
	}
	d.Sections = append(d.Sections, Section{Name: name})
	return &d.Sections[len(d.Sections)-1]
}

// add records qty copies of a card under the given heading, folding a name
// the list happens to spell twice into one entry.
func (d *Deck) add(section, name string, qty int) {
	s := d.section(section)
	for i, e := range s.Cards {
		if strings.EqualFold(e.Name, name) {
			s.Cards[i].Quantity += qty
			return
		}
	}
	s.Cards = append(s.Cards, Entry{Quantity: qty, Name: name})
}

// Copies totals how many copies of each card the deck asks for, in name
// order. A card listed in both the main deck and the sideboard needs the two
// counts side by side, so the sections are summed rather than merged.
func (d Deck) Copies(sideboard bool) []Entry { return d.copies(sideboard, true) }

// Cards totals what the collection actually has to cover, which is the same
// list without the runes.
func (d Deck) Cards(sideboard bool) []Entry { return d.copies(sideboard, false) }

func (d Deck) copies(sideboard, runes bool) []Entry {
	want := map[string]int{}
	var order []string
	for _, s := range d.Sections {
		if !sideboard && isSideboard(s.Name) {
			continue
		}
		if !runes && isRunePool(s.Name) {
			continue
		}
		for _, e := range s.Cards {
			if _, seen := want[e.Name]; !seen {
				order = append(order, e.Name)
			}
			want[e.Name] += e.Quantity
		}
	}

	slices.Sort(order)
	out := make([]Entry, len(order))
	for i, name := range order {
		out[i] = Entry{Quantity: want[name], Name: name}
	}
	return out
}

// Size counts the copies a deck asks for, so a list can be reported by the
// number of cards it puts on the table.
func (d Deck) Size(sideboard bool) int { return total(d.Copies(sideboard)) }

func total(es []Entry) int {
	n := 0
	for _, e := range es {
		n += e.Quantity
	}
	return n
}

// fingerprint renders a deck's cards in a fixed order, so two pastes of the
// same list compare equal however they were laid out. The name and the date
// it was registered are deliberately left out: what makes a deck a duplicate
// is the cards in it.
func (d Deck) fingerprint() string {
	sections := slices.Clone(d.Sections)
	slices.SortFunc(sections, func(a, b Section) int {
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	var b strings.Builder
	for _, s := range sections {
		cards := slices.Clone(s.Cards)
		slices.SortFunc(cards, func(x, y Entry) int {
			return cmp.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
		})
		for _, e := range cards {
			b.WriteString(strings.ToLower(strings.TrimSpace(s.Name)))
			b.WriteByte('|')
			b.WriteString(strconv.Itoa(e.Quantity))
			b.WriteByte(' ')
			b.WriteString(strings.ToLower(strings.TrimSpace(e.Name)))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// plural picks the ending for a count, so a report can say "1 deck" without
// the parenthesised "(s)" that reads like a form.
func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// registry is the file of decks on disk, in the order they were added.
type registry struct {
	Decks []Deck `json:"decks"`
}

func loadRegistry(path string) (*registry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &registry{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var r registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &r, nil
}

// duplicate reports the registered deck that holds exactly these cards.
func (r *registry) duplicate(d Deck) (Deck, bool) {
	print := d.fingerprint()
	for _, have := range r.Decks {
		if have.fingerprint() == print {
			return have, true
		}
	}
	return Deck{}, false
}

func (r *registry) save(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal the deck register: %w", err)
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
