package decks

import (
	"slices"
	"testing"

	"rb/cards"
)

// printing is a card as the catalog files it. It comes out of Vendetta in
// the Fury domain unless a test cares otherwise, since most don't.
func printing(id, name string) cards.Card {
	c := cards.Card{Name: name, RiftboundID: id}
	c.Set = cards.CardSet{SetID: "VEN", Label: "Vendetta"}
	c.Classification.Domain = []string{"Fury"}
	return c
}

// printedIn refiles a printing under another set, for the cards the catalog
// reprints.
func printedIn(c cards.Card, setID, label string) cards.Card {
	c.Set = cards.CardSet{SetID: setID, Label: label}
	return c
}

// inDomains says what a card costs to play, for the ones that take two.
func inDomains(c cards.Card, ds ...string) cards.Card {
	c.Classification.Domain = ds
	return c
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
		printedIn(printing("ogn-201-298", "Lightning Rush"), "OGN", "Origins"),
		printing("ven-111-166", "Nocturne - Horrifying"),
		printing("ven-155-166", "Yordle, Kennen - Heart of the Tempest"),
		inDomains(printedIn(printing("ven-197-166", "Heart of the Tempest"), "OGN", "Origins"), "Order"),
		printing("ven-042-166", "Shadow"),
		inDomains(printing("ven-160-166", "Rek'sai - Void Burrower"), "Fury", "Order"),
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

// A card is worth buying wherever it is printed, so every set it has appeared
// in is listed, not just the one the first printing came out of.
func TestDetailsGatherEverySetACardIsPrintedIn(t *testing.T) {
	got := testPool(nil).details("Lightning Rush")
	if want := []string{"Vendetta", "Origins"}; !slices.Equal(got.Sets, want) {
		t.Errorf("sets = %v, want %v", got.Sets, want)
	}
	if want := []string{"Fury"}; !slices.Equal(got.Domains, want) {
		t.Errorf("domains = %v, want %v", got.Domains, want)
	}
}

func TestDetailsKeepBothDomainsOfADualCard(t *testing.T) {
	got := testPool(nil).details("Rek'Sai, Void Burrower")
	if want := []string{"Fury", "Order"}; !slices.Equal(got.Domains, want) {
		t.Errorf("domains = %v, want %v", got.Domains, want)
	}
}

// Where a name is only matched on its tail it can answer to more than one
// printing, and the details have to cover all of them.
func TestDetailsMergeThePrintingsThatAnswerForAName(t *testing.T) {
	got := testPool(nil).details("Kennen - Heart of the Tempest")
	if want := []string{"Origins", "Vendetta"}; !slices.Equal(got.Sets, want) {
		t.Errorf("sets = %v, want %v", got.Sets, want)
	}
	if want := []string{"Order", "Fury"}; !slices.Equal(got.Domains, want) {
		t.Errorf("domains = %v, want %v", got.Domains, want)
	}
}

// A name nothing is printed under has nowhere to be bought and no domain, so
// there is nothing to say about it beyond that the catalog doesn't know it.
func TestDetailsOfAnUnknownNameAreEmpty(t *testing.T) {
	got := testPool(nil).details("Lightnign Rush")
	if len(got.Sets) != 0 || len(got.Domains) != 0 {
		t.Errorf("details = %+v, want nothing for a name the catalog doesn't print", got)
	}
}
