package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"rb/cards"
)

// CardRef is a card a ruling's question names, and where its scan is.
type CardRef struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	// Landscape marks the battlefields, which are printed on their side. The
	// page reserves the right shape before the scan arrives, so a row of them
	// doesn't jump about as they load.
	Landscape bool `json:"landscape,omitempty"`
}

// LinkCards writes, for every ruling whose question names a card, the cards it
// names and their scans.
//
// The rules tab shows them beside the question, which means it needs a scan's
// URL, which lives in the catalog — and the catalog is 2.3 MB the tab would
// otherwise have no reason to fetch. So the matching is done here and the tab
// reads the handful of kilobytes that come out of it.
//
// Keyed by question text, which is unique across the dataset and is what the
// tab already has in hand. Rewording a question loses its cards until this is
// run again, which a reworded question wants anyway: it may name other cards.
func LinkCards(rulingsPath, catalogPath, outPath string) error {
	rs, err := Load(rulingsPath)
	if err != nil {
		return err
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	idx := indexNames(cs)
	out := map[string][]CardRef{}
	for _, r := range rs {
		if named := cardsNamedIn(r.Question, idx); len(named) > 0 {
			out[r.Question] = named
		}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal the ruling cards: %w", err)
	}
	if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	return nil
}

// named is a card name as the catalog prints it, indexed by the same name with
// its punctuation flattened.
type named struct {
	name      string
	image     string
	landscape bool
}

// flatten reduces text to its words and the spaces between them, leaving the
// case alone.
//
// Punctuation has to go because the two are written differently either side:
// the rulings say "Akshan, Mischievous" where Origins prints "Akshan -
// Mischievous", and Vendetta prints the comma itself. Case has to stay,
// because it is the only thing telling the card Block from the act of
// blocking, and half the shortest card names are ordinary words — Flash,
// Smite, Sacrifice, Abandon.
func flatten(s string) string {
	s = strings.NewReplacer("'", "", "’", "").Replace(s)

	var b strings.Builder
	space := true // leading spaces are dropped
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			space = false
		case !space:
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// indexNames collects every card the catalog prints, keyed by its flattened
// name. Printings of one card share a name, so the one kept is the plainest:
// an alternate art or a promo carries the same name and a scan of a different
// frame, and the frame a reader pictures is the ordinary one.
func indexNames(cs []cards.Card) map[string]named {
	idx := map[string]named{}
	plain := map[string]bool{}
	for _, c := range cs {
		if c.Media.ImageURL == "" {
			continue
		}
		key := flatten(c.BaseName())
		if key == "" {
			continue
		}

		ordinary := !c.IsChasePrinting() && !c.IsPromo()
		if _, seen := idx[key]; seen && (plain[key] || !ordinary) {
			continue
		}
		idx[key] = named{
			name:      c.BaseName(),
			image:     c.Media.ImageURL,
			landscape: c.Orientation == "landscape",
		}
		plain[key] = ordinary
	}
	return idx
}

// span is where a card's name sits in a flattened question.
type span struct {
	start, end int
	card       named
}

// cardsNamedIn finds the cards a question names, in the order they are named.
//
// A name only counts on whole words, and a name sitting inside a longer one is
// dropped with it: "Does Shadow Assassin count itself" names Shadow Assassin,
// not Vex's Shadow.
func cardsNamedIn(question string, idx map[string]named) []CardRef {
	q := flatten(question)

	var spans []span
	for key, c := range idx {
		for at := 0; ; {
			i := strings.Index(q[at:], key)
			if i < 0 {
				break
			}
			i += at
			at = i + 1
			if wholeWords(q, i, i+len(key)) {
				spans = append(spans, span{i, i + len(key), c})
			}
		}
	}

	var out []CardRef
	seen := map[string]bool{}
	for i, s := range spans {
		if within(spans, i) || seen[s.card.name] {
			continue
		}
		seen[s.card.name] = true
		out = append(out, CardRef{Name: s.card.name, Image: s.card.image, Landscape: s.card.landscape})
	}
	return out
}

// within reports whether the i'th span is covered by another, which is how a
// name that is part of a longer card's name is left out.
func within(spans []span, i int) bool {
	for j, o := range spans {
		if j != i && o.start <= spans[i].start && spans[i].end <= o.end && o.end-o.start > spans[i].end-spans[i].start {
			return true
		}
	}
	return false
}

func wholeWords(s string, start, end int) bool {
	return (start == 0 || s[start-1] == ' ') && (end == len(s) || s[end] == ' ')
}
