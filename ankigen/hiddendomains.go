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
// Each domain's list is learnt a card at a time, as an overlapping cloze (see
// clozeRoster), and each card is shown when its blank is revealed, since a
// roster recalled as a list of words is not yet a read: it is the cards
// themselves you have to see coming.
//
// A domain's roster grows with every set, so this is regenerated rather than
// edited. Each note's GUID is made from the domain and the card, so a later
// run rewrites the list around a card that was already there and the note
// keeps its scheduling.
func GenerateHiddenDomains(opts Options) (HiddenDomainResult, error) {
	cs, err := loadLegal(opts.CatalogPath)
	if err != nil {
		return HiddenDomainResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	// A signature card is unlocked by a legend rather than by a domain's
	// runes, so it is no answer to what a domain's open runes could be
	// holding; the signature deck asks after it from the legend's end.
	hidden := slices.DeleteFunc(selectHidden(cs, false), cards.Card.IsSignature)
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

	d := deck{name: opts.HiddenDomainDeckName, notetype: "Cloze"}
	rosters := byDomain(hidden)
	for _, domain := range slices.Sorted(maps.Keys(rosters)) {
		heading := fmt.Sprintf("<b>Hidden</b> cards available to <b>%s</b>:", domain)
		d.notes = append(d.notes, clozeRoster(heading, rosters[domain], images, []string{
			"riftbound::hidden-domains",
			"riftbound::domain::" + strings.ToLower(domain),
		})...)
	}

	deckFile := filepath.Join(opts.OutDir, DeckFile)
	if err := d.write(deckFile); err != nil {
		return HiddenDomainResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return HiddenDomainResult{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes)}, nil
}
