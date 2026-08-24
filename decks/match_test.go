package decks

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestCheckCountsOnlyWhatIsShort(t *testing.T) {
	d := parse(t, "MainDeck:\n3 Lightning Rush\n1 Shadow\n\nSideboard:\n2 Nocturne, Horrifying\n")
	p := testPool(map[string]int{"ven-156-166": 1, "ven-042-166": 1})

	got := check(d, p, false)
	if len(got.Missing) != 1 {
		t.Fatalf("missing = %+v, want Lightning Rush alone", got.Missing)
	}
	if m := got.Missing[0]; m.Name != "Lightning Rush" || m.Need != 3 || m.Have != 1 || m.short() != 2 {
		t.Errorf("missing = %+v, want 2 more of the 3 Lightning Rush", m)
	}
	if got.Short() != 2 || got.Buildable() {
		t.Errorf("Short() = %d, Buildable() = %v, want 2 copies short", got.Short(), got.Buildable())
	}
}

func TestCheckCountsTheSideboardWhenAsked(t *testing.T) {
	d := parse(t, "MainDeck:\n1 Shadow\n\nSideboard:\n2 Nocturne, Horrifying\n")
	p := testPool(map[string]int{"ven-042-166": 1})

	if got := check(d, p, false); !got.Buildable() {
		t.Errorf("the deck without its sideboard is %+v, want buildable", got.Missing)
	}
	got := check(d, p, true)
	if len(got.Missing) != 1 || got.Missing[0].Name != "Nocturne, Horrifying" || got.Short() != 2 {
		t.Errorf("with the sideboard: %+v, want the two Nocturne short", got.Missing)
	}
}

func TestCheckFlagsANameTheCatalogDoesntPrint(t *testing.T) {
	d := parse(t, "MainDeck:\n2 Lightnign Rush\n")
	got := check(d, testPool(nil), false)
	if len(got.Missing) != 1 || !got.Missing[0].Unknown {
		t.Fatalf("missing = %+v, want the misspelling flagged as unknown", got.Missing)
	}
	if !strings.Contains(strings.Join(shortfallLines(got.Missing), "\n"), "no such card in the catalog") {
		t.Errorf("the report doesn't say the name is unknown: %v", shortfallLines(got.Missing))
	}
}

// A deck four copies short of one card is further off than one a copy short
// of two, so copies rank before cards.
func TestEvaluateRanksByCopiesShort(t *testing.T) {
	p := testPool(map[string]int{"ven-042-166": 1})
	ds := []Deck{
		parse(t, "MainDeck:\n4 Lightning Rush\n"),
		parse(t, "MainDeck:\n1 Shadow\n"),
		parse(t, "MainDeck:\n1 Lightning Rush\n1 Nocturne, Horrifying\n"),
	}
	ds[0].Name, ds[1].Name, ds[2].Name = "four short", "built", "two short"

	var order []string
	for _, r := range evaluate(ds, p, false) {
		order = append(order, r.Deck.Name)
	}
	if want := "built,two short,four short"; strings.Join(order, ",") != want {
		t.Errorf("order = %v, want %s", order, want)
	}
}

// Runes come in every product and hold nothing up, so they are no part of
// what a deck asks the collection for — wherever the list keeps them.
func TestRunesAreNotCounted(t *testing.T) {
	p := testPool(map[string]int{"ven-042-166": 1})

	pool := parse(t, "MainDeck:\n1 Shadow\n\nRune Pool:\n8 Chaos Rune\n")
	if got := check(pool, p, false); !got.Buildable() || got.Size != 1 {
		t.Errorf("with a rune pool: %d cards, missing %+v, want the one Shadow alone", got.Size, got.Missing)
	}

	loose := parse(t, "MainDeck:\n1 Shadow\n4 Chaos Rune\n")
	if got := check(loose, p, false); !got.Buildable() || got.Size != 1 {
		t.Errorf("with runes in the main deck: %d cards, missing %+v, want the one Shadow alone", got.Size, got.Missing)
	}
}

func TestReportSaysWhatCanBeBuilt(t *testing.T) {
	p := testPool(map[string]int{"ven-042-166": 1, "ven-156-166": 1})
	ds := []Deck{parse(t, "MainDeck:\n1 Shadow\n"), parse(t, "MainDeck:\n3 Lightning Rush\n")}
	ds[0].Name, ds[1].Name = "Shadow Aggro", "Rush Combo"

	var b strings.Builder
	writeReport(&b, evaluate(ds, p, false), false)

	got := b.String()
	for _, want := range []string{
		"2 decks · 1 you can build · runes ignored · sideboards excluded",
		"✓ Shadow Aggro · 1 card",
		"✗ Rush Combo · 2 of 3 cards missing",
		"2 Lightning Rush   have 1 of 3",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the report doesn't say %q:\n%s", want, got)
		}
	}
}

// The copy on disk has to be the report that was read on screen, so a run
// can be looked up later without being run again.
func TestMatchKeepsTheReportItPrinted(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		DecksPath:      dir + "/decks.json",
		CollectionPath: dir + "/collection.json",
		CatalogPath:    dir + "/cards.json",
		ReportPath:     dir + "/match-decks-result.txt",
	}

	write(t, opts.CatalogPath, `[{"name": "Shadow", "riftbound_id": "ven-042-166",
		"classification": {"type": "Unit"}, "set": {"set_id": "ven"},
		"metadata": {"updated_on": "2026-01-01"}}]`)
	write(t, opts.CollectionPath, `{"cards": [{"riftbound_id": "ven-042-166", "quantity": 1}]}`)

	reg := &registry{Decks: []Deck{parse(t, "MainDeck:\n1 Shadow\n")}}
	if err := reg.save(opts.DecksPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	var printed strings.Builder
	if err := Match(opts, &printed); err != nil {
		t.Fatalf("Match: %v", err)
	}

	kept, err := os.ReadFile(opts.ReportPath)
	if err != nil {
		t.Fatalf("the report wasn't kept: %v", err)
	}
	if string(kept) != printed.String() {
		t.Errorf("the copy on disk isn't what was printed:\n%s\nprinted:\n%s", kept, printed.String())
	}
	if !strings.Contains(printed.String(), "✓ Shadow · 1 card") {
		t.Errorf("the report doesn't say the deck can be built:\n%s", printed.String())
	}

	// Asking for no copy leaves none behind.
	opts.ReportPath = ""
	if err := Match(opts, io.Discard); err != nil {
		t.Fatalf("Match without a report path: %v", err)
	}
	if _, err := os.Stat(dir + "/match-decks-result.txt.tmp"); !os.IsNotExist(err) {
		t.Error("the atomic write left its temporary file behind")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
