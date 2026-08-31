package collection

import (
	"strings"
	"testing"

	"rb/cards"
)

func printing(id, name, setID string, flags cards.Metadata) cards.Card {
	return cards.Card{
		Name:        name,
		RiftboundID: id,
		Set:         cards.CardSet{SetID: setID, Label: "Vendetta"},
		Metadata:    flags,
	}
}

func statsCards() []cards.Card {
	var (
		plain = cards.Metadata{}
		alt   = cards.Metadata{AlternateArt: true}
		over  = cards.Metadata{Overnumbered: true}
		sig   = cards.Metadata{Signature: true}
	)
	return []cards.Card{
		printing("ven-001-166", "Astral Heron", "ven", plain),
		printing("ven-001a-166", "Astral Heron (Alternate Art)", "ven", alt),
		printing("ven-002-166", "Oasis Raider", "ven", plain),
		printing("ven-003-166", "Mesmerize", "ven", plain),
		printing("ven-200-166", "Ahri - Sly (Overnumbered)", "ven", over),
		printing("ven-201*-166", "Ahri - Sly (Signature)", "ven", sig),
	}
}

func statsFor(t *testing.T, owned ...string) setStats {
	t.Helper()
	var coll collection
	for _, id := range owned {
		coll.Cards = append(coll.Cards, collectedCard{RiftboundID: id, Quantity: 1})
	}
	got := summarise(statsCards(), &coll)
	if len(got) != 1 {
		t.Fatalf("summarise returned %d sets, want one", len(got))
	}
	return got[0]
}

// A named card counts once however many printings it has, and the extras are
// not part of the named total.
func TestNamedCardsCountOnePerName(t *testing.T) {
	if got := statsFor(t).named; got != (tally{owned: 0, total: 3}) {
		t.Errorf("named = %v, want 0/3", got)
	}
}

// Owning any printing completes the named card, which is the whole point of
// the headline number: an alternate art is the same card to play with.
func TestAlternateArtCompletesTheNamedCard(t *testing.T) {
	s := statsFor(t, "ven-001a-166")
	if s.named != (tally{owned: 1, total: 3}) {
		t.Errorf("named = %v, want 1/3", s.named)
	}
	if s.alt != (tally{owned: 1, total: 1}) {
		t.Errorf("alt = %v, want 1/1", s.alt)
	}
}

// Overnumbered and signature printings are their own chase, not names the
// set is expected to yield.
func TestExtrasAreCountedSeparately(t *testing.T) {
	s := statsFor(t, "ven-200-166", "ven-201*-166")
	if s.named != (tally{owned: 0, total: 3}) {
		t.Errorf("named = %v, want 0/3", s.named)
	}
	if s.over != (tally{owned: 1, total: 1}) {
		t.Errorf("over = %v, want 1/1", s.over)
	}
	if s.sig != (tally{owned: 1, total: 1}) {
		t.Errorf("sig = %v, want 1/1", s.sig)
	}
}

func TestZeroQuantityIsNotOwned(t *testing.T) {
	var coll collection
	coll.Cards = append(coll.Cards, collectedCard{RiftboundID: "ven-001-166", Quantity: 0})
	if got := summarise(statsCards(), &coll)[0].named; got.owned != 0 {
		t.Errorf("named = %v, want nothing owned", got)
	}
}

func TestEmptyTallyPrintsADash(t *testing.T) {
	if got := (tally{}).String(); got != "-" {
		t.Errorf("empty tally = %q, want a dash", got)
	}
}

func TestWriteStatsIsAligned(t *testing.T) {
	var out strings.Builder
	writeStats(&out, summarise(statsCards(), &collection{
		Cards: []collectedCard{{RiftboundID: "ven-001-166", Quantity: 2}},
	}))

	want := strings.Join([]string{
		"set            cards    playsets  alt art  overnumbered  signature",
		"VEN  Vendetta  1/3 33%  0/3 0%    0/1 0%   0/1 0%        0/1 0%",
		"all            1/3 33%  0/3 0%    0/1 0%   0/1 0%        0/1 0%",
		"",
	}, "\n")
	if out.String() != want {
		t.Errorf("stats table:\n%s\nwant:\n%s", out.String(), want)
	}
}

// A dozen alternate arts are missing the API's flag, so the "a" in the
// riftbound_id has to be enough on its own.
func TestUnflaggedAlternateArtStillCounts(t *testing.T) {
	cs := []cards.Card{
		printing("ven-113-166", "Kennen, Storm of Shuriken", "ven", cards.Metadata{}),
		printing("ven-113a-166", "Kennen, Storm of Shuriken", "ven", cards.Metadata{}),
	}
	s := summarise(cs, &collection{
		Cards: []collectedCard{{RiftboundID: "ven-113a-166", Quantity: 1}},
	})[0]

	if s.alt != (tally{owned: 1, total: 1}) {
		t.Errorf("alt = %v, want 1/1", s.alt)
	}
	if s.named != (tally{owned: 1, total: 1}) {
		t.Errorf("named = %v, want the alternate art to complete the one card", s.named)
	}
}

// typedPrinting is a plain printing of a card whose type decides its playset,
// which the ordinary helper leaves empty for the repeatable majority.
func typedPrinting(id, name, setID, cardType string) cards.Card {
	c := printing(id, name, setID, cards.Metadata{})
	c.Classification.Type = cardType
	return c
}

// playsetFor summarises a collection held in the quantities given, keyed by
// riftbound_id, rather than the single copies the other tests assume.
func playsetFor(t *testing.T, cs []cards.Card, held map[string]int) setStats {
	t.Helper()
	var coll collection
	for id, n := range held {
		coll.Cards = append(coll.Cards, collectedCard{RiftboundID: id, Quantity: n})
	}
	got := summarise(cs, &coll)
	if len(got) != 1 {
		t.Fatalf("summarise returned %d sets, want one", len(got))
	}
	return got[0]
}

// A playset is three copies, so anything short of that leaves the card
// incomplete however many of it the collection holds.
func TestPlaysetNeedsThreeCopies(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		got := playsetFor(t, statsCards(), map[string]int{"ven-002-166": n}).playset
		if got != (tally{owned: 0, total: 3}) {
			t.Errorf("%d copies: playset = %v, want 0/3", n, got)
		}
	}
	got := playsetFor(t, statsCards(), map[string]int{"ven-002-166": 3}).playset
	if got != (tally{owned: 1, total: 3}) {
		t.Errorf("3 copies: playset = %v, want 1/3", got)
	}
}

// Copies are counted over every printing, since a deck cares how many of the
// card it has to play with and not which art they wear.
func TestPlaysetCountsAcrossPrintings(t *testing.T) {
	got := playsetFor(t, statsCards(), map[string]int{
		"ven-001-166":  2, // plain
		"ven-001a-166": 1, // alternate art of the same card
	}).playset
	if got != (tally{owned: 1, total: 3}) {
		t.Errorf("playset = %v, want 1/3", got)
	}
}

// A deck fields one battlefield or legend rather than three, so neither is
// measured against a playset at all.
func TestPlaysetExcludesBattlefieldsAndLegends(t *testing.T) {
	cs := append(statsCards(),
		typedPrinting("ven-010-166", "Abandoned Hall", "ven", cards.TypeBattlefield),
		typedPrinting("ven-011-166", "Zed - Master of Shadows", "ven", cards.TypeLegend),
	)
	s := playsetFor(t, cs, map[string]int{"ven-010-166": 3, "ven-011-166": 3})

	// Both are still cards the set prints, so the named total grows by two.
	if s.named != (tally{owned: 2, total: 5}) {
		t.Errorf("named = %v, want 2/5", s.named)
	}
	if s.playset != (tally{owned: 0, total: 3}) {
		t.Errorf("playset = %v, want 0/3, leaving both out", s.playset)
	}
}
