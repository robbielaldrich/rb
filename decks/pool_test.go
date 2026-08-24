package decks

import (
	"testing"

	"rb/cards"
)

func printing(id, name string) cards.Card {
	return cards.Card{Name: name, RiftboundID: id}
}

func runePrinting(id, name string) cards.Card {
	c := printing(id, name)
	c.Classification.Type = "Rune"
	return c
}

// testPool holds the naming quirks the catalog really has: one card printed
// under several ids, champions written with a dash where lists write a comma,
// and a legend that carries the champion's species in front of it.
func testPool(owned map[string]int) *pool {
	return newPool([]cards.Card{
		printing("ven-156-166", "Lightning Rush"),
		printing("ven-156a-166", "Lightning Rush (Alternate Art)"),
		printing("ogn-201-298", "Lightning Rush"),
		printing("ven-111-166", "Nocturne - Horrifying"),
		printing("ven-155-166", "Yordle, Kennen - Heart of the Tempest"),
		printing("ven-197-166", "Heart of the Tempest"),
		printing("ven-042-166", "Shadow"),
		printing("ven-160-166", "Rek'sai - Void Burrower"),
		runePrinting("ven-r05", "Chaos Rune"),
		printing("ven-043-166", "Shadow Order Disciple"),
	}, owned)
}

func TestEveryPrintingCountsTowardsTheSameSlot(t *testing.T) {
	p := testPool(map[string]int{"ven-156-166": 1, "ven-156a-166": 2, "ogn-201-298": 1})
	if got, known := p.have("Lightning Rush"); !known || got != 4 {
		t.Errorf("have(Lightning Rush) = %d, %v, want 4 copies across the printings", got, known)
	}
}

func TestPunctuationDoesntHideACard(t *testing.T) {
	p := testPool(map[string]int{"ven-111-166": 3})
	if got, known := p.have("Nocturne, Horrifying"); !known || got != 3 {
		t.Errorf("have(Nocturne, Horrifying) = %d, %v, want the 3 copies of Nocturne - Horrifying", got, known)
	}
}

// The catalog files one legend under two names; a list that writes a third
// still has to find both printings.
func TestLegendFoundWithoutItsSpecies(t *testing.T) {
	p := testPool(map[string]int{"ven-155-166": 1, "ven-197-166": 1})
	if got, known := p.have("Kennen, Heart of the Tempest"); !known || got != 2 {
		t.Errorf("have(Kennen, Heart of the Tempest) = %d, %v, want both printings", got, known)
	}
}

func TestOneWordNameDoesntBorrowFromLongerOnes(t *testing.T) {
	p := testPool(map[string]int{"ven-042-166": 1, "ven-043-166": 3})
	if got, known := p.have("Shadow"); !known || got != 1 {
		t.Errorf("have(Shadow) = %d, %v, want the one Shadow alone", got, known)
	}
	if got, known := p.have("Disciple"); known || got != 0 {
		t.Errorf("have(Disciple) = %d, %v, want nothing: no card is printed under it", got, known)
	}
}

// The catalog prints an apostrophe where a list may type nothing or a dash,
// and neither should hide the card.
func TestApostrophesAndDashesDontHideACard(t *testing.T) {
	p := testPool(map[string]int{"ven-160-166": 2})
	for _, name := range []string{
		"Rek'sai - Void Burrower",
		"Rek'Sai, Void Burrower",
		"Reksai, Void Burrower",
		"Rek-Sai, Void Burrower",
		"Rek Sai, Void Burrower",
	} {
		if got, known := p.have(name); !known || got != 2 {
			t.Errorf("have(%q) = %d, %v, want the 2 copies of Rek'sai", name, got, known)
		}
	}
}

func TestRunesAreKnownAsSuch(t *testing.T) {
	p := testPool(nil)
	if !p.isRune("Chaos Rune") {
		t.Error("isRune(Chaos Rune) = false, want the catalog's own classification")
	}
	if p.isRune("Shadow") {
		t.Error("isRune(Shadow) = true, want a card to stay a card")
	}
}

func TestNearestNamesAMisspelling(t *testing.T) {
	p := testPool(nil)
	if got, ok := p.nearest("Lightnign Rush"); !ok || got != "Lightning Rush" {
		t.Errorf("nearest(Lightnign Rush) = %q, %v, want Lightning Rush", got, ok)
	}
	// A card the catalog simply doesn't carry has no near neighbour worth
	// putting to the user.
	if got, ok := p.nearest("Summon Tibbers"); ok {
		t.Errorf("nearest(Summon Tibbers) = %q, want nothing close enough", got)
	}
}

func TestUnknownNameIsToldFromAnUnownedCard(t *testing.T) {
	p := testPool(nil)
	if got, known := p.have("Lightning Rush"); !known || got != 0 {
		t.Errorf("have(Lightning Rush) = %d, %v, want a card known but unowned", got, known)
	}
	if _, known := p.have("Summon Tibbers"); known {
		t.Error("have(Summon Tibbers) claims the catalog prints it")
	}
}

// The pasted list is measured against the real catalog, where the spellings
// that matter actually live.
func TestPastedNamesResolveAgainstTheCatalog(t *testing.T) {
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}

	p := newPool(cs, nil)
	for _, e := range parse(t, pasted).Copies(true) {
		if _, known := p.have(e.Name); !known {
			t.Errorf("no card in the catalog answers to %q", e.Name)
		}
	}
}
