package ankigen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/cards"
)

func reactionCard(id, name string, power int, domains ...string) cards.Card {
	return cards.Card{
		Name: name, RiftboundID: id, CollectorNumber: 3,
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Spell", Domain: domains},
		Attributes:     cards.Attributes{Power: &power},
		Text:           cards.Text{Plain: "[Reaction] (Play any time, even before spells and abilities resolve.)Discard 1, then draw 2."},
	}
}

// reactionCardNoPower is a Reaction spell whose Power attribute the API left
// null, the way most Reaction spells actually come back.
func reactionCardNoPower(id, name string, domains ...string) cards.Card {
	return cards.Card{
		Name: name, RiftboundID: id, CollectorNumber: 3,
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Spell", Domain: domains},
		Text:           cards.Text{Plain: "[Reaction] (Play any time, even before spells and abilities resolve.)Discard 1, then draw 2."},
	}
}

func reactionOpts(t *testing.T, cs []cards.Card) Options {
	t.Helper()
	dir := t.TempDir()

	data, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	catPath := filepath.Join(dir, "cards.json")
	if err := os.WriteFile(catPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	imgDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		writeScan(t, filepath.Join(imgDir, c.RiftboundID+".png"))
	}

	return Options{
		CatalogPath:      catPath,
		ImageDir:         imgDir,
		OutDir:           filepath.Join(dir, "out"),
		ReactionDeckName: "Riftbound::Reaction Spells",
		ImageWidth:       60,
	}
}

func TestGenerateReactionSpells(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCard("unl-001-219", "Lunar Boon", 1, "Chaos"),
		reactionCard("unl-002-219", "Highlander", 2, "Calm", "Body"),
		{
			Name: "Discipline", RiftboundID: "unl-003-219", CollectorNumber: 3,
			Set:            cards.CardSet{SetID: "unl"},
			Classification: cards.Classification{Type: "Unit", Domain: []string{"Calm"}},
			Text:           cards.Text{Plain: "I enter ready."},
		},
	})

	res, err := GenerateReactionSpells(opts)
	if err != nil {
		t.Fatalf("GenerateReactionSpells: %v", err)
	}
	// Each domain gets its roster. No band survives beside one here: Body and
	// Calm hold only Highlander, and Chaos only Lunar Boon, so every band is
	// either empty or the whole roster over again. A non-Spell card sharing
	// the reaction wording isn't a Reaction spell at all.
	if res.Notes != 3 {
		t.Fatalf("Notes = %d, want one roster each for Body, Calm and Chaos", res.Notes)
	}

	headers, rows := readDeck(t, res.DeckFile)
	wantHeaders := []string{
		"#separator:Tab", "#html:true", "#notetype:Basic",
		"#deck:Riftbound::Reaction Spells", "#tags column:3",
	}
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}

	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if front != "What are all the Reaction spells in Body?" {
		t.Errorf("front = %q", front)
	}
	if !strings.Contains(back, "<b>1 card</b>") || !strings.Contains(back, "Highlander") {
		t.Errorf("back = %q, want a counted roster holding Highlander", back)
	}
	// The back is the shared roster: the set beside each name, and the scan.
	if !strings.Contains(back, "<i>(UNL)</i>") {
		t.Errorf("back = %q, want the set named", back)
	}
	if !strings.Contains(back, "<img src='rb-unl-002-219.jpg' width='180'>") {
		t.Errorf("back = %q, want the scan shown as a thumbnail", back)
	}
	if !strings.Contains(tags, "riftbound::domain::body") {
		t.Errorf("tags = %q", tags)
	}
}

// A band is worth a note only where it narrows the roster. Mind holds a spell
// at 1 and another at 3, so "1 or less" is a real question; Chaos holds one
// spell at 1, so every band is the roster over again and only the roster is
// asked.
func TestReactionSpellsKeepOnlyBandsThatNarrow(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCard("unl-001-219", "Quick Word", 1, "Mind"),
		reactionCard("unl-002-219", "Long Word", 3, "Mind"),
		reactionCard("unl-003-219", "Lunar Boon", 1, "Chaos"),
	})

	res, err := GenerateReactionSpells(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)

	var fronts []string
	for _, r := range rows {
		fronts = append(fronts, r[0])
	}
	want := []string{
		"What are all the Reaction spells in Chaos?",
		"What are all the Reaction spells in Mind?",
		"What are all the Reaction spells in Mind that cost 1 Power or less?",
	}
	if strings.Join(fronts, "\n") != strings.Join(want, "\n") {
		t.Errorf("fronts =\n%s\nwant\n%s", strings.Join(fronts, "\n"), strings.Join(want, "\n"))
	}
}

func TestReactionSpellsSkipEmptyPowerBuckets(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCardNoPower("unl-001-219", "Cheap Reflex", "Fury"),
		reactionCard("unl-002-219", "Costly Counter", 2, "Mind"),
	})

	res, err := GenerateReactionSpells(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)
	for _, row := range rows {
		if strings.Contains(row[0], "Mind") && strings.Contains(row[0], "cost 0 Power or less") {
			t.Errorf("Mind has no Reaction spell at 0 Power, want that bucket skipped, got row %v", row)
		}
	}
	// One roster each, and no band beside either: Fury's only spell is free
	// and Mind's only spell costs 2, so every band is empty or the whole
	// roster.
	if res.Notes != 2 {
		t.Fatalf("Notes = %d, want a roster each for Fury and Mind", res.Notes)
	}
}

func TestReactionSpellsCollapsePrintings(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCard("unl-001-219", "Lunar Boon", 1, "Chaos"),
		reactionCard("unl-001a-219", "Lunar Boon (Alternate Art)", 1, "Chaos"),
	})

	res, err := GenerateReactionSpells(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want the two printings collapsed into one note", len(rows))
	}
	if strings.Count(rows[0][1], "<li>") != 1 {
		t.Errorf("back = %q, want Lunar Boon listed once despite two printings", rows[0][1])
	}
}

func TestNoReactionSpellsIsAnError(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{{
		Name: "Arena Kingpin", RiftboundID: "unl-001-219",
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Unit"},
		Text:           cards.Text{Plain: "I enter ready."},
	}})
	if _, err := GenerateReactionSpells(opts); err == nil {
		t.Fatal("want an error when the catalog holds no Reaction spells")
	}
}
