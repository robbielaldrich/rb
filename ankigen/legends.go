package ankigen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"rb/cards"
)

// LegendResult reports what GenerateLegendAbilities produced.
type LegendResult struct {
	DeckFile string
	MediaDir string
	Notes    int
}

// legendTitleFraction is how much of a legend's scan to keep, from the top,
// to show its name and art without its ability. Every legend frame puts the
// bottom of the title band at the same height, 76.4–76.6% down on all 58
// legends printed so far, whatever the length of the text below it, so a
// fixed cut is reliable where it wouldn't be for units. It is taken a hair
// under the band so the band's own edge doesn't survive as a sliver.
const legendTitleFraction = 0.768

// GenerateLegendAbilities builds a deck learning each legend's ability in
// both directions: the card, cut off above its rules text, asks for the
// ability; and the ability, with the legend's name blanked out of it, asks
// for the legend.
//
// Both directions are worth having because both come up. Across the table an
// opponent's legend is in view and what it can do has to be recalled; in the
// middle of a game an ability goes off and which legend did it is the
// question. One note of Anki's "Basic (and reversed card)" type gives a card
// for each, scheduled apart.
func GenerateLegendAbilities(opts Options) (LegendResult, error) {
	cs, err := loadLegal(opts.CatalogPath)
	if err != nil {
		return LegendResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	legends := baseLegends(cs)
	if len(legends) == 0 {
		return LegendResult{}, fmt.Errorf("no legends in %s", opts.CatalogPath)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return LegendResult{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}

	d := deck{name: opts.LegendAbilityDeckName, notetype: "Basic (and reversed card)"}
	for _, l := range legends {
		name, err := renderLegendTitle(l, mediaDir, opts)
		if err != nil {
			return LegendResult{}, err
		}
		d.notes = append(d.notes, note{
			front: img(name) + "<br>Legend ability",
			back:  "What legend:<br>" + abilityHTML(l),
			tags: []string{
				"riftbound::legend-abilities",
				"riftbound::set::" + strings.ToUpper(l.Set.SetID),
			},
		})
	}

	deckFile := filepath.Join(opts.OutDir, DeckFile)
	if err := d.write(deckFile); err != nil {
		return LegendResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return LegendResult{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes)}, nil
}

// baseLegends collects one plain printing of each legend, in name order.
// Overnumbered, signature and alternate-art printings frame the card
// differently, and the title band moves with it, so the crop is only right
// for the plain one; promos are passed over for the set's own printing.
func baseLegends(cs []cards.Card) []cards.Card {
	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if c.Classification.Type != cards.TypeLegend || c.IsChasePrinting() {
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
	// A legend handed out only as a promo would be framed as one, so it is
	// left out rather than cropped in the wrong place.
	out = slices.DeleteFunc(out, cards.Card.IsPromo)
	slices.SortFunc(out, func(a, b cards.Card) int {
		return strings.Compare(a.BaseName(), b.BaseName())
	})
	return out
}

// renderLegendTitle writes the legend's scan cut off under its title band.
func renderLegendTitle(c cards.Card, mediaDir string, opts Options) (string, error) {
	scan, err := cardScan(c, opts)
	if err != nil {
		return "", err
	}
	name := mediaName(c, "-title")
	if err := writeJPEG(filepath.Join(mediaDir, name), cropTop(scan, legendTitleFraction)); err != nil {
		return "", fmt.Errorf("failed to write the cropped image of %s: %w", c.Label(), err)
	}
	return name, nil
}

var (
	paragraphRe = regexp.MustCompile(`</p>\s*<p>`)
	energyRe    = regexp.MustCompile(`:rb_energy_(\d+):`)
	iconRe      = regexp.MustCompile(`:rb_([a-z0-9_]+):`)
	keywordRe   = regexp.MustCompile(`\[([A-Za-z][A-Za-z0-9 ]*)\]`)
)

// iconNames spell out the symbols the rules text uses in place of words.
// Anything not listed falls back to its own name, so a new symbol reads as
// something rather than as a raw token.
var iconNames = map[string]string{
	"exhaust":      "[Exhaust]",
	"might":        "Might",
	"rune_rainbow": "[Any Rune]",
}

// abilityHTML renders a legend's rules text for the back of its note, with
// the legend's own name blanked wherever the text uses it — Teemo's ability
// fetches "a Teemo unit", which would give the answer away.
func abilityHTML(c cards.Card) string {
	s := strings.TrimSpace(c.Text.Rich)
	s = paragraphRe.ReplaceAllString(s, "<br>")
	s = strings.TrimPrefix(strings.TrimSuffix(s, "</p>"), "<p>")
	// [>] is the arrow printed between a keyword and the ability it marks,
	// which reads as noise once the keyword is bold.
	s = strings.NewReplacer("<br />", "<br>", "<br/>", "<br>", "[&gt;]", " ").Replace(s)

	// Keywords are bolded before the icons are spelt out, since those are
	// bracketed too and aren't keywords.
	s = keywordRe.ReplaceAllString(s, "<b>[$1]</b>")
	s = energyRe.ReplaceAllStringFunc(s, func(m string) string {
		n, _ := strconv.Atoi(energyRe.FindStringSubmatch(m)[1])
		if n >= 1 && n <= 20 {
			return string(rune('①' + n - 1))
		}
		return "(" + strconv.Itoa(n) + ")"
	})
	s = iconRe.ReplaceAllStringFunc(s, func(m string) string {
		key := iconRe.FindStringSubmatch(m)[1]
		if name, ok := iconNames[key]; ok {
			return name
		}
		return "[" + strings.ReplaceAll(key, "_", " ") + "]"
	})

	for _, n := range legendNames(c) {
		s = regexp.MustCompile(`\b`+regexp.QuoteMeta(n)+`\b`).ReplaceAllString(s, "____")
	}
	return s
}

// legendNames are the words that would name the legend outright: the
// champion, and the title after it.
func legendNames(c cards.Card) []string {
	champion, title, _ := strings.Cut(c.BaseName(), " - ")
	var out []string
	for _, n := range []string{champion, title} {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}
