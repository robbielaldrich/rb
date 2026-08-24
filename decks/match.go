package decks

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"rb/cards"
	"rb/collection"
)

// Shortfall is one card a deck asks for that the collection can't cover.
type Shortfall struct {
	Name string
	Need int
	Have int
	// Unknown marks a name no card in the catalog is printed under, usually a
	// typo in the pasted list or a set the catalog hasn't caught up with.
	Unknown bool
	// Nearest is the catalog name closest to an unknown one, where there is
	// one close enough to be worth suggesting.
	Nearest string
}

func (s Shortfall) short() int { return s.Need - s.Have }

// Result is how one deck stands against the collection.
type Result struct {
	Deck    Deck
	Size    int
	Missing []Shortfall
}

// Buildable reports whether every card the deck asks for is already owned.
func (r Result) Buildable() bool { return len(r.Missing) == 0 }

// Short counts the copies that would have to be found to build the deck.
func (r Result) Short() int {
	n := 0
	for _, m := range r.Missing {
		n += m.short()
	}
	return n
}

// Options says what to measure and where to leave the answer.
type Options struct {
	DecksPath      string
	CollectionPath string
	CatalogPath    string
	// ReportPath is where to keep a copy of the report to read later; no copy
	// is kept when it is empty.
	ReportPath string
	// Sideboard requires the sideboard as well, which a deck can be taken to
	// a table without.
	Sideboard bool
}

// Match measures every registered deck against the collection and writes the
// report to w: the decks that can be built today first, then the rest in the
// order they are close to being built. The copy on disk is the same report
// word for word, so what was read on screen is what can be read back.
func Match(opts Options, w io.Writer) error {
	reg, err := loadRegistry(opts.DecksPath)
	if err != nil {
		return fmt.Errorf("failed to load the deck register: %w", err)
	}
	if len(reg.Decks) == 0 {
		return fmt.Errorf("%s holds no decks yet, run `rb add-decks` first", opts.DecksPath)
	}

	cs, err := cards.Load(opts.CatalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	owned, err := collection.Owned(opts.CollectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	var report bytes.Buffer
	writeReport(&report, evaluate(reg.Decks, newPool(cs, owned), opts.Sideboard), opts.Sideboard)

	if _, err := w.Write(report.Bytes()); err != nil {
		return fmt.Errorf("failed to write the report: %w", err)
	}
	if opts.ReportPath == "" {
		return nil
	}
	if err := writeFile(opts.ReportPath, report.Bytes()); err != nil {
		return fmt.Errorf("failed to keep a copy of the report: %w", err)
	}
	return nil
}

// evaluate ranks the decks by how far from buildable they are.
func evaluate(ds []Deck, p *pool, sideboard bool) []Result {
	out := make([]Result, len(ds))
	for i, d := range ds {
		out[i] = check(d, p, sideboard)
	}

	slices.SortFunc(out, func(a, b Result) int {
		// Copies come first: a deck four short of one card is nearer to
		// built than one a card short of four.
		if n := cmp.Compare(a.Short(), b.Short()); n != 0 {
			return n
		}
		if n := cmp.Compare(len(a.Missing), len(b.Missing)); n != 0 {
			return n
		}
		return cmp.Compare(strings.ToLower(a.Deck.Name), strings.ToLower(b.Deck.Name))
	})
	return out
}

func check(d Deck, p *pool, sideboard bool) Result {
	r := Result{Deck: d}
	for _, e := range d.Cards(sideboard) {
		if p.isRune(e.Name) {
			continue
		}
		r.Size += e.Quantity

		have, known := p.have(e.Name)
		if known && have >= e.Quantity {
			continue
		}
		short := Shortfall{Name: e.Name, Need: e.Quantity, Have: have, Unknown: !known}
		if !known {
			short.Nearest, _ = p.nearest(e.Name)
		}
		r.Missing = append(r.Missing, short)
	}
	return r
}

func writeReport(w io.Writer, res []Result, sideboard bool) {
	scope := "runes ignored · sideboards excluded"
	if sideboard {
		scope = "runes ignored · sideboards included"
	}
	built := 0
	for _, r := range res {
		if r.Buildable() {
			built++
		}
	}
	fmt.Fprintf(w, "%d %s · %d you can build · %s\n", len(res), plural(len(res), "deck"), built, scope)

	for _, r := range res {
		if r.Buildable() {
			fmt.Fprintf(w, "\n  ✓ %s · %d %s\n", r.Deck.Name, r.Size, plural(r.Size, "card"))
			continue
		}
		fmt.Fprintf(w, "\n  ✗ %s · %d of %d %s missing\n", r.Deck.Name, r.Short(), r.Size, plural(r.Size, "card"))
		for _, line := range shortfallLines(r.Missing) {
			fmt.Fprintf(w, "      %s\n", line)
		}
	}
}

// shortfallLines renders the missing cards as a shopping list, e.g.
// "2 Lightning Rush    have 1 of 3", with the counts in a column.
func shortfallLines(ms []Shortfall) []string {
	width := 0
	for _, m := range ms {
		width = max(width, utf8.RuneCountInString(m.Name))
	}

	lines := make([]string, len(ms))
	for i, m := range ms {
		note := fmt.Sprintf("have %d of %d", m.Have, m.Need)
		switch {
		case m.Unknown && m.Nearest != "":
			note = fmt.Sprintf("no such card in the catalog, did you mean %q?", m.Nearest)
		case m.Unknown:
			note = "no such card in the catalog"
		}
		pad := strings.Repeat(" ", width-utf8.RuneCountInString(m.Name))
		lines[i] = strconv.Itoa(m.short()) + " " + m.Name + pad + "   " + note
	}
	return lines
}
