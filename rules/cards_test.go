package rules

import (
	"testing"

	"rb/cards"
)

func card(name, id string, meta cards.Metadata) cards.Card {
	c := cards.Card{Name: name, RiftboundID: id, Metadata: meta}
	c.Media.ImageURL = "https://example.test/" + id + ".png"
	return c
}

func namesIn(t *testing.T, question string, cs []cards.Card) []string {
	t.Helper()
	var out []string
	for _, r := range cardsNamedIn(question, indexNames(cs)) {
		out = append(out, r.Name)
	}
	return out
}

func TestFlattenIgnoresPunctuation(t *testing.T) {
	for in, want := range map[string]string{
		"Akshan - Mischievous":  "Akshan Mischievous",
		"Akshan, Mischievous":   "Akshan Mischievous",
		"Kai'Sa - Evolutionary": "KaiSa Evolutionary",
		"Zhonya's Hourglass?":   "Zhonyas Hourglass",
		"  spaced   out  ":      "spaced out",
	} {
		if got := flatten(in); got != want {
			t.Errorf("flatten(%q) = %q, want %q", in, got, want)
		}
	}
}

// The rulings write a champion with a comma where Origins prints a dash, so
// the two only meet once the punctuation is out of the way.
func TestQuestionFindsAChampionWrittenWithAComma(t *testing.T) {
	cs := []cards.Card{card("Akshan - Mischievous", "ogn-001-298", cards.Metadata{})}

	got := namesIn(t, "How does Akshan, Mischievous interact with the chain?", cs)
	if len(got) != 1 || got[0] != "Akshan - Mischievous" {
		t.Errorf("got %v, want the card named as the catalog prints it", got)
	}
}

// Half the shortest card names are ordinary words, so the case is what tells
// the card Block from the act of blocking.
func TestOrdinaryWordsAreNotCardNames(t *testing.T) {
	cs := []cards.Card{card("Block", "ogn-002-298", cards.Metadata{})}

	if got := namesIn(t, "Can I block after the chain resolves?", cs); len(got) != 0 {
		t.Errorf("got %v, want the lower-case verb left alone", got)
	}
	if got := namesIn(t, "Does Block counter a Flow spell?", cs); len(got) != 1 {
		t.Errorf("got %v, want the card itself found", got)
	}
}

// A name sitting inside a longer card's name goes with it.
func TestALongerNameWins(t *testing.T) {
	cs := []cards.Card{
		card("Shadow", "ven-001-166", cards.Metadata{}),
		card("Shadow Assassin", "ven-002-166", cards.Metadata{}),
	}

	got := namesIn(t, "Does Shadow Assassin count itself when I play it from my trash?", cs)
	if len(got) != 1 || got[0] != "Shadow Assassin" {
		t.Errorf("got %v, want only the longer name", got)
	}
}

// A question naming several cards gets all of them, each once however often
// it is named.
func TestEveryCardNamedIsFound(t *testing.T) {
	cs := []cards.Card{
		card("Guardian Angel", "ogn-003-298", cards.Metadata{}),
		card("Tactical Retreat", "ogn-004-298", cards.Metadata{}),
		card("Smite", "ogn-005-298", cards.Metadata{}),
	}

	got := namesIn(t, "Can Guardian Angel or Tactical Retreat save a unit from Smite? Smite again?", cs)
	if len(got) != 3 {
		t.Errorf("got %v, want each of the three named once", got)
	}
}

// Printings share a name and differ in frame, so the scan shown is the plain
// one rather than an alternate art or a promo.
func TestThePlainPrintingsScanIsUsed(t *testing.T) {
	cs := []cards.Card{
		card("Fox-Fire (Alternate Art)", "ogn-256a-298", cards.Metadata{AlternateArt: true}),
		card("Fox-Fire", "ogn-256-298", cards.Metadata{}),
	}

	got := cardsNamedIn("When does Fox-Fire resolve?", indexNames(cs))
	if len(got) != 1 {
		t.Fatalf("got %v, want one card", got)
	}
	if got[0].Image != "https://example.test/ogn-256-298.png" {
		t.Errorf("image = %q, want the plain printing's scan", got[0].Image)
	}
}

// A card with no scan can't be shown, so it isn't offered.
func TestCardsWithoutAScanAreSkipped(t *testing.T) {
	c := cards.Card{Name: "Ghostly", RiftboundID: "ven-009-166"}
	if got := namesIn(t, "What does Ghostly do?", []cards.Card{c}); len(got) != 0 {
		t.Errorf("got %v, want a card with no scan left out", got)
	}
}

// Guarded against the real data, where the names that are ordinary words and
// the champions spelt two ways actually live.
func TestRealQuestionsNameOnlyRealCards(t *testing.T) {
	rs, err := Load("rulings.json")
	if err != nil {
		t.Skip(err)
	}
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}

	idx := indexNames(cs)
	known := map[string]bool{}
	for _, c := range idx {
		known[c.name] = true
	}

	withCards := 0
	for _, r := range rs {
		named := cardsNamedIn(r.Question, idx)
		if len(named) > 0 {
			withCards++
		}
		for _, n := range named {
			if !known[n.Name] {
				t.Errorf("%q names %q, which the catalog doesn't print", r.Question, n.Name)
			}
			if n.Image == "" {
				t.Errorf("%q names %q with no scan", r.Question, n.Name)
			}
		}
	}
	if withCards == 0 {
		t.Error("no question names a card, which the dataset is full of")
	}
}
