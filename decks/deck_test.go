package decks

import (
	"os"
	"testing"
)

func parse(t *testing.T, text string) Deck {
	t.Helper()
	d, err := Parse(text)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

// The same cards pasted in a different order, under a different name, are the
// same deck.
func TestDuplicateIgnoresOrderAndName(t *testing.T) {
	var r registry
	first := parse(t, pasted)
	first.Name = "Worlds winner"
	r.Decks = append(r.Decks, first)

	shuffled := parse(t, `Sideboard:
2 Ravenbloom Prefect
1 Hard Bargain

Rune Pool:
8 Chaos Rune

Champion:
1 Kennen, Storm of Shuriken

Legend:
1 Kennen, Heart of the Tempest

Battlefields:
1 Minefield

MainDeck:
1 Baron Nashor
2 Star-Crossed
3 Lightning Rush
`)
	have, ok := r.duplicate(shuffled)
	if !ok {
		t.Fatal("the same cards in another order weren't recognised as registered")
	}
	if have.Name != "Worlds winner" {
		t.Errorf("duplicate names %q, want the deck already on file", have.Name)
	}
}

func TestDuplicateSeesASideboardChange(t *testing.T) {
	var r registry
	r.Decks = append(r.Decks, parse(t, pasted))

	swapped := parse(t, pasted+"1 Atakhan\n")
	if _, ok := r.duplicate(swapped); ok {
		t.Error("a deck with an extra sideboard card was taken for one already registered")
	}
}

// A card in both the main deck and the sideboard needs both sets of copies.
func TestCopiesSumsTheSectionsThatCount(t *testing.T) {
	d := parse(t, "MainDeck:\n1 Up from the Deep\n2 Flash\n\nSideboard:\n1 Up from the Deep\n")

	if got := d.Copies(false); len(got) != 2 || got[1].Name != "Up from the Deep" || got[1].Quantity != 1 {
		t.Errorf("Copies(false) = %+v, want the main deck alone, in name order", got)
	}
	if got := d.Copies(true); len(got) != 2 || got[1].Quantity != 2 {
		t.Errorf("Copies(true) = %+v, want both copies of Up from the Deep", got)
	}
}

func TestRegistryRoundTrips(t *testing.T) {
	path := t.TempDir() + "/decks.json"

	empty, err := loadRegistry(path)
	if err != nil {
		t.Fatalf("loadRegistry of a file that isn't there yet: %v", err)
	}
	if len(empty.Decks) != 0 {
		t.Fatalf("loadRegistry returned %d decks for a missing file", len(empty.Decks))
	}

	empty.Decks = append(empty.Decks, parse(t, pasted))
	if err := empty.save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("save left its temporary file behind")
	}

	back, err := loadRegistry(path)
	if err != nil {
		t.Fatalf("loadRegistry: %v", err)
	}
	if len(back.Decks) != 1 || back.Decks[0].fingerprint() != empty.Decks[0].fingerprint() {
		t.Errorf("the register came back as %+v", back.Decks)
	}
}
