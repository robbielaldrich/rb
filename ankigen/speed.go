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

// SpeedResult reports what a speed deck generator produced.
type SpeedResult struct {
	DeckFile string
	MediaDir string
	Notes    int
}

// GenerateReactionCards builds the deck for learning what each domain can
// answer with at reaction speed.
func GenerateReactionCards(opts Options) (SpeedResult, error) {
	return generateSpeedCards("Reaction", opts.ReactionDeckName, opts)
}

// GenerateActionCards builds the deck for learning what each domain can play
// at action speed.
func GenerateActionCards(opts Options) (SpeedResult, error) {
	return generateSpeedCards("Action", opts.ActionDeckName, opts)
}

// generateSpeedCards builds a deck for learning which cards a domain can play
// at a given speed, the keyword being Action or Reaction.
//
// Each domain gets one list per Energy cost it actually prints at that speed:
// "cards that cost 3 Energy". Exactly 3, whatever Power they also ask for —
// Power is paid from a domain's own runes and doesn't change which card this
// is. Each list is learnt a card at a time, as an overlapping cloze (see
// clozeRoster).
//
// The costs don't nest. A cumulative band — "3 Energy or less" — repeats every
// cheaper band inside itself, so the dearest one carries the whole roster and
// the cheap cards are re-read at every level above their own. The lists grow
// with the set and most of what grows is the repetition.
//
// It is cards rather than spells because the speed isn't a spell's alone: a
// gear or a unit can carry the keyword too. Legends are left out, since a
// legend is never played from hand, and so are signature cards: one may only
// be run by a deck whose legend is that champion, so it is not among the
// things a domain can answer with — it is something one deck in that domain
// brought with it.
func generateSpeedCards(speed, deckName string, opts Options) (SpeedResult, error) {
	cs, err := loadLegal(opts.CatalogPath)
	if err != nil {
		return SpeedResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	found := speedCards(cs, speed)
	if len(found) == 0 {
		return SpeedResult{}, fmt.Errorf("no %s cards in %s", speed, opts.CatalogPath)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return SpeedResult{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}
	images, err := renderScans(found, mediaDir, opts)
	if err != nil {
		return SpeedResult{}, err
	}

	d := deck{name: deckName, notetype: "Cloze"}
	rosters := byDomain(found)
	for _, domain := range slices.Sorted(maps.Keys(rosters)) {
		all := rosters[domain]
		slices.SortFunc(all, func(a, b cards.Card) int {
			return strings.Compare(a.BaseName(), b.BaseName())
		})
		// Only the costs the domain prints at: a list of nothing at 4 Energy
		// has no card to ask after.
		for _, energy := range energyCosts(all) {
			heading := fmt.Sprintf("<b>%s</b> cards in <b>%s</b> that cost <b>%d Energy</b>:", speed, domain, energy)
			d.notes = append(d.notes, clozeRoster(heading, atCost(all, energy), images, []string{
				"riftbound::" + strings.ToLower(speed) + "-cards",
				"riftbound::domain::" + strings.ToLower(domain),
				"riftbound::energy::" + strconv.Itoa(energy),
			})...)
		}
	}

	deckFile := filepath.Join(opts.OutDir, DeckFile)
	if err := d.write(deckFile); err != nil {
		return SpeedResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return SpeedResult{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes)}, nil
}

// speedCards collects every playable card carrying the keyword, one entry per
// card: the same card printed twice would otherwise land in a domain's list
// under both its alternate art and its plain printing.
//
// Which printing stands for the card matters, because the roster names the
// set beside it. A card also handed out as a promo is kept as its plain
// printing: Lunar Boon is an Unleashed card that was also given away, and the
// giveaway art says nothing about where to find it.
func speedCards(cs []cards.Card, keyword string) []cards.Card {
	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if c.Classification.Type == cards.TypeLegend || c.IsSignature() || !c.HasKeyword(keyword) || isManaAbility(c) {
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

// energyCost reads a card's Energy attribute, treating the absence of one as a
// spell that costs no Energy at all rather than an unknown cost.
func energyCost(c cards.Card) int {
	if c.Attributes.Energy == nil {
		return 0
	}
	return *c.Attributes.Energy
}

// energyCosts lists, in ascending order, every Energy cost the cards are
// actually printed at.
func energyCosts(cs []cards.Card) []int {
	seen := map[int]bool{}
	for _, c := range cs {
		seen[energyCost(c)] = true
	}
	return slices.Sorted(maps.Keys(seen))
}

// atCost keeps the cards costing exactly that much Energy, in the order they
// were given.
func atCost(cs []cards.Card, energy int) []cards.Card {
	var out []cards.Card
	for _, c := range cs {
		if energyCost(c) == energy {
			out = append(out, c)
		}
	}
	return out
}

// isManaAbility reports whether a card's speed keyword heads an activated
// ability that only adds resources, as Dragonsoul Sage's "[Reaction][>]
// Exhaust: [Add] 1 Energy" does. That keyword says when the ability can be
// used, not when the card can be played, and an ability that adds resources
// can't be reacted to, so it is nothing an opponent has to play around.
func isManaAbility(c cards.Card) bool {
	text := strings.TrimSpace(c.Text.Plain)
	activated := false
	for strings.HasPrefix(text, "[") {
		end := strings.Index(text, "]")
		if end < 0 {
			return false
		}
		if k := text[1:end]; k == ">" || k == "&gt;" {
			activated = true
		}
		text = strings.TrimLeft(text[end+1:], " ")
	}
	if !activated {
		return false
	}
	// The ability runs to its first full stop: its cost, then its effect.
	ability, _, _ := strings.Cut(text, ".")
	return strings.Contains(ability, "[Add]")
}
