package collection

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"rb/cards"
)

// Surplus writes every card held beyond a playset to w: the spare copies that
// no single deck can use, and so the ones free to trade away.
//
// Copies are counted over every printing of a card, since a deck cares only
// how many it has to play with, not which art they wear. That does mean a
// spare can be an alternate art or a signature you would never trade — the
// report says what is surplus to a deck, and leaves what to do about it to
// the reader.
func Surplus(collectionPath, catalogPath string, w io.Writer) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	writeSurplus(w, spares(cs, coll))
	return nil
}

// spare is one card held past what a deck can use.
type spare struct {
	setID   string
	name    string
	copies  int
	playset int
}

// over is how many copies beyond the playset are held.
func (s spare) over() int { return s.copies - s.playset }

func spares(cs []cards.Card, coll *collection) []spare {
	qty := make(map[string]int, len(coll.Cards))
	for _, e := range coll.Cards {
		if e.Quantity > 0 {
			qty[e.RiftboundID] = e.Quantity
		}
	}

	type nameKey struct{ set, name string }
	held := map[nameKey]*spare{}
	for _, c := range cs {
		n := qty[c.RiftboundID]
		if n == 0 {
			continue
		}
		id := strings.ToUpper(c.Set.SetID)
		k := nameKey{id, c.BaseName()}
		s, ok := held[k]
		if !ok {
			s = &spare{setID: id, name: c.BaseName(), playset: c.PlaysetSize()}
			held[k] = s
		}
		s.copies += n
	}

	var out []spare
	for _, s := range held {
		if s.over() > 0 {
			out = append(out, *s)
		}
	}

	// Deepest surplus first, since that is what a trade binder is built from;
	// set and name break ties so the list is stable between runs.
	slices.SortFunc(out, func(a, b spare) int {
		return cmp.Or(
			cmp.Compare(b.over(), a.over()),
			cmp.Compare(a.setID, b.setID),
			cmp.Compare(a.name, b.name),
		)
	})
	return out
}

func writeSurplus(w io.Writer, out []spare) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no card is held past a playset")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "set\tcard\thave\tplayset\tspare")
	total := 0
	for _, s := range out {
		total += s.over()
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t+%d\n", s.setID, s.name, s.copies, s.playset, s.over())
	}
	fmt.Fprintf(tw, "\t%d cards\t\t\t+%d\n", len(out), total)
	tw.Flush()
}
