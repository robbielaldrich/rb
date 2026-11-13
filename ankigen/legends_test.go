package ankigen

import (
	"image"
	"strings"
	"testing"

	"rb/cards"
)

func TestGenerateLegendAbilities(t *testing.T) {
	teemo := legendCard("ogn-263-298", "Teemo - Swift Scout", "ogn", 263, "Teemo")
	teemo.Text.Rich = "<p>You may pay :rb_energy_1: to hide a card with [Hidden] instead of :rb_rune_rainbow:.<br />:rb_energy_1:, :rb_exhaust:: Put a Teemo unit you own into your hand.</p>"
	overnumbered := legendCard("ogn-299-298", "Teemo - Swift Scout (Overnumbered)", "ogn", 299, "Teemo")
	overnumbered.Metadata.Overnumbered = true
	opts := fixture(t, []cards.Card{teemo, overnumbered})

	res, err := GenerateLegendAbilities(opts)
	if err != nil {
		t.Fatalf("GenerateLegendAbilities: %v", err)
	}
	if res.Notes != 1 {
		t.Fatalf("Notes = %d, want one note for the legend's plain printing", res.Notes)
	}

	_, rows := readDeck(t, res.DeckFile)
	r := rows[0]
	if r[0] != "<img src='rb-ogn-263-298-title.jpg'><br>Legend ability" {
		t.Errorf("front = %q", r[0])
	}
	want := "What legend:<br>You may pay ① to hide a card with <b>[Hidden]</b> instead of [Any Rune].<br>①, [Exhaust]: Put a ____ unit you own into your hand."
	if r[1] != want {
		t.Errorf("back = %q\nwant  %q", r[1], want)
	}
	if r[3] != "Riftbound::Legend Abilities" || r[4] != "Basic (and reversed card)" {
		t.Errorf("deck, notetype = %q, %q", r[3], r[4])
	}
	if !strings.Contains(r[2], "riftbound::legend-abilities") {
		t.Errorf("tags = %q", r[2])
	}
}

// Only whole words are blanked: a champion's name inside another word is not
// their name.
func TestAbilityHTMLBlanksOnlyWholeNames(t *testing.T) {
	c := legendCard("x", "Vi - Piltover Enforcer", "unl", 1, "Vi")
	c.Text.Rich = "<p>Give Vi units +1 :rb_might: this turn, via Piltover.</p>"
	got := abilityHTML(c)
	want := "Give ____ units +1 Might this turn, via Piltover."
	if got != want {
		t.Errorf("abilityHTML = %q, want %q", got, want)
	}
}

func TestCropTopKeepsTheTop(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 10, 100))
	got := cropTop(src, 0.768)
	if got.Bounds().Dx() != 10 || got.Bounds().Dy() != 77 {
		t.Errorf("bounds = %v, want 10x77", got.Bounds())
	}
}
