package collection

import (
	"fmt"
	"testing"
	"time"
)

func loadIndex(t *testing.T) *cardIndex {
	t.Helper()
	cs, err := loadCatalog("../cards/cards.json")
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
			lines = append(lines, fmt.Sprintf("%s [d=%d num=%v]", cardLabel(r.card), r.dist, r.numMatch))
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
	t.Logf("before: %s", cardLabel(base[0].card))
	second := base[1].card

	coll := &collection{}
	coll.set(second, 3, time.Now())
	after := ix.search("rune", coll, 5)
	t.Logf("after touching %s: top is %s", cardLabel(second), cardLabel(after[0].card))
	if after[0].card.RiftboundID != second.RiftboundID {
		t.Errorf("recency did not promote %s", cardLabel(second))
	}
}
