package collection

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"rb/cards"
)

// summarise's headline is the named cards: one copy of every card a set
// prints, whatever art it wears. Alternate arts, overnumbered copies and
// signatures are the same card to play with, so any printing completes the
// entry; they are then counted again on their own, printing by printing, as
// the chase cards they are.
//
// Playsets ask the harder question beside it: not whether a card is in the
// collection at all, but whether it is there in the three copies a deck may
// run. Battlefields and legends stay out of that count, being cards a deck
// fields one of rather than in repeats.

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
	setID   string
	label   string
	named   tally
	playset tally
	alt     tally
	over    tally
	sig     tally
}

func summarise(cs []cards.Card, coll *collection) []setStats {
	qty := make(map[string]int, len(coll.Cards))
	for _, e := range coll.Cards {
		if e.Quantity > 0 {
			qty[e.RiftboundID] = e.Quantity
		}
	}

	// Named cards are keyed by set and base name so that every printing of a
	// card — plain, alternate art, overnumbered — folds into the one entry.
	// Copies are summed over those printings too: three of a card make a
	// playset whether or not they wear the same art.
	//
	// The set size each is numbered out of joins the key, because a promo set
	// holds promos of several sets at once: OPP's Origins Fury Rune and its
	// Vendetta one are two cards to collect, not one, and only the 298 and the
	// 166 they are numbered out of tell them apart.
	type nameKey struct{ set, name, size string }
	type card struct {
		printed bool // the set prints this card at its own number
		owned   bool // some printing of it is in the collection
		copies  int  // how many, over every printing
		playset int  // how many one deck can use
		promo   bool // the set prints it as a promo, not as a card of its own
	}
	byName := map[nameKey]*card{}

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

		k := nameKey{id, c.BaseName(), c.SetSize()}
		e, ok := byName[k]
		if !ok {
			e = &card{playset: c.PlaysetSize()}
			byName[k] = e
		}
		n := qty[c.RiftboundID]
		e.copies += n
		e.owned = e.owned || n > 0

		switch {
		case c.IsAlternateArt():
			s.alt.count(n > 0)
		case c.Metadata.Overnumbered:
			s.over.count(n > 0)
		case c.Metadata.Signature:
			s.sig.count(n > 0)
		default:
			e.printed = true
			e.promo = e.promo || c.IsPromo()
		}
	}

	for k, e := range byName {
		if !e.printed {
			continue
		}
		s := stats[k.set]
		s.named.count(e.owned)
		// Battlefields and legends sit out the playset share: a deck fields one
		// of each, so holding them to three would only dilute the reading of
		// how far the cards you do need in triplicate have come. So do the
		// promo sets, whose every card is another printing of one a main set
		// already prints: the playset is measured once, against the card, and
		// promo copies count towards it wherever they are filed.
		if e.playset > 1 && !e.promo {
			s.playset.count(e.copies >= e.playset)
		}
	}

	out := make([]setStats, len(order))
	for i, id := range order {
		out[i] = *stats[id]
	}
	return out
}

// totals folds every set into the one row that stands for the whole game.
func totals(sets []setStats) setStats {
	var all setStats
	for _, s := range sets {
		all.named.add(s.named)
		all.playset.add(s.playset)
		all.alt.add(s.alt)
		all.over.add(s.over)
		all.sig.add(s.sig)
	}
	return all
}

func writeStats(w io.Writer, sets []setStats) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "set\t\tcards\tplaysets\talt art\tovernumbered\tsignature")

	for _, s := range sets {
		fmt.Fprintf(tw, "%s\t%s\t%v\t%v\t%v\t%v\t%v\n", s.setID, s.label, s.named, s.playset, s.alt, s.over, s.sig)
	}

	all := totals(sets)
	fmt.Fprintf(tw, "all\t\t%v\t%v\t%v\t%v\t%v\n", all.named, all.playset, all.alt, all.over, all.sig)
	tw.Flush()
}
