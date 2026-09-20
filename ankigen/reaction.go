package ankigen

import (
	"fmt"
	"maps"
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
	MediaDir string
	Notes    int
}

// GenerateReactionSpells builds a deck for learning what each domain can
// answer with at instant speed.
//
// Each domain gets its whole roster, the question an open rune actually
// raises: they have Calm up, what can they be holding? Under it come the
// Power bands — "2 Power or less" — because Power, not Energy, is what a
// domain's own runes pay for, so it is the number a player has on hand to
// spend within one domain, and what is affordable is a narrower read than
// what exists.
//
// A band is only worth a note where it answers differently from the one below
// it and from the roster above it. Three of the four bands printed cover every
// Chaos spell there is, and a note whose answer is another note's answer is
// two cards to keep in step and one thing learnt.
func GenerateReactionSpells(opts Options) (ReactionResult, error) {
	cs, err := cards.Load(opts.CatalogPath)
	if err != nil {
		return ReactionResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	spells := reactionSpells(cs)
	if len(spells) == 0 {
		return ReactionResult{}, fmt.Errorf("no Reaction spells in %s", opts.CatalogPath)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return ReactionResult{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}
	images, err := renderScans(spells, mediaDir, opts)
	if err != nil {
		return ReactionResult{}, err
	}

	d := deck{name: opts.ReactionDeckName, notetype: "Basic"}
	rosters := byDomain(spells)
	for _, domain := range slices.Sorted(maps.Keys(rosters)) {
		all := rosters[domain]
		d.notes = append(d.notes, note{
			front: fmt.Sprintf("What are all the Reaction spells in %s?", domain),
			back:  roster(all, images),
			tags: []string{
				"riftbound::reaction-spells",
				"riftbound::domain::" + strings.ToLower(domain),
			},
		})

		var last int
		for _, threshold := range powerBuckets(spells) {
			within := atOrUnder(all, threshold)
			// Nothing to ask where the band is empty, where it holds what the
			// band below it held, or where it holds the whole roster already
			// asked for above.
			if len(within) == 0 || len(within) == last || len(within) == len(all) {
				continue
			}
			last = len(within)

			d.notes = append(d.notes, note{
				front: fmt.Sprintf("What are all the Reaction spells in %s that cost %d Power or less?", domain, threshold),
				back:  roster(within, images),
				tags: []string{
					"riftbound::reaction-spells",
					"riftbound::domain::" + strings.ToLower(domain),
					"riftbound::power::" + strconv.Itoa(threshold),
				},
			})
		}
	}

	deckFile := filepath.Join(opts.OutDir, "riftbound-reaction-spells.txt")
	if err := d.write(deckFile); err != nil {
		return ReactionResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return ReactionResult{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes)}, nil
}

// reactionSpells collects every Reaction spell, one entry per card: the same
// spell printed twice would otherwise land in a domain's list under both its
// alternate art and its plain printing.
//
// Which printing stands for the card matters, because the roster names the
// set beside it. A card handed out as an organized-play promo is kept as the
// printing from the set it belongs to: Lunar Boon is an Unleashed card that
// was also given away, and reading it as an OPP card says nothing about where
// to find it.
func reactionSpells(cs []cards.Card) []cards.Card {
	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if c.Classification.Type != "Spell" || !c.HasKeyword("Reaction") {
			continue
		}
		i, ok := at[c.BaseName()]
		if !ok {
			at[c.BaseName()] = len(out)
			out = append(out, c)
			continue
		}
		if out[i].IsPromo() && !c.IsPromo() {
			out[i] = c
		}
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

// powerBuckets lists, in ascending order, every Power cost a Reaction spell
// is actually printed at. The thresholds a deck quizzes on come from the
// cards themselves rather than a fixed scale, so a set that never prints a
// 3-Power Reaction spell doesn't get an empty "3 or less" band for every
// domain.
func powerBuckets(spells []cards.Card) []int {
	seen := map[int]bool{}
	for _, s := range spells {
		seen[powerCost(s)] = true
	}
	buckets := make([]int, 0, len(seen))
	for p := range seen {
		buckets = append(buckets, p)
	}
	slices.Sort(buckets)
	return buckets
}

// atOrUnder keeps the spells costing threshold Power or less, in the order
// they were given.
func atOrUnder(spells []cards.Card, threshold int) []cards.Card {
	var out []cards.Card
	for _, s := range spells {
		if powerCost(s) <= threshold {
			out = append(out, s)
		}
	}
	return out
}
