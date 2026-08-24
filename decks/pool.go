package decks

import (
	"regexp"
	"strings"

	"rb/cards"
)

// pool is how many copies of each card the collection holds, keyed by a
// normalised card name. Every printing counts towards the same key —
// alternate arts, overnumbered copies and reprints in a later set are all the
// same card to build a deck with.
type pool struct {
	owned map[string]int
	// squashed indexes the same cards with the spaces taken out as well, so a
	// name that breaks into different words — "Rek-Sai" against "Rek'Sai" —
	// still finds them.
	squashed map[string]int
	// printed is the name the catalog gives each key, for talking about a
	// card in the spelling it is actually sold under.
	printed map[string]string
	// runes marks the keys the catalog files as runes, so a list that keeps
	// them somewhere other than its rune pool still has them passed over.
	runes map[string]bool
}

func newPool(cs []cards.Card, owned map[string]int) *pool {
	p := &pool{
		owned:    make(map[string]int, len(cs)),
		squashed: make(map[string]int, len(cs)),
		printed:  make(map[string]string, len(cs)),
		runes:    map[string]bool{},
	}
	for _, c := range cs {
		// Cards nobody owns are keyed too, so a name the catalog knows can be
		// told apart from one nothing prints.
		k := normalize(c.BaseName())
		p.owned[k] += owned[c.RiftboundID]
		p.squashed[squash(k)] += owned[c.RiftboundID]
		if _, seen := p.printed[k]; !seen {
			p.printed[k] = c.BaseName()
		}
		if c.Classification.Type == "Rune" {
			p.runes[k], p.runes[squash(k)] = true, true
		}
	}
	return p
}

var (
	apostropheRe  = regexp.MustCompile(`['’]`)
	punctuationRe = regexp.MustCompile(`[^a-z0-9]+`)
)

// normalize reduces a card name to the words in it. Decklists don't spell
// names quite the way the catalog does — champions take a comma where the
// catalog prints a dash, "Fizz, Trickster" against "Fizz - Trickster" — and
// punctuation is all that differs, so dropping it lines the two up.
//
// An apostrophe goes rather than becoming a space, so that the "Rek'Sai" the
// catalog prints and the "Reksai" a list types stay one word either way.
func normalize(name string) string {
	name = apostropheRe.ReplaceAllString(strings.ToLower(name), "")
	return strings.TrimSpace(punctuationRe.ReplaceAllString(name, " "))
}

// squash drops the word breaks as well, leaving the letters alone. It is the
// last word on whether two spellings are the same name, since it doesn't care
// where the punctuation fell.
func squash(normalized string) string {
	return strings.ReplaceAll(normalized, " ", "")
}

// have reports how many copies of the named card the collection holds, and
// whether any card is printed under that name at all.
func (p *pool) have(name string) (qty int, known bool) {
	want := normalize(name)
	if n, ok := p.owned[want]; ok {
		return n, true
	}
	if n, ok := p.squashed[squash(want)]; ok {
		return n, true
	}

	// Fall back to the names that end the same way, and add up what they
	// hold: where more than one matches, they are printings of a single card
	// that the catalog names inconsistently.
	for k, held := range p.owned {
		if sharesTail(k, want) {
			qty, known = qty+held, true
		}
	}
	return qty, known
}

// sharesTail reports whether one of the names ends in the other, on a whole
// word and over at least two words.
//
// A legend carries the champion's species in front of it in the catalog —
// "Yordle, Kennen - Heart of the Tempest" — which decklists leave off, so the
// tail is what the two spellings have in common. The two-word floor keeps a
// one-word card like Shadow from answering for every card it ends.
func sharesTail(a, b string) bool {
	long, short := a, b
	if len(short) > len(long) {
		long, short = short, long
	}
	if !strings.Contains(short, " ") {
		return false
	}
	return long == short || strings.HasSuffix(long, " "+short)
}

// isRune reports whether the catalog files the named card as a rune.
func (p *pool) isRune(name string) bool {
	k := normalize(name)
	return p.runes[k] || p.runes[squash(k)]
}

// nearest picks the catalog name closest in spelling to the one asked for, so
// a typo in a pasted list can be told from a card the catalog hasn't caught
// up with. It gives up rather than reach for a name that is nothing like it.
func (p *pool) nearest(name string) (string, bool) {
	want := normalize(name)
	limit := max(len(want)/4, 1)

	best, found := "", false
	for k := range p.owned {
		d := editDistance(want, k)
		if d <= limit {
			best, limit, found = k, d-1, true
		}
	}
	if !found {
		return "", false
	}
	return p.printed[best], true
}

// editDistance counts the single-character edits between two names, over one
// row of the matrix at a time since only the count is wanted.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			sub := prev[j-1]
			if a[i-1] != b[j-1] {
				sub++
			}
			curr[j] = min(sub, min(prev[j]+1, curr[j-1]+1))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
