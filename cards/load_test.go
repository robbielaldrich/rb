package cards

import "testing"

func card(text string) Card { return Card{Text: Text{Plain: text}} }

func TestHasKeyword(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want bool
	}{
		{"bracketed", "[Hidden] (Hide now for :rb_rune_rainbow: to react later.)Deal 2.", true},
		{"second in a run", "[Ganking][Hidden]I can move.", true},
		{"run without reminder text", "[Hidden][Backline]Once each turn...", true},
		// Windsinger's rules text is missing its brackets in the API data.
		{"bare with reminder text", "Hidden (Hide now for :rb_rune_rainbow:.)When you play me...", true},
		// Teemo lets you hide other cards without being Hidden himself.
		{"named mid-sentence", "You may pay :rb_energy_1: to hide a card with [Hidden] instead.", false},
		{"named in reminder text", "During showdowns here, cards cost more. (Hidden cards have...)", false},
		{"other keyword only", "[Ganking]I can move from battlefield to battlefield.", false},
		{"bare without reminder text", "Hidden Blade deals 2 damage.", false},
		{"empty", "", false},
	} {
		if got := card(tc.text).HasKeyword("Hidden"); got != tc.want {
			t.Errorf("%s: HasKeyword(Hidden) = %v, want %v for %q", tc.name, got, tc.want, tc.text)
		}
	}
}

func TestBaseName(t *testing.T) {
	for name, want := range map[string]string{
		"Teemo - Scout":                  "Teemo - Scout",
		"Teemo - Scout (Alternate Art)":  "Teemo - Scout",
		"Teemo - Strategist (Signature)": "Teemo - Strategist",
		"Pyke - Dockside Butcher":        "Pyke - Dockside Butcher",
	} {
		if got := (Card{Name: name}).BaseName(); got != want {
			t.Errorf("BaseName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestNumberAndLabel(t *testing.T) {
	c := Card{Name: "Astral Heron", RiftboundID: "ven-044-166", CollectorNumber: 44, Set: CardSet{SetID: "ven"}}
	if got := c.Number(); got != "44/166" {
		t.Errorf("Number() = %q, want 44/166", got)
	}
	if got := c.Label(); got != "Astral Heron 44/166 VEN" {
		t.Errorf("Label() = %q", got)
	}
	// A riftbound_id that carries no set size falls back to the raw number.
	bare := Card{Name: "Fury Rune", RiftboundID: "rune-fury", CollectorNumber: 7}
	if got := bare.Number(); got != "7" {
		t.Errorf("Number() = %q, want 7", got)
	}
}

// TestHiddenCardsInCatalog guards the keyword rule against the real data,
// where the interesting cases actually live.
func TestHiddenCardsInCatalog(t *testing.T) {
	cards, err := Load("cards.json")
	if err != nil {
		t.Skip(err)
	}
	var names []string
	for _, c := range cards {
		if c.HasKeyword("Hidden") {
			names = append(names, c.Label())
			if c.Attributes.Energy == nil {
				t.Errorf("%s has Hidden but no energy cost to ask about", c.Label())
			}
		}
		if got, want := c.HasKeyword("Hidden"), c.Name == "Windsinger"; want && !got {
			t.Errorf("%s should have Hidden: %q", c.Label(), c.Text.Plain)
		}
		if c.BaseName() == "Teemo - Swift Scout" && c.HasKeyword("Hidden") {
			t.Errorf("%s only refers to Hidden, it does not have it", c.Label())
		}
	}
	t.Logf("%d cards with Hidden: %v", len(names), names)
	if len(names) < 20 {
		t.Errorf("found only %d Hidden cards, expected the whole keyword cycle", len(names))
	}
}
