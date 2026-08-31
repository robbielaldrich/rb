package collection

import (
	"strings"
	"testing"

	"rb/cards"
)

func sparesFor(t *testing.T, cs []cards.Card, held map[string]int) []spare {
	t.Helper()
	var coll collection
	for id, n := range held {
		coll.Cards = append(coll.Cards, collectedCard{RiftboundID: id, Quantity: n})
	}
	return spares(cs, &coll)
}

// Only what is held past a playset is surplus; the playset itself is not.
func TestSurplusStartsPastThePlayset(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3} {
		if got := sparesFor(t, statsCards(), map[string]int{"ven-002-166": n}); len(got) != 0 {
			t.Errorf("%d copies: got %v, want no surplus", n, got)
		}
	}

	got := sparesFor(t, statsCards(), map[string]int{"ven-002-166": 5})
	if len(got) != 1 {
		t.Fatalf("5 copies: got %d spares, want one", len(got))
	}
	if got[0].copies != 5 || got[0].playset != 3 || got[0].over() != 2 {
		t.Errorf("got %+v, want 5 copies of a 3-card playset with 2 over", got[0])
	}
}

// A deck fields one battlefield or legend, so the second copy is already
// spare — the threshold that makes the count worth reporting per type.
func TestSurplusCountsBattlefieldsAndLegendsAgainstOne(t *testing.T) {
	cs := append(statsCards(),
		typedPrinting("ven-010-166", "Abandoned Hall", "ven", cards.TypeBattlefield),
		typedPrinting("ven-011-166", "Zed - Master of Shadows", "ven", cards.TypeLegend),
	)
	got := sparesFor(t, cs, map[string]int{"ven-010-166": 2, "ven-011-166": 1})

	if len(got) != 1 {
		t.Fatalf("got %d spares, want only the battlefield", len(got))
	}
	if got[0].name != "Abandoned Hall" || got[0].playset != 1 || got[0].over() != 1 {
		t.Errorf("got %+v, want Abandoned Hall 1 over a playset of 1", got[0])
	}
}

// Copies are summed over printings, so a spare can be made of mixed art.
func TestSurplusCountsAcrossPrintings(t *testing.T) {
	got := sparesFor(t, statsCards(), map[string]int{
		"ven-001-166":  3,
		"ven-001a-166": 1,
	})
	if len(got) != 1 {
		t.Fatalf("got %d spares, want one", len(got))
	}
	if got[0].copies != 4 || got[0].over() != 1 {
		t.Errorf("got %+v, want 4 copies with 1 over", got[0])
	}
}

// The deepest surplus leads, since that is what a trade binder is built from.
func TestSurplusOrdersByDepth(t *testing.T) {
	got := sparesFor(t, statsCards(), map[string]int{
		"ven-002-166": 4, // 1 over
		"ven-003-166": 9, // 6 over
	})
	if len(got) != 2 {
		t.Fatalf("got %d spares, want two", len(got))
	}
	if got[0].name != "Mesmerize" || got[1].name != "Oasis Raider" {
		t.Errorf("got %s then %s, want the deeper surplus first", got[0].name, got[1].name)
	}
}

// An empty report says so rather than printing a bare header.
func TestSurplusReportSaysWhenNothingIsSpare(t *testing.T) {
	var b strings.Builder
	writeSurplus(&b, nil)
	if !strings.Contains(b.String(), "no card is held past a playset") {
		t.Errorf("got %q, want the empty-report line", b.String())
	}
}
