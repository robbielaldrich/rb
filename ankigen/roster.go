package ankigen

import (
	"fmt"
	"slices"
	"strings"

	"rb/cards"
)

// rosterThumbWidth is how wide the card images are shown on the back of a
// roster note. A domain can hold a dozen cards or more, so they are sized to
// sit several to a row and be recognised rather than read — the names above
// them carry the detail, and the image only has to bring the card to mind.
const rosterThumbWidth = 180

// loadLegal reads the catalog without its banned cards. Every deck here is
// about what an opponent can be holding, and a banned card is one they can't.
func loadLegal(path string) ([]cards.Card, error) {
	cs, err := cards.Load(path)
	if err != nil {
		return nil, err
	}
	return cards.Legal(cs), nil
}

// byDomain groups cards by the domains they belong to, each list in name order
// so that a regenerated deck doesn't reshuffle a list the reader has half
// learnt. A card of two domains is listed under both, being playable out of
// either.
func byDomain(cs []cards.Card) map[string][]cards.Card {
	out := map[string][]cards.Card{}
	for _, c := range cs {
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

// renderScans writes the intact scan of each card and reports what each was
// filed under, keyed by riftbound_id. A card belonging to two domains appears
// on two rosters but is one scan, so each is written once.
func renderScans(cs []cards.Card, mediaDir string, opts Options) (map[string]string, error) {
	images := map[string]string{}
	for _, c := range cs {
		if images[c.RiftboundID] != "" {
			continue
		}
		name, err := renderFull(c, mediaDir, opts)
		if err != nil {
			return nil, err
		}
		images[c.RiftboundID] = name
	}
	return images, nil
}

// roster renders a note's back: how many cards the answer holds, the names
// with the set each was printed in, then the scans.
//
// The count leads because it is what makes the answer checkable — recalling
// nine of eleven cards feels like knowing the list until the number says
// otherwise. The set follows each name since a roster is only ever the roster
// of the format being played.
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
