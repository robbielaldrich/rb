package ankigen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"rb/cards"
)

// ReactionResult reports what GenerateReactionSpells produced.
type ReactionResult struct {
	DeckFile string
	Notes    int
}

// GenerateReactionSpells builds a deck for learning which spells in each
// domain can be played as a Reaction, one note per domain listing every
// Reaction spell in it. Recognising that a spell is playable at instant speed
// matters more than any one card's wording, and a list is what that takes:
// image decks like the Hidden ones drill a single card, this drills a domain.
func GenerateReactionSpells(opts Options) (ReactionResult, error) {
	cs, err := cards.Load(opts.CatalogPath)
	if err != nil {
		return ReactionResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	byDomain := reactionSpellsByDomain(cs)
	if len(byDomain) == 0 {
		return ReactionResult{}, fmt.Errorf("no Reaction spells in %s", opts.CatalogPath)
	}

	domains := make([]string, 0, len(byDomain))
	for domain := range byDomain {
		domains = append(domains, domain)
	}
	slices.Sort(domains)

	d := deck{name: opts.ReactionDeckName, notetype: "Basic"}
	for _, domain := range domains {
		names := byDomain[domain]
		slices.Sort(names)
		d.notes = append(d.notes, note{
			front: fmt.Sprintf("What are all the Reaction spells in %s?", domain),
			back:  spellList(names),
			tags:  []string{"riftbound::reaction-spells", "riftbound::domain::" + strings.ToLower(domain)},
		})
	}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return ReactionResult{}, fmt.Errorf("failed to create %s: %w", opts.OutDir, err)
	}
	deckFile := filepath.Join(opts.OutDir, "riftbound-reaction-spells.txt")
	if err := d.write(deckFile); err != nil {
		return ReactionResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return ReactionResult{DeckFile: deckFile, Notes: len(d.notes)}, nil
}

// reactionSpellsByDomain groups the base name of every Reaction spell under
// each domain it belongs to; a multi-domain spell like Highlander is listed
// under both. Printings collapse to one entry per card, the same way the
// Hidden decks do, since alternate arts don't teach anything extra.
func reactionSpellsByDomain(cs []cards.Card) map[string][]string {
	byDomain := map[string][]string{}
	seen := map[string]bool{}
	for _, c := range cs {
		if c.Classification.Type != "Spell" || !c.HasKeyword("Reaction") {
			continue
		}
		if seen[c.BaseName()] {
			continue
		}
		seen[c.BaseName()] = true
		for _, domain := range c.Classification.Domain {
			byDomain[domain] = append(byDomain[domain], c.BaseName())
		}
	}
	return byDomain
}

// spellList renders a note's back as a bulleted list of card names.
func spellList(names []string) string {
	var b strings.Builder
	b.WriteString("<ul>")
	for _, n := range names {
		b.WriteString("<li>" + n + "</li>")
	}
	b.WriteString("</ul>")
	return b.String()
}
