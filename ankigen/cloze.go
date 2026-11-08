package ankigen

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"rb/cards"
)

// clozeThumbWidth is how wide a cloze note shows the card it blanked out. The
// note answers with one card rather than a row of them, so it can be larger
// than a roster's thumbnails while still leaving the list above in view.
const clozeThumbWidth = 250

// clozeRoster turns one roster into a note per card, each showing the whole
// list with that one card blanked out — an overlapping cloze.
//
// Asking for a roster in one go asks for every card in it at once: the answer
// is half-right more often than right or wrong, and the card that keeps
// slipping is carried by the ones that don't. A note per card lets Anki
// schedule each on its own, while the rest of the list stays in view, so it is
// still the set being learnt and not a run of cards out of context.
//
// A note's identity is its tags, which name the roster and the card in it.
// They make its GUID, so a later run that grows the list rewrites the notes
// already in it in place, and they make the search to-delete.txt lists for a
// note that goes, since a cloze note has no question to search by.
func clozeRoster(heading string, cs []cards.Card, images map[string]string, tags []string) []note {
	out := make([]note, 0, len(cs))
	for i, c := range cs {
		var b strings.Builder
		b.WriteString(heading)
		b.WriteString("<ul>")
		for j, other := range cs {
			item := fmt.Sprintf("%s <i>(%s)</i>", other.BaseName(), strings.ToUpper(other.Set.SetID))
			if i == j {
				// The set goes inside the blank, being as much a clue to the
				// card as its first letter.
				item = "{{c1::" + item + "}}"
			}
			fmt.Fprintf(&b, "<li>%s</li>", item)
		}
		b.WriteString("</ul>")

		noteTags := append(slices.Clone(tags), "riftbound::card::"+slug(c.BaseName()))
		out = append(out, note{
			front: b.String(),
			back:  thumb(images[c.RiftboundID], clozeThumbWidth),
			tags:  noteTags,
			guid:  guidFor(noteTags),
		})
	}
	return out
}

// guidFor derives a note's GUID from the tags that identify it, so the same
// card in the same roster comes out the same on every run.
func guidFor(tags []string) string {
	sum := sha256.Sum256([]byte(strings.Join(tags, " ")))
	return fmt.Sprintf("rb%x", sum[:8])
}

// slug renders a card name as a tag: lower case, with every run of anything
// but letters and digits made a single hyphen.
func slug(s string) string {
	var b strings.Builder
	gap := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if gap && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			gap = false
		} else {
			gap = true
		}
	}
	return b.String()
}
