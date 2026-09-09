package ankigen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"rb/cards"
)

// ReactionResult reports what GenerateReactionSpells produced.
type ReactionResult struct {
	DeckFile string
	Notes    int
}

// reactionSpell is one Reaction spell, reduced to what the deck below needs
// to know about it.
type reactionSpell struct {
	name    string
	domains []string
	power   int
}

// GenerateReactionSpells builds a deck for learning which spells in each
// domain can be played as a Reaction at or under a given Power cost. Power,
// not Energy, is what a domain's own runes pay for, so it's the number a
// player actually has on hand to spend within one domain; the deck asks about
// it in bands — "2 Power or less" — rather than card by card, since knowing
// what's available within a budget is the skill worth drilling, not the exact
// cost of any one card.
func GenerateReactionSpells(opts Options) (ReactionResult, error) {
	cs, err := cards.Load(opts.CatalogPath)
	if err != nil {
		return ReactionResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	spells := reactionSpells(cs)
	if len(spells) == 0 {
		return ReactionResult{}, fmt.Errorf("no Reaction spells in %s", opts.CatalogPath)
	}

	d := deck{name: opts.ReactionDeckName, notetype: "Basic"}
	for _, domain := range domainsOf(spells) {
		for _, threshold := range powerBuckets(spells) {
			names := namesAtOrUnder(spells, domain, threshold)
			if len(names) == 0 {
				continue
			}
			d.notes = append(d.notes, note{
				front: fmt.Sprintf("What are all the Reaction spells in %s that cost %d Power or less?", domain, threshold),
				back:  spellList(names),
				tags: []string{
					"riftbound::reaction-spells",
					"riftbound::domain::" + strings.ToLower(domain),
					"riftbound::power::" + strconv.Itoa(threshold),
				},
			})
		}
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

// reactionSpells collects every Reaction spell, one entry per card: the same
// spell printed twice would otherwise land in a domain's list under both its
// alternate art and its plain printing.
func reactionSpells(cs []cards.Card) []reactionSpell {
	var out []reactionSpell
	seen := map[string]bool{}
	for _, c := range cs {
		if c.Classification.Type != "Spell" || !c.HasKeyword("Reaction") {
			continue
		}
		if seen[c.BaseName()] {
			continue
		}
		seen[c.BaseName()] = true
		out = append(out, reactionSpell{name: c.BaseName(), domains: c.Classification.Domain, power: powerCost(c)})
	}
	return out
}

// powerCost reads a card's Power attribute, treating the absence of one as a
// spell that costs no Power at all rather than an unknown cost.
func powerCost(c cards.Card) int {
	if c.Attributes.Power == nil {
		return 0
	}
	return *c.Attributes.Power
}

// domainsOf lists, in sorted order, every domain at least one Reaction spell
// belongs to.
func domainsOf(spells []reactionSpell) []string {
	seen := map[string]bool{}
	for _, s := range spells {
		for _, domain := range s.domains {
			seen[domain] = true
		}
	}
	domains := make([]string, 0, len(seen))
	for domain := range seen {
		domains = append(domains, domain)
	}
	slices.Sort(domains)
	return domains
}

// powerBuckets lists, in ascending order, every Power cost a Reaction spell
// is actually printed at. The thresholds a deck quizzes on come from the
// cards themselves rather than a fixed scale, so a set that never prints a
// 3-Power Reaction spell doesn't get an empty "3 or less" band for every
// domain.
func powerBuckets(spells []reactionSpell) []int {
	seen := map[int]bool{}
	for _, s := range spells {
		seen[s.power] = true
	}
	buckets := make([]int, 0, len(seen))
	for p := range seen {
		buckets = append(buckets, p)
	}
	slices.Sort(buckets)
	return buckets
}

// namesAtOrUnder lists, sorted, the Reaction spells in domain that cost
// threshold Power or less.
func namesAtOrUnder(spells []reactionSpell, domain string, threshold int) []string {
	var names []string
	for _, s := range spells {
		if s.power > threshold || !slices.Contains(s.domains, domain) {
			continue
		}
		names = append(names, s.name)
	}
	slices.Sort(names)
	return names
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
