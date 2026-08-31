package collection

import (
	"strings"
	"testing"

	"rb/cards"
)

func wantsFor(t *testing.T, cs []cards.Card, held map[string]int) []want {
	t.Helper()
	var coll collection
	for id, n := range held {
		coll.Cards = append(coll.Cards, collectedCard{RiftboundID: id, Quantity: n})
	}
	return wants(cs, &coll)
}

func wantFor(t *testing.T, out []want, name string) want {
	t.Helper()
	for _, w := range out {
		if w.name == name {
			return w
		}
	}
	t.Fatalf("%q is not wanted, got %v", name, out)
	return want{}
}

// A card is wanted until the playset is full, and by however many copies it
// is still short.
func TestMissingCountsUpToThePlayset(t *testing.T) {
	for n, short := range map[int]int{0: 3, 1: 2, 2: 1} {
		got := wantFor(t, wantsFor(t, statsCards(), map[string]int{"ven-002-166": n}), "Oasis Raider")
		if got.copies != n || got.short() != short {
			t.Errorf("%d copies: got %+v, want %d short", n, got, short)
		}
	}

	for _, n := range []int{3, 4} {
		for _, w := range wantsFor(t, statsCards(), map[string]int{"ven-002-166": n}) {
			if w.name == "Oasis Raider" {
				t.Errorf("%d copies: got %+v, want a full playset to go unwanted", n, w)
			}
		}
	}
}

// Copies are summed over printings, so an alternate art fills the playset the
// plain printing started.
func TestMissingCountsAcrossPrintings(t *testing.T) {
	held := map[string]int{"ven-001-166": 2, "ven-001a-166": 1}
	for _, w := range wantsFor(t, statsCards(), held) {
		if w.name == "Astral Heron" {
			t.Errorf("got %+v, want three copies across two printings to fill the playset", w)
		}
	}
}

// A card known only by its chase printings is not something to go looking for,
// so nothing asks for it at a number the set never printed it at.
func TestMissingSkipsChaseOnlyPrintings(t *testing.T) {
	for _, w := range wantsFor(t, statsCards(), nil) {
		if strings.HasPrefix(w.name, "Ahri - Sly") {
			t.Errorf("got %+v, want the overnumbered and signature printings left out", w)
		}
	}
}

// A deck fields one battlefield or legend, so one copy is the whole want.
func TestMissingWantsOneBattlefieldOrLegend(t *testing.T) {
	cs := append(statsCards(),
		typedPrinting("ven-010-166", "Abandoned Hall", "ven", cards.TypeBattlefield),
		typedPrinting("ven-011-166", "Zed - Master of Shadows", "ven", cards.TypeLegend),
	)
	got := wantsFor(t, cs, map[string]int{"ven-011-166": 1})

	hall := wantFor(t, got, "Abandoned Hall")
	if hall.playset != 1 || hall.short() != 1 {
		t.Errorf("got %+v, want one copy of the battlefield wanted", hall)
	}
	for _, w := range got {
		if w.name == "Zed - Master of Shadows" {
			t.Errorf("got %+v, want the one legend held to be enough", w)
		}
	}
}

// The report reads like a binder page: main run first in printed order, then
// the series that number themselves apart from it.
func TestMissingSortsInBinderOrder(t *testing.T) {
	cs := []cards.Card{
		printing("ven-012-166", "Twelve", "ven", cards.Metadata{}),
		printing("ven-r01", "Fury Rune", "ven", cards.Metadata{}),
		printing("ven-002-166", "Two", "ven", cards.Metadata{}),
	}
	var names []string
	for _, w := range wantsFor(t, cs, nil) {
		names = append(names, w.name)
	}
	if got, want := strings.Join(names, " "), "Two Twelve Fury Rune"; got != want {
		t.Errorf("order = %q, want %q", got, want)
	}
}

// rarityPrinting is a printing pulled at a given rarity, the Showcase ones
// being the reprints that share a card's name.
func rarityPrinting(id, name, setID, rarity string, flags cards.Metadata) cards.Card {
	c := printing(id, name, setID, flags)
	c.Classification.Rarity = rarity
	return c
}

// A card is wanted at the rarity the set prints it at, not the Showcase its
// alternate art wears, whichever printing the catalog lists first.
func TestMissingShowsThePrintedRarity(t *testing.T) {
	plain := rarityPrinting("ven-020-166", "Sand Soldier", "ven", "Rare", cards.Metadata{})
	alt := rarityPrinting("ven-020a-166", "Sand Soldier (Alternate Art)", "ven", "Showcase", cards.Metadata{AlternateArt: true})

	for _, cs := range [][]cards.Card{{plain, alt}, {alt, plain}} {
		got := wantFor(t, wantsFor(t, cs, nil), "Sand Soldier")
		if got.rarity != "Rare" {
			t.Errorf("rarity = %q, want Rare", got.rarity)
		}
	}
}
