package cards

// banned lists the cards banned in Standard Constructed, by the name every
// printing of them shares. The catalog carries no legality of its own, so this
// is kept by hand from the official announcements; the latest took effect
// September 18, 2026 (https://playriftbound.com/en-us/news/announcements/).
//
// Only cards a deck is built from are listed. The banned battlefields — The
// Dreaming Tree, Obelisk of Power, Reaver's Row, The Arena's Greatest and
// Aspirant's Climb — are left off since nothing here asks about battlefields,
// and so is Master Yi, Wuju Bladesman, who is banned in 2v2 only.
var banned = map[string]bool{
	"Called Shot":         true,
	"Draven - Vanquisher": true,
	"Ekko - Recurrent":    true,
	"Fight or Flight":     true,
	"Scrapheap":           true,
	"Stacked Deck":        true,
	"Stealthy Pursuer":    true,
}

// IsBanned reports whether a card is banned in Standard Constructed. Nobody
// can be holding one, so it is not worth learning to play around.
func (c Card) IsBanned() bool { return banned[c.BaseName()] }

// Legal drops the banned cards from cs, keeping the rest in order.
func Legal(cs []Card) []Card {
	out := make([]Card, 0, len(cs))
	for _, c := range cs {
		if !c.IsBanned() {
			out = append(out, c)
		}
	}
	return out
}
