package cards

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Load reads cards.json and deduplicates it by riftbound_id. The API returns
// more than one record for some cards (a stale one and a refreshed one); the
// most recently updated record wins.
func Load(path string) ([]Card, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found, run `rb download-cards` first: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var all []Card
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	newest := make(map[string]Card, len(all))
	for _, c := range all {
		if prev, ok := newest[c.RiftboundID]; ok && prev.Metadata.UpdatedOn >= c.Metadata.UpdatedOn {
			continue
		}
		newest[c.RiftboundID] = c
	}

	out := make([]Card, 0, len(newest))
	for _, c := range newest {
		out = append(out, c)
	}
	// Alternate arts share a collector number with their plain printing, so
	// the id breaks the tie: without it the sort is free to order the two
	// differently on every run, and anything that picks one printing per card
	// changes its mind each time.
	slices.SortFunc(out, func(a, b Card) int {
		if n := cmp.Compare(a.Set.SetID, b.Set.SetID); n != 0 {
			return n
		}
		if n := cmp.Compare(a.CollectorNumber, b.CollectorNumber); n != 0 {
			return n
		}
		return cmp.Compare(a.RiftboundID, b.RiftboundID)
	})
	return out, nil
}

// riftIDRe matches the common riftbound_id shape, e.g. ven-044-166 or
// ven-044a-166, whose parts are the set, the collector number (with an
// optional variant letter), and the printed set size.
var riftIDRe = regexp.MustCompile(`^([a-z]+)-([0-9]+[a-z*]?)-([0-9]+)$`)

// Number renders a card's printed number, e.g. "44/166". Cards whose
// riftbound_id doesn't carry a set size (runes, some promos) fall back to
// their collector number alone.
func (c Card) Number() string {
	if m := riftIDRe.FindStringSubmatch(c.RiftboundID); m != nil {
		num := strings.TrimLeft(m[2], "0")
		if num == "" {
			num = "0"
		}
		return num + "/" + m[3]
	}
	return strconv.Itoa(c.CollectorNumber)
}

// IsAlternateArt reports whether this printing is an alternate art.
//
// The API flags most of them, but a dozen are missing the flag; their
// riftbound_id still carries the "a" that tells the printing apart from the
// plain one at the same collector number, e.g. ven-113a-166.
func (c Card) IsAlternateArt() bool {
	if c.Metadata.AlternateArt {
		return true
	}
	m := riftIDRe.FindStringSubmatch(c.RiftboundID)
	return m != nil && strings.HasSuffix(m[2], "a")
}

// Label renders a card the way it is written on paper, e.g.
// "Astral Heron 44/166 VEN".
func (c Card) Label() string {
	return fmt.Sprintf("%s %s %s", c.Name, c.Number(), strings.ToUpper(c.Set.SetID))
}

// variantRe matches the parenthesised suffix that distinguishes reprints of
// one card, e.g. "Teemo - Scout (Alternate Art)".
var variantRe = regexp.MustCompile(`\s*\([^()]*\)$`)

// BaseName is a card's name with any printing-variant suffix removed, so
// every printing of one card shares it.
func (c Card) BaseName() string {
	return variantRe.ReplaceAllString(c.Name, "")
}

// HasKeyword reports whether a card actually has the named keyword ability.
//
// Keywords are printed at the head of the rules text and run together in
// brackets, e.g. "[Hidden][Ganking]"; a few cards are missing the brackets in
// the API data and read "Hidden (Hide now for ...". Position is what
// distinguishes having a keyword from merely naming one: Teemo - Swift Scout
// lets you "hide a card with [Hidden]" without being Hidden itself.
func (c Card) HasKeyword(name string) bool {
	text := strings.TrimSpace(c.Text.Plain)

	// An unbracketed keyword is only recognised at the very start, and only
	// when its reminder text follows, so ordinary prose can't trip it.
	if rest, ok := cutFold(text, name); ok && strings.HasPrefix(rest, " (") {
		return true
	}

	for strings.HasPrefix(text, "[") {
		end := strings.Index(text, "]")
		if end < 0 {
			return false
		}
		if strings.EqualFold(text[1:end], name) {
			return true
		}
		text = strings.TrimLeft(text[end+1:], " ")
	}
	return false
}

func cutFold(s, prefix string) (rest string, ok bool) {
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return "", false
	}
	return s[len(prefix):], true
}
