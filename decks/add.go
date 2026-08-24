package decks

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"rb/cards"
)

// endOfDeck is what closes a pasted list. A blank line can't do it: lists
// carry blank lines between their sections, and the point is to paste one
// exactly as it was copied.
const endOfDeck = "."

// RunAdder takes pasted decklists one after another and appends the ones the
// register doesn't already hold, writing the file through after each.
//
// The catalog is read only to warn about names no card is printed under; a
// list is recorded as pasted either way, since the catalog may simply be
// behind the format.
func RunAdder(decksPath, catalogPath string) error {
	reg, err := loadRegistry(decksPath)
	if err != nil {
		return fmt.Errorf("failed to load the deck register: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	a := &adder{registry: reg, path: decksPath, pool: newPool(cs, nil)}
	if err := a.run(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("failed to read the pasted decks: %w", err)
	}

	fmt.Printf("\nadded %d %s · %d in the register\n", a.added, plural(a.added, "deck"), len(reg.Decks))
	return nil
}

type adder struct {
	registry *registry
	pool     *pool
	path     string
	added    int
}

func (a *adder) run(in io.Reader, out io.Writer) error {
	fmt.Fprintf(out, "paste a decklist, then %q on a line of its own to save it · ctrl+d to stop\n", endOfDeck)

	sc := bufio.NewScanner(in)
	for {
		fmt.Fprintf(out, "\ndeck %d:\n", len(a.registry.Decks)+1)
		text, more := readDeck(sc)
		if strings.TrimSpace(text) != "" {
			if err := a.record(sc, out, text); err != nil {
				return err
			}
		}
		if !more {
			break
		}
	}

	if err := sc.Err(); err != nil {
		return fmt.Errorf("failed to read a pasted line: %w", err)
	}
	return nil
}

// record parses one pasted list and, unless the register already holds these
// exact cards, files it under a name the user is asked for.
func (a *adder) record(sc *bufio.Scanner, out io.Writer, text string) error {
	d, err := Parse(text)
	if err != nil {
		fmt.Fprintf(out, "  not a decklist: %v\n", err)
		return nil
	}
	if have, ok := a.registry.duplicate(d); ok {
		fmt.Fprintf(out, "  same cards as %q, already registered\n", have.Name)
		return nil
	}
	for _, e := range d.Copies(true) {
		if _, known := a.pool.have(e.Name); known {
			continue
		}
		if near, ok := a.pool.nearest(e.Name); ok {
			fmt.Fprintf(out, "  no card called %q in the catalog, did you mean %q? recording it as pasted\n", e.Name, near)
			continue
		}
		fmt.Fprintf(out, "  no card called %q in the catalog, recording it as pasted\n", e.Name)
	}

	fmt.Fprintf(out, "  name [%s]: ", d.Name)
	if name := readLine(sc); name != "" {
		d.Name = name
	}
	d.AddedAt = time.Now()

	a.registry.Decks = append(a.registry.Decks, d)
	if err := a.registry.save(a.path); err != nil {
		return fmt.Errorf("failed to save the deck register: %w", err)
	}
	a.added++

	fmt.Fprintf(out, "  saved %q · %d cards, %d in the sideboard\n",
		d.Name, d.Size(false), d.Size(true)-d.Size(false))
	return nil
}

// readDeck reads one pasted list, up to the line that closes it or the end of
// the input, and reports whether any input is left after it.
func readDeck(sc *bufio.Scanner) (text string, more bool) {
	var b strings.Builder
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == endOfDeck {
			return b.String(), true
		}
		b.WriteString(sc.Text())
		b.WriteByte('\n')
	}
	return b.String(), false
}

// readLine reads an answer to a prompt, treating the end of the input as an
// empty answer so a deck already pasted is still filed.
func readLine(sc *bufio.Scanner) string {
	if !sc.Scan() {
		return ""
	}
	return strings.TrimSpace(sc.Text())
}
