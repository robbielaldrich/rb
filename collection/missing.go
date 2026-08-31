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

// Missing writes every card the collection is still short of to w: the ones
// no copy of is held, and the ones held in fewer than the copies a deck may
// run. It is the mirror of Surplus — what to look for rather than what to
// trade away.
//
// With no filters the whole catalog is measured; a set label or a domain
// narrows it, for the common case of finishing off one set.
//
// Only the cards a set prints at their own number are wanted here. Alternate
// arts, overnumbered copies and signatures are chase printings rather than
// cards a deck is short of, so they are not asked for — though holding one
// does count towards the card it reprints, since a deck plays it just the same.
func Missing(collectionPath, catalogPath string, filters []string, w io.Writer) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	cs, scope, err := narrow(cs, filters)
	if err != nil {
		return err
	}

	writeMissing(w, wants(cs, coll), scope)
	return nil
}

// want is one card held in fewer copies than a deck can use.
type want struct {
	setID   string
	number  string
	sortID  string
	name    string
	rarity  string
	copies  int
	playset int
}

// short is how many copies are still needed to fill the playset.
func (w want) short() int { return w.playset - w.copies }

func wants(cs []cards.Card, coll *collection) []want {
	qty := make(map[string]int, len(coll.Cards))
	for _, e := range coll.Cards {
		if e.Quantity > 0 {
			qty[e.RiftboundID] = e.Quantity
		}
	}

	// Keyed by set and base name so every printing of a card folds into the
	// one entry, the way Stats counts them: three copies make a playset
	// whether or not they wear the same art.
	type nameKey struct{ set, name string }
	held := map[nameKey]*want{}
	for _, c := range cs {
		id := strings.ToUpper(c.Set.SetID)
		k := nameKey{id, c.BaseName()}
		e, ok := held[k]
		if !ok {
			e = &want{setID: id, name: c.BaseName(), playset: c.PlaysetSize()}
			held[k] = e
		}
		e.copies += qty[c.RiftboundID]

		// A chase printing carries neither number nor rarity of its own into
		// the report: the card is wanted at the number the set prints it at,
		// and at the rarity it is pulled at there rather than the Showcase a
		// reprint of it wears.
		if !c.IsChasePrinting() {
			e.number, e.sortID, e.rarity = c.Number(), c.RiftboundID, c.Classification.Rarity
		}
	}

	var out []want
	for _, e := range held {
		// An entry with no number is one the filtered catalog shows only in
		// chase printings, so there is no plain card to go looking for.
		if e.number != "" && e.short() > 0 {
			out = append(out, *e)
		}
	}

	// Binder order: by set, then by riftbound_id, whose zero-padded numbers
	// sort as they are printed and whose prefixes keep the main run, the runes
	// and the promos in their own blocks rather than interleaved by a
	// collector number the three series each start again from one.
	slices.SortFunc(out, func(a, b want) int {
		return cmp.Or(
			cmp.Compare(a.setID, b.setID),
			cmp.Compare(a.sortID, b.sortID),
		)
	})
	return out
}

func writeMissing(w io.Writer, out []want, scope string) {
	where := "the collection"
	if scope != "" {
		where = scope
	}
	if len(out) == 0 {
		fmt.Fprintf(w, "%s is complete in playsets\n", where)
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "set\tnumber\trarity\tcard\thave\tplayset\tneed")
	absent, copies := 0, 0
	for _, e := range out {
		if e.copies == 0 {
			absent++
		}
		copies += e.short()
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\t%d\n", e.setID, e.number, e.rarity, e.name, e.copies, e.playset, e.short())
	}
	fmt.Fprintf(tw, "\t\t\t%d cards, %d of them unowned\t\t\t%d\n", len(out), absent, copies)
	tw.Flush()
}
