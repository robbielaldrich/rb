package decks

import (
	"strings"
	"testing"
)

// script builds the input of a session: each deck pasted, closed, and named.
func script(parts ...string) string { return strings.Join(parts, "") }

func newAdder(t *testing.T) *adder {
	t.Helper()
	return &adder{
		registry: &registry{},
		pool:     testPool(nil),
		path:     t.TempDir() + "/decks.json",
	}
}

func (a *adder) session(t *testing.T, input string) string {
	t.Helper()
	var out strings.Builder
	if err := a.run(strings.NewReader(input), &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	return out.String()
}

func TestAddRecordsEachPastedDeck(t *testing.T) {
	a := newAdder(t)
	out := a.session(t, script(
		pasted, ".\n", "Worlds T1\n",
		"MainDeck:\n1 Shadow\n", ".\n", "\n",
	))

	if len(a.registry.Decks) != 2 || a.added != 2 {
		t.Fatalf("registered %d decks, want both", len(a.registry.Decks))
	}
	if got := a.registry.Decks[0].Name; got != "Worlds T1" {
		t.Errorf("first deck is named %q, want the name that was typed", got)
	}
	// An empty answer keeps the name the list gives itself.
	if got := a.registry.Decks[1].Name; got != "Shadow" {
		t.Errorf("second deck is named %q, want the name it defaults to", got)
	}
	if a.registry.Decks[0].AddedAt.IsZero() {
		t.Error("the deck was filed without a date")
	}
	if !strings.Contains(out, `saved "Worlds T1" · 17 cards, 3 in the sideboard`) {
		t.Errorf("the session doesn't report what it saved:\n%s", out)
	}

	// Each deck is written through as it is taken, not at the end.
	back, err := loadRegistry(a.path)
	if err != nil {
		t.Fatalf("loadRegistry: %v", err)
	}
	if len(back.Decks) != 2 {
		t.Errorf("the file holds %d decks, want both", len(back.Decks))
	}
}

func TestAddIgnoresADeckAlreadyRegistered(t *testing.T) {
	a := newAdder(t)
	out := a.session(t, script(pasted, ".\n", "Worlds T1\n", pasted, ".\n"))

	if len(a.registry.Decks) != 1 {
		t.Fatalf("registered %d decks, want the duplicate dropped", len(a.registry.Decks))
	}
	if !strings.Contains(out, `same cards as "Worlds T1"`) {
		t.Errorf("the session doesn't say which deck it matched:\n%s", out)
	}
}

// A list closed by the end of the input, rather than by the line that ends a
// deck, is still filed.
func TestAddTakesTheLastDeckAtEndOfInput(t *testing.T) {
	a := newAdder(t)
	a.session(t, "MainDeck:\n1 Shadow\n")

	if len(a.registry.Decks) != 1 {
		t.Fatalf("registered %d decks, want the pasted one", len(a.registry.Decks))
	}
	if got := a.registry.Decks[0].Name; got != "Shadow" {
		t.Errorf("deck named %q, want the name it defaults to", got)
	}
}

func TestAddWarnsAboutANameTheCatalogDoesntPrint(t *testing.T) {
	a := newAdder(t)
	out := a.session(t, script("MainDeck:\n1 Lightnign Rush\n", ".\n", "typo deck\n"))

	if !strings.Contains(out, `no card called "Lightnign Rush" in the catalog`) {
		t.Errorf("the session doesn't warn about the misspelling:\n%s", out)
	}
	if len(a.registry.Decks) != 1 {
		t.Error("the deck wasn't recorded; a name the catalog is behind on shouldn't lose the list")
	}
}

func TestAddKeepsGoingAfterAPasteThatIsntADeck(t *testing.T) {
	a := newAdder(t)
	out := a.session(t, script("just some text\n", ".\n", "MainDeck:\n1 Shadow\n", ".\n", "\n"))

	if !strings.Contains(out, "not a decklist") {
		t.Errorf("the session doesn't report the bad paste:\n%s", out)
	}
	if len(a.registry.Decks) != 1 {
		t.Errorf("registered %d decks, want the good one that followed", len(a.registry.Decks))
	}
}
