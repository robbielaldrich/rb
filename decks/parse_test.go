package decks

import (
	"strings"
	"testing"
)

// pasted is a decklist in the shape the builders export, blank lines and all.
const pasted = `Legend:
1 Kennen, Heart of the Tempest

Champion:
1 Kennen, Storm of Shuriken

MainDeck:
3 Lightning Rush
2 Star-Crossed
1 Baron Nashor

Battlefields:
1 Minefield

Rune Pool:
8 Chaos Rune

Sideboard:
1 Hard Bargain
2 Ravenbloom Prefect
`

func TestParseKeepsTheSectionsAsPasted(t *testing.T) {
	d, err := Parse(pasted)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := []string{"Legend", "Champion", "MainDeck", "Battlefields", "Rune Pool", "Sideboard"}
	var got []string
	for _, s := range d.Sections {
		got = append(got, s.Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("sections = %v, want %v", got, want)
	}

	if d.Name != "Kennen, Heart of the Tempest" {
		t.Errorf("Name = %q, want the legend", d.Name)
	}
	if n := d.Size(false); n != 17 {
		t.Errorf("Size(false) = %d, want 17 cards outside the sideboard", n)
	}
	if n := d.Size(true); n != 20 {
		t.Errorf("Size(true) = %d, want 20 cards with the sideboard", n)
	}
}

func TestParseSumsCopiesAndSpellings(t *testing.T) {
	d, err := Parse("MainDeck:\n1 Flash\n3x Lightning Rush\n2 flash\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Sections) != 1 || len(d.Sections[0].Cards) != 2 {
		t.Fatalf("parsed %+v, want one section of two cards", d.Sections)
	}
	if got := d.Sections[0].Cards[0]; got.Quantity != 3 || got.Name != "Flash" {
		t.Errorf("first entry = %+v, want 3 Flash under the spelling it was first given", got)
	}
	if got := d.Sections[0].Cards[1]; got.Quantity != 3 || got.Name != "Lightning Rush" {
		t.Errorf("second entry = %+v, want 3 Lightning Rush", got)
	}
}

func TestParseRejectsWhatIsntADecklist(t *testing.T) {
	for name, text := range map[string]string{
		"a line with no count":      "MainDeck:\nLightning Rush\n",
		"a card before any heading": "3 Lightning Rush\n",
		"a heading with no name":    ":\n3 Lightning Rush\n",
		"no cards at all":           "MainDeck:\n\nSideboard:\n",
		"a count of zero":           "MainDeck:\n0 Lightning Rush\n",
	} {
		if _, err := Parse(text); err == nil {
			t.Errorf("%s: want an error, got none", name)
		}
	}
}

// A card named for a number word mustn't be eaten by the "3x" spelling.
func TestParseKeepsNamesThatStartWithX(t *testing.T) {
	d, err := Parse("MainDeck:\n1 Xerath, Ascended\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := d.Sections[0].Cards[0].Name; got != "Xerath, Ascended" {
		t.Errorf("name = %q, want Xerath, Ascended", got)
	}
}
