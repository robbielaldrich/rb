package ankigen

import (
	"cmp"
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
	return generateSpeedCards("Reaction", opts.ReactionDeckName, "riftbound-reaction-cards.txt", opts)
}

// GenerateActionCards builds the deck for learning what each domain can play
// at action speed.
func GenerateActionCards(opts Options) (SpeedResult, error) {
	return generateSpeedCards("Action", opts.ActionDeckName, "riftbound-action-cards.txt", opts)
}

// generateSpeedCards builds a deck for learning which cards a domain can play
// at a given speed, the keyword being Action or Reaction.
//
// Each domain gets one question per Energy cost, from 1 up to the dearest
// card it holds at that speed: "cards that cost 3 Energy or less". The top
// band is the whole roster, the question an open rune actually raises — they
// have Calm up, what can they be holding? — and the bands under it are the
// narrower read of what is affordable. A free card is in every band.
//
// It is cards rather than spells because the speed isn't a spell's alone: a
// gear or a unit can carry the keyword too. Legends are left out, since a
// legend is never played from hand, and so are signature cards: one may only
// be run by a deck whose legend is that champion, so it is not among the
// things a domain can answer with — it is something one deck in that domain
// brought with it.
func generateSpeedCards(speed, deckName, fileName string, opts Options) (SpeedResult, error) {
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

	d := deck{name: deckName, notetype: "Basic"}
	rosters := byDomain(found)
	for _, domain := range slices.Sorted(maps.Keys(rosters)) {
		// Cheapest first, and by name among equals, so a band's list
		// is the one below it with the dearer cards added.
		all := rosters[domain]
		slices.SortStableFunc(all, func(a, b cards.Card) int {
			if c := cmp.Compare(energyCost(a), energyCost(b)); c != 0 {
				return c
			}
			return strings.Compare(a.BaseName(), b.BaseName())
		})
		for energy := 1; energy <= max(maxEnergy(all), 1); energy++ {
			within := atOrUnder(all, energy)
			if len(within) == 0 {
				continue
			}
			d.notes = append(d.notes, note{
				front: fmt.Sprintf("What are all the %s cards in %s that cost %d Energy or less?", speed, domain, energy),
				back:  roster(within, images),
				tags: []string{
					"riftbound::" + strings.ToLower(speed) + "-cards",
					"riftbound::domain::" + strings.ToLower(domain),
					"riftbound::energy::" + strconv.Itoa(energy),
				},
			})
		}
	}

	deckFile := filepath.Join(opts.OutDir, fileName)
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
// set beside it. A card handed out as an organized-play promo is kept as the
// printing from the set it belongs to: Lunar Boon is an Unleashed card that
// was also given away, and reading it as an OPP card says nothing about where
// to find it.
func speedCards(cs []cards.Card, keyword string) []cards.Card {
	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if c.Classification.Type == cards.TypeLegend || c.IsSignature() || !c.HasKeyword(keyword) {
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

// maxEnergy is the dearest Energy cost among the cards.
func maxEnergy(cs []cards.Card) int {
	top := 0
	for _, c := range cs {
		top = max(top, energyCost(c))
	}
	return top
}

// atOrUnder keeps the cards costing threshold Energy or less, in the order
// they were given.
func atOrUnder(cs []cards.Card, threshold int) []cards.Card {
	var out []cards.Card
	for _, c := range cs {
		if energyCost(c) <= threshold {
			out = append(out, c)
		}
	}
	return out
}
