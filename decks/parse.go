package decks

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// entryRe matches a decklist line: a count, then the card name. The optional
// "x" covers the "3x Lightning Rush" spelling some builders export; it can't
// swallow the first letter of a name, since a space has to follow it.
var entryRe = regexp.MustCompile(`^([0-9]+)\s*[xX]?\s+(.+)$`)

// Parse reads a pasted decklist: headed blocks of "<count> <card name>"
// lines, as in exampledeck.txt. Blank lines are separators and carry no
// meaning, so a list can be pasted exactly as it was copied.
//
// The deck comes back named after its legend, which the caller is free to
// overwrite with something the tournament would recognise.
func Parse(text string) (Deck, error) {
	var d Deck
	section := ""
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if head, ok := strings.CutSuffix(line, ":"); ok {
			section = strings.TrimSpace(head)
			if section == "" {
				return Deck{}, fmt.Errorf("line %d: a section heading with no name", i+1)
			}
			d.section(section)
			continue
		}

		m := entryRe.FindStringSubmatch(line)
		if m == nil {
			return Deck{}, fmt.Errorf("line %d: %q is neither a section heading nor a \"<count> <card>\" line", i+1, line)
		}
		if section == "" {
			return Deck{}, fmt.Errorf("line %d: %q comes before any section heading", i+1, line)
		}
		qty, err := strconv.Atoi(m[1])
		if err != nil {
			return Deck{}, fmt.Errorf("line %d: failed to read the count in %q: %w", i+1, line, err)
		}
		if qty == 0 {
			return Deck{}, fmt.Errorf("line %d: %q asks for no copies", i+1, line)
		}
		d.add(section, strings.TrimSpace(m[2]), qty)
	}

	if d.Size(true) == 0 {
		return Deck{}, errors.New("no cards in this list")
	}
	d.Name = d.defaultName()
	return d, nil
}

// defaultName names a deck after its legend, the way players talk about
// decks, falling back to the first card listed for a list with no legend.
func (d Deck) defaultName() string {
	for _, s := range d.Sections {
		if strings.EqualFold(s.Name, "legend") && len(s.Cards) > 0 {
			return s.Cards[0].Name
		}
	}
	for _, s := range d.Sections {
		if len(s.Cards) > 0 {
			return s.Cards[0].Name
		}
	}
	return ""
}
