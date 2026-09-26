package ankigen

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// HiddenDomainResult reports what GenerateHiddenDomains produced.
type HiddenDomainResult struct {
	DeckFile string
	MediaDir string
	Notes    int
}

// GenerateHiddenDomains builds a deck for learning which cards in each domain
// carry Hidden.
//
// The Effects deck asks about one card at a time, which teaches
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
	cs, err := loadLegal(opts.CatalogPath)
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

	images, err := renderScans(hidden, mediaDir, opts)
	if err != nil {
		return HiddenDomainResult{}, err
	}

	d := deck{name: opts.HiddenDomainDeckName, notetype: "Basic"}
	rosters := byDomain(hidden)
	for _, domain := range slices.Sorted(maps.Keys(rosters)) {
		d.notes = append(d.notes, note{
			front: fmt.Sprintf("Which Hidden cards are available to %s?", domain),
			back:  roster(rosters[domain], images),
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
