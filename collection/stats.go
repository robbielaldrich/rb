package collection

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"rb/cards"
)

// Stats writes a set-by-set completion summary to w.
//
// The headline is the named cards: one copy of every card a set prints,
// whatever art it wears. Alternate arts, overnumbered copies and signatures
// are the same card to play with, so any printing completes the entry; they
// are then counted again on their own, printing by printing, as the chase
// cards they are.
func Stats(collectionPath, catalogPath string, w io.Writer) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	writeStats(w, summarise(cs, coll))
	return nil
}

// tally is a count of what is owned out of what exists.
type tally struct{ owned, total int }

func (t tally) String() string {
	if t.total == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d %d%%", t.owned, t.total, t.owned*100/t.total)
}

func (t *tally) add(o tally) {
	t.owned, t.total = t.owned+o.owned, t.total+o.total
}

func (t *tally) count(owned bool) {
	t.total++
	if owned {
		t.owned++
	}
}

type setStats struct {
	setID string
	label string
	named tally
	alt   tally
	over  tally
	sig   tally
}

func summarise(cs []cards.Card, coll *collection) []setStats {
	owned := make(map[string]bool, len(coll.Cards))
	for _, e := range coll.Cards {
		if e.Quantity > 0 {
			owned[e.RiftboundID] = true
		}
	}

	// Named cards are keyed by set and base name so that every printing of a
	// card — plain, alternate art, overnumbered — folds into the one entry.
	type nameKey struct{ set, name string }
	printed := map[nameKey]bool{} // the set prints this card at its own number
	have := map[nameKey]bool{}    // some printing of it is in the collection

	var order []string
	stats := map[string]*setStats{}
	for _, c := range cs {
		id := strings.ToUpper(c.Set.SetID)
		s, ok := stats[id]
		if !ok {
			s = &setStats{setID: id, label: c.Set.Label}
			stats[id] = s
			order = append(order, id)
		}

		k := nameKey{id, c.BaseName()}
		if owned[c.RiftboundID] {
			have[k] = true
		}

		switch {
		case c.IsAlternateArt():
			s.alt.count(owned[c.RiftboundID])
		case c.Metadata.Overnumbered:
			s.over.count(owned[c.RiftboundID])
		case c.Metadata.Signature:
			s.sig.count(owned[c.RiftboundID])
		default:
			printed[k] = true
		}
	}

	for k := range printed {
		stats[k.set].named.count(have[k])
	}

	out := make([]setStats, len(order))
	for i, id := range order {
		out[i] = *stats[id]
	}
	return out
}

func writeStats(w io.Writer, sets []setStats) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "set\t\tcards\talt art\tovernumbered\tsignature")

	var all setStats
	for _, s := range sets {
		fmt.Fprintf(tw, "%s\t%s\t%v\t%v\t%v\t%v\n", s.setID, s.label, s.named, s.alt, s.over, s.sig)
		all.named.add(s.named)
		all.alt.add(s.alt)
		all.over.add(s.over)
		all.sig.add(s.sig)
	}

	fmt.Fprintf(tw, "all\t\t%v\t%v\t%v\t%v\n", all.named, all.alt, all.over, all.sig)
	tw.Flush()
}
