package ankigen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"rb/cards"
)

// SignatureResult reports what GenerateSignatureCards produced.
type SignatureResult struct {
	DeckFile string
	MediaDir string
	Notes    int
}

// GenerateSignatureCards builds a deck for learning which card each legend
// brings with it.
//
// A signature card is the one thing in the game that isn't generally
// available: only a deck whose legend is that champion may run it, which is
// why the speed decks leave them out. That makes them worth knowing from the
// other end — an opponent names their legend before a card is played, and
// what they have unlocked by doing so is knowable from that alone.
//
// The tie is to the champion rather than to one printing of their legend, so
// a champion with two legends shares their signature cards between both:
// either Master Yi can run Alpha Strike and Highlander. Ornn brings three
// gears rather than a card, which is why the question asks for a card and the
// answer counts them.
func GenerateSignatureCards(opts Options) (SignatureResult, error) {
	cs, err := loadLegal(opts.CatalogPath)
	if err != nil {
		return SignatureResult{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	legends := firstPrintedLegends(cs)
	if len(legends) == 0 {
		return SignatureResult{}, fmt.Errorf("no legends in %s", opts.CatalogPath)
	}
	signatures := signatureCards(cs)

	// Only the cards actually named by a note need a scan.
	var shown []cards.Card
	byLegend := map[string][]cards.Card{}
	for _, l := range legends {
		sigs := signaturesOf(signatures, l)
		if len(sigs) == 0 {
			continue
		}
		byLegend[l.BaseName()] = sigs
		shown = append(shown, sigs...)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return SignatureResult{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}
	images, err := renderScans(shown, mediaDir, opts)
	if err != nil {
		return SignatureResult{}, err
	}

	d := deck{name: opts.SignatureDeckName, notetype: "Basic"}
	for _, l := range legends {
		sigs, ok := byLegend[l.BaseName()]
		if !ok {
			continue
		}
		d.notes = append(d.notes, note{
			front: fmt.Sprintf("What is the signature card of %s?", l.BaseName()),
			back:  roster(sigs, images),
			tags: []string{
				"riftbound::signature-cards",
				"riftbound::set::" + strings.ToUpper(l.Set.SetID),
			},
		})
	}

	deckFile := filepath.Join(opts.OutDir, "riftbound-signature-cards.txt")
	if err := d.write(deckFile); err != nil {
		return SignatureResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return SignatureResult{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes)}, nil
}

// firstPrintedLegends collects the legends a set introduces, in name order,
// one entry per legend.
//
// A legend is taken as first printed where its signature card follows it at
// the next collector number, which is how the sets lay the two out. That is
// doing more than it looks: the promo and overnumbered reprints of a legend
// are printed away from their signature card, and at least one of them —
// Vendetta's overnumbered Jayce — reaches the catalog with both its champion
// prefix and its overnumbered flag missing, so neither the name nor the
// metadata tells it from a first printing. Where it sits does.
func firstPrintedLegends(cs []cards.Card) []cards.Card {
	next := map[printedAt]cards.Card{}
	for _, c := range cs {
		if !c.IsChasePrinting() {
			next[printedAt{strings.ToUpper(c.Set.SetID), c.CollectorNumber}] = c
		}
	}

	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if c.Classification.Type != cards.TypeLegend || c.IsChasePrinting() {
			continue
		}
		after, ok := next[printedAt{strings.ToUpper(c.Set.SetID), c.CollectorNumber + 1}]
		if !ok || !after.IsSignature() || !sharesChampion(after, c) {
			continue
		}

		i, seen := at[c.BaseName()]
		if !seen {
			at[c.BaseName()] = len(out)
			out = append(out, c)
			continue
		}
		if out[i].IsPromo() && !c.IsPromo() {
			out[i] = c
		}
	}

	slices.SortFunc(out, func(a, b cards.Card) int {
		return strings.Compare(a.BaseName(), b.BaseName())
	})
	return out
}

// printedAt is where a card sits in its set, which is what pairs a legend with
// the signature card printed after it.
type printedAt struct {
	setID  string
	number int
}

// signatureCards collects every signature card, one entry per card, keeping
// the printing from the set it belongs to over a promo of it.
func signatureCards(cs []cards.Card) []cards.Card {
	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if !c.IsSignature() {
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

// signaturesOf picks the signature cards a legend unlocks, in name order.
func signaturesOf(signatures []cards.Card, legend cards.Card) []cards.Card {
	var out []cards.Card
	for _, s := range signatures {
		if sharesChampion(s, legend) {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b cards.Card) int {
		return strings.Compare(a.BaseName(), b.BaseName())
	})
	return out
}

// sharesChampion reports whether two cards name a champion in common. Tags
// carry regions and species as well — Kennen's legend is tagged Yordle — so
// this is looser than it sounds, but a signature card and a legend share a tag
// only where they share the champion.
func sharesChampion(a, b cards.Card) bool {
	for _, t := range a.Tags {
		if slices.Contains(b.Tags, t) {
			return true
		}
	}
	return false
}
