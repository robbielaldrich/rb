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
		"set            cards    alt art  overnumbered  signature",
		"VEN  Vendetta  1/3 33%  0/1 0%   0/1 0%        0/1 0%",
		"all            1/3 33%  0/1 0%   0/1 0%        0/1 0%",
		"",
	}, "\n")
	if out.String() != want {
		t.Errorf("stats table:\n%s\nwant:\n%s", out.String(), want)
	}
}
