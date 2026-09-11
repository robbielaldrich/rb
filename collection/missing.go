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

// Axis picks which reading of "missing" a report answers.
type Axis int

const (
	// AxisPlayset asks whether a deck's three copies (one, for a battlefield
	// or legend) are in hand — the harder question, and the default.
	AxisPlayset Axis = iota
	// AxisSet asks only whether the card is held at all, mirroring the named
	// total Stats reports set by set.
	AxisSet
)

func (a Axis) String() string {
	if a == AxisSet {
		return "set completion"
	}
	return "playset completion"
}

// ParseAxis reads the axis names the command line and the stats viewer both
// accept.
func ParseAxis(s string) (Axis, error) {
	switch s {
	case "", "playset":
		return AxisPlayset, nil
	case "set":
		return AxisSet, nil
	default:
		return 0, fmt.Errorf("%q is not an axis, want \"set\" or \"playset\"", s)
	}
}

// Missing writes every card the collection is still short of to w, along the
// given axis: every card no copy of is held (AxisSet), or every card held in
// fewer than the copies a deck may run (AxisPlayset). It is the mirror of
// Surplus — what to look for rather than what to trade away.
//
// With no filters the whole catalog is measured; a set label or a domain
// narrows it, for the common case of finishing off one set.
//
// Only the cards a set prints at their own number are wanted here. Alternate
// arts, overnumbered copies and signatures are chase printings rather than
// cards a deck is short of, so they are not asked for — though holding one
// does count towards the card it reprints, since a deck plays it just the same.
func Missing(collectionPath, catalogPath string, filters []string, axis Axis, w io.Writer) error {
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

	writeMissing(w, wants(cs, coll, axis), scope, axis)
	return nil
}

// want is one card held short of the axis's target: the playset, or just one
// copy.
type want struct {
	setID  string
	number string
	sortID string
	name   string
	rarity string
	copies int
	target int
}

// short is how many copies are still needed to reach the target.
func (w want) short() int { return w.target - w.copies }

func wants(cs []cards.Card, coll *collection, axis Axis) []want {
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
			target := c.PlaysetSize()
			if axis == AxisSet {
				target = 1
			}
			e = &want{setID: id, name: c.BaseName(), target: target}
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

func writeMissing(w io.Writer, out []want, scope string, axis Axis) {
	where := "the collection"
	if scope != "" {
		where = scope
	}
	if len(out) == 0 {
		if axis == AxisSet {
			fmt.Fprintf(w, "%s is fully collected\n", where)
		} else {
			fmt.Fprintf(w, "%s is complete in playsets\n", where)
		}
		return
	}

	targetCol := "playset"
	if axis == AxisSet {
		targetCol = "target"
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "set\tnumber\trarity\tcard\thave\t%s\tneed\n", targetCol)
	absent, copies := 0, 0
	for _, e := range out {
		if e.copies == 0 {
			absent++
		}
		copies += e.short()
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\t%d\n", e.setID, e.number, e.rarity, e.name, e.copies, e.target, e.short())
	}
	fmt.Fprintf(tw, "\t\t\t%d cards, %d of them unowned\t\t\t%d\n", len(out), absent, copies)
	tw.Flush()
}
