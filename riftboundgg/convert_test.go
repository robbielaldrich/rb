package riftboundgg

import (
	"encoding/json"
	"strings"
	"testing"

	"rb/cards"
)

// radiance is a slice of the Radiance set as riftbound.gg lists it: enough of
// the main run to number the set, and one of each printing the ids tell apart.
func radiance() []row {
	unit := []string{"Unit"}
	return []row{
		{ID: "RAD-001", NameNormal: "Amateur Demolitionist", Rarity: "Common", Type: unit, Cost: "2", Might: "1", Power: "0"},
		{ID: "RAD-020", NameNormal: "Mordekaiser - Twice Slain", Rarity: "Rare", Type: unit},
		{ID: "RAD-020A", NameNormal: "Mordekaiser - Twice Slain", Rarity: "Showcase", Type: unit},
		{ID: "RAD-139", NameNormal: "Ekko - Boy Who Shattered Time", Rarity: "Rare", Type: []string{"Legend"}},
		{ID: "RAD-166", NameNormal: "Ruined Monastery", Rarity: "Uncommon", Type: []string{"Battlefield"}},
		{ID: "RAD-167", NameNormal: "Neeko - Blending In", Rarity: "Epic", Type: unit},
		{ID: "RAD-168", NameNormal: "Ekko - Boy Who Shattered Time", Rarity: "Showcase", Type: []string{"Legend"}},
		{ID: "RAD-168-STAR", NameNormal: "Ekko - Boy Who Shattered Time", Rarity: "Showcase", Type: []string{"Legend"}},
		{ID: "RAD-R01", NameNormal: "Fury Rune", Rarity: "Common", Type: []string{"Rune"}},
		{ID: "RAD-R01A", NameNormal: "Fury Rune", Rarity: "Showcase", Type: []string{"Rune"}},
		{ID: "RAD-R01b-P", Name: "Fury Rune (Radiance Nexus Night Promo)", NameNormal: "Fury Rune", Rarity: "Showcase", Type: []string{"Rune"}, Promo: "1"},
		{ID: "RAD-SP1", NameNormal: "Mordekaiser - Twice Slain", Rarity: "Showcase", Type: unit},
		{ID: "RAD-SP2", NameNormal: "Neeko - Blending In", Rarity: "Showcase", Type: unit},
		{ID: "RAD-015-P2", Name: "Akali - Brash (Champion Promo)", NameNormal: "Akali - Brash", Rarity: "Rare", Type: unit, Promo: "1"},
		{ID: "RAD-T01", NameNormal: "Bird // Bomb", Rarity: "Common", Type: unit},
	}
}

func convertAll(t *testing.T, rows []row) map[string]cards.Card {
	t.Helper()
	s := newSetInfo("RAD", "Radiance", rows, plainPrintings(rows))
	out := map[string]cards.Card{}
	for _, r := range rows {
		p, ok := parseID(r.ID)
		if !ok {
			t.Fatalf("parseID(%q) failed", r.ID)
		}
		if c, ok := convert(r, p, s); ok {
			out[r.ID] = c
		}
	}
	return out
}

// The set size isn't published, so it is read off the last battlefield: past
// it, Neeko at 167 is still a card of the set, not an overnumbered copy.
func TestSetIsNumberedByItsLastBattlefield(t *testing.T) {
	rows := radiance()
	if s := newSetInfo("RAD", "Radiance", rows, plainPrintings(rows)); s.size != 166 || s.specials != 2 {
		t.Errorf("size = %d, specials = %d, want 166 and 2", s.size, s.specials)
	}
}

// Ids follow the scheme the collection was recorded against, so that the
// cards already owned keep their ids.
func TestConvertedIDsAndNames(t *testing.T) {
	got := convertAll(t, radiance())

	for id, want := range map[string]struct{ riftboundID, name, rarity string }{
		"RAD-001":      {"rad-001-166", "Amateur Demolitionist", "Common"},
		"RAD-020A":     {"rad-020a-166", "Mordekaiser - Twice Slain (Alternate Art)", "Rare"},
		"RAD-167":      {"rad-167-166", "Neeko - Blending In", "Epic"},
		"RAD-168":      {"rad-168-166", "Ekko - Boy Who Shattered Time (Overnumbered)", "Rare"},
		"RAD-168-STAR": {"rad-168*-166", "Ekko - Boy Who Shattered Time (Signature)", "Rare"},
		"RAD-R01":      {"rad-r01", "Fury Rune", "Common"},
		"RAD-R01A":     {"rad-r01a", "Fury Rune (Alternate Art)", "Showcase"},
		"RAD-R01b-P":   {"rad-r01bp-166", "Fury Rune (Radiance Nexus Night Promo)", "Promo"},
		"RAD-SP1":      {"rad-sp1-002", "Mordekaiser - Twice Slain (Alternate Art)", "Rare"},
		"RAD-015-P2":   {"rad-015p2-166", "Akali - Brash (Champion Promo)", "Promo"},
	} {
		c, ok := got[id]
		if !ok {
			t.Errorf("%s was not converted", id)
			continue
		}
		if c.RiftboundID != want.riftboundID || c.Name != want.name || c.Classification.Rarity != want.rarity {
			t.Errorf("%s = %s %q %s, want %s %q %s", id, c.RiftboundID, c.Name, c.Classification.Rarity, want.riftboundID, want.name, want.rarity)
		}
	}
}

// The flags are what the collection stats sort chase printings by, and a
// promo must never read as an alternate art just because it reprints one.
func TestConvertedFlags(t *testing.T) {
	got := convertAll(t, radiance())

	if c := got["RAD-020A"]; !c.IsAlternateArt() || !c.IsChasePrinting() {
		t.Errorf("RAD-020A is not an alternate art: %+v", c.Metadata)
	}
	if c := got["RAD-168"]; !c.Metadata.Overnumbered {
		t.Errorf("RAD-168 is not overnumbered: %+v", c.Metadata)
	}
	if c := got["RAD-168-STAR"]; !c.Metadata.Signature {
		t.Errorf("RAD-168-STAR is not signed: %+v", c.Metadata)
	}
	if c := got["RAD-R01b-P"]; !c.IsPromo() || c.IsChasePrinting() {
		t.Errorf("RAD-R01b-P should be a promo and nothing else: %+v", c)
	}
	if _, ok := got["RAD-T01"]; ok {
		t.Error("a token was converted")
	}
}

func TestConvertedGameData(t *testing.T) {
	r := row{
		ID: "UNL-103", NameNormal: "Choose", Rarity: "Common", Type: []string{"Spell"}, Color: []string{"Calm"},
		Cost: "2", Power: "0",
		Effect: "[Reaction]<br />Choose one —<ul><li>Draw 1.</li></ul>",
	}
	c := convertAll(t, []row{r, {ID: "UNL-219", NameNormal: "Field", Rarity: "Uncommon", Type: []string{"Battlefield"}}})["UNL-103"]

	if c.Attributes.Energy == nil || *c.Attributes.Energy != 2 || c.Attributes.Power != nil || c.Attributes.Might != nil {
		t.Errorf("attributes = %+v, want energy 2 and nothing else", c.Attributes)
	}
	if want := "<p>[Reaction]<br />Choose one —</p><ul><li>Draw 1.</li></ul>"; c.Text.Rich != want {
		t.Errorf("rich = %q, want %q", c.Text.Rich, want)
	}
	if want := "[Reaction]Choose one —Draw 1."; c.Text.Plain != want {
		t.Errorf("plain = %q, want %q", c.Text.Plain, want)
	}
	if c.Text.Flavour != nil || c.Tags == nil {
		t.Errorf("flavour = %v, tags = %v, want no flavour and empty tags", c.Text.Flavour, c.Tags)
	}
}

// build refuses two rows that convert to one id, which cards.Load would
// otherwise quietly collapse into one card.
func TestBuildRefusesCollidingIDs(t *testing.T) {
	rows := append(radiance(), row{ID: "RAD-020a", NameNormal: "Mordekaiser - Twice Slain", Rarity: "Showcase", Type: []string{"Unit"}})
	_, _, err := build([]apiSet{{Code: "RAD", Name: "Radiance", ReleaseDate: "1792713600"}}, rows)
	if err == nil || !strings.Contains(err.Error(), "rad-020a-166") {
		t.Fatalf("build = %v, want a collision on rad-020a-166", err)
	}
}

func TestBuildDatesSetsAndSkipsUnlisted(t *testing.T) {
	rows := append(radiance(), row{ID: "ARC-001", NameNormal: "Vi - Destructive", Rarity: "Showcase"})
	sets, cs, err := build([]apiSet{{Code: "RAD", Name: "Radiance", CardsCount: "282", ReleaseDate: "1792713600"}}, rows)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(sets) != 1 || sets[0].PublishedOn != "2026-10-23T00:00:00" || sets[0].CardCount != 282 {
		t.Errorf("sets = %+v", sets)
	}
	for _, c := range cs {
		if c.Set.SetID != "RAD" {
			t.Errorf("%s from an unlisted set was kept", c.RiftboundID)
		}
	}
}

func TestStarterSuffixIsDropped(t *testing.T) {
	rows := []row{
		{ID: "OGS-017", NameNormal: "Annie - Dark Child - Starter", Rarity: "Rare", Type: []string{"Legend"}},
		{ID: "OGS-017-P", Name: "Annie - Dark Child - Starter (Nexus Night Promo)", NameNormal: "Annie - Dark Child - Starter", Rarity: "Rare", Type: []string{"Legend"}, Promo: "1"},
		{ID: "OGS-024", NameNormal: "Decisive Strike", Rarity: "Epic", Type: []string{"Spell"}},
	}
	got := convertAll(t, rows)
	if n := got["OGS-017"].Name; n != "Annie - Dark Child" {
		t.Errorf("name = %q, want the suffix dropped", n)
	}
	if n := got["OGS-017-P"].Name; n != "Annie - Dark Child (Nexus Night Promo)" {
		t.Errorf("promo name = %q, want the suffix dropped", n)
	}
}

func TestFlexReadsStringsNumbersAndNull(t *testing.T) {
	var got struct{ A, B, C flex }
	if err := json.Unmarshal([]byte(`{"A":"2","B":3,"C":null}`), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.A != "2" || got.B != "3" || got.C != "" {
		t.Errorf("got %+v", got)
	}
}
