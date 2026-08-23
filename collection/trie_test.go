package collection

import (
	"fmt"
	"testing"

	"rb/cards"
	"time"
)

func loadIndex(t *testing.T) *cardIndex {
	t.Helper()
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}
	t.Logf("catalog: %d unique cards", len(cs))
	return buildIndex(cs)
}

func TestSearchQueries(t *testing.T) {
	ix := loadIndex(t)
	coll := &collection{}
	for _, q := range []string{"astral 44", "astral", "astrel heron", "heron", "44 ven", "sona harm", "annie", "fury rune", "zzzzqqq"} {
		res := ix.search(q, coll, 5)
		var lines []string
		for _, r := range res {
			lines = append(lines, fmt.Sprintf("%s [d=%d num=%v]", r.card.Label(), r.dist, r.numMatch))
		}
		t.Logf("%-14q -> %v", q, lines)
	}
}

func TestRecencyBreaksTies(t *testing.T) {
	ix := loadIndex(t)
	base := ix.search("rune", &collection{}, 5)
	if len(base) < 2 {
		t.Fatalf("need >=2 results, got %d", len(base))
	}
	t.Logf("before: %s", base[0].card.Label())
	second := base[1].card

	coll := &collection{}
	coll.set(second, 3, time.Now())
	after := ix.search("rune", coll, 5)
	t.Logf("after touching %s: top is %s", second.Label(), after[0].card.Label())
	if after[0].card.RiftboundID != second.RiftboundID {
		t.Errorf("recency did not promote %s", second.Label())
	}
}

// Punctuation inside a name is a separator to the index, so it has to be one
// to the query too: "star-crossed" and "ol' poro" are typed as they are read
// off the card.
func TestPunctuationInQuery(t *testing.T) {
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}
	ix := buildIndex(cs)

	for _, tc := range []struct{ query, want string }{
		{"star-crossed", "Star-Crossed"},
		{"star crossed", "Star-Crossed"},
		{"ol' poro", "Ol' Poro"},
		{"akali - rogue assassin", "Akali - Rogue Assassin"},
	} {
		got := ix.search(tc.query, &collection{}, maxMatches)
		if len(got) == 0 {
			t.Errorf("%q found nothing, want %q", tc.query, tc.want)
			continue
		}
		if got[0].card.Name != tc.want {
			t.Errorf("%q found %q, want %q", tc.query, got[0].card.Name, tc.want)
		}
	}
}
