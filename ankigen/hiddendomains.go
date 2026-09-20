package ankigen

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"rb/cards"
)

// HiddenDomainResult reports what GenerateHiddenDomains produced.
type HiddenDomainResult struct {
	DeckFile string
	MediaDir string
	Notes    int
}

// rosterThumbWidth is how wide the card images are shown on the back. A
// domain can hold a dozen cards, so they are sized to sit several to a row
// and be recognised rather than read — the names above them carry the detail,
// and the image only has to bring the card to mind.
const rosterThumbWidth = 180

// GenerateHiddenDomains builds a deck for learning which cards in each domain
// carry Hidden.
//
// The Costs and Effects decks ask about one card at a time, which teaches
// recognition: shown a card, you recall what it does. At a table the question
// runs the other way. An opponent holds a face-down card and leaves a domain's
// runes open, and what matters is the whole set of things it could be, not
// what any one of them costs. This deck asks that direction — domain to
// roster — which no per-card deck can drill however well it is known.
//
// The back carries the scans as well as the names, since a roster recalled as
// a list of words is not yet a read: it is the cards themselves you have to
// see coming.
//
// A domain's roster grows with every set, so this is regenerated rather than
// edited. Anki takes a note's first field as its identity, and the question
// names only the domain, so a later run rewrites the list behind a question
// that has not changed — the note keeps its scheduling instead of coming back
// as a new card with the old one left beside it.
func GenerateHiddenDomains(opts Options) (HiddenDomainResult, error) {
	cs, err := cards.Load(opts.CatalogPath)
	if err != nil {
		return HiddenDomainResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	hidden := selectHidden(cs, false)
	if len(hidden) == 0 {
		return HiddenDomainResult{}, fmt.Errorf("no Hidden cards in %s", opts.CatalogPath)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return HiddenDomainResult{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}

	// A card in two domains belongs to both rosters but is one scan, so the
	// images are written once and looked up per domain.
	images := map[string]string{}
	for _, c := range hidden {
		if images[c.RiftboundID] != "" {
			continue
		}
		name, err := renderFull(c, mediaDir, opts)
		if err != nil {
			return HiddenDomainResult{}, err
		}
		images[c.RiftboundID] = name
	}

	d := deck{name: opts.HiddenDomainDeckName, notetype: "Basic"}
	byDomain := hiddenByDomain(hidden)
	for _, domain := range slices.Sorted(maps.Keys(byDomain)) {
		d.notes = append(d.notes, note{
			front: fmt.Sprintf("Which Hidden cards are available to %s?", domain),
			back:  roster(byDomain[domain], images),
			tags: []string{
				"riftbound::hidden-domains",
				"riftbound::domain::" + strings.ToLower(domain),
			},
		})
	}

	deckFile := filepath.Join(opts.OutDir, "riftbound-hidden-by-domain.txt")
	if err := d.write(deckFile); err != nil {
		return HiddenDomainResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return HiddenDomainResult{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes)}, nil
}

// hiddenByDomain groups the cards by the domains they belong to, each roster
// in name order so that a regenerated deck doesn't reshuffle a list the reader
// has half learnt. A card of two domains is listed under both, being playable
// out of either.
func hiddenByDomain(hidden []cards.Card) map[string][]cards.Card {
	out := map[string][]cards.Card{}
	for _, c := range hidden {
		for _, domain := range c.Classification.Domain {
			out[domain] = append(out[domain], c)
		}
	}
	for domain := range out {
		slices.SortFunc(out[domain], func(a, b cards.Card) int {
			return strings.Compare(a.BaseName(), b.BaseName())
		})
	}
	return out
}

// roster renders a note's back: how many cards the domain holds, the names
// with the set each was printed in, then the scans.
//
// The count leads because it is what makes the answer checkable — recalling
// nine of eleven cards feels like knowing the list until the number says
// otherwise. The set follows each name since a roster is only the roster of
// the format being played.
func roster(cs []cards.Card, images map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<b>%d %s</b><ul>", len(cs), plural(len(cs), "card"))
	for _, c := range cs {
		fmt.Fprintf(&b, "<li>%s <i>(%s)</i></li>", c.BaseName(), strings.ToUpper(c.Set.SetID))
	}
	b.WriteString("</ul>")
	for _, c := range cs {
		b.WriteString(thumb(images[c.RiftboundID], rosterThumbWidth))
	}
	return b.String()
}
