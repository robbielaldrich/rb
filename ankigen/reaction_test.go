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

	return Options{
		CatalogPath:      catPath,
		OutDir:           filepath.Join(dir, "out"),
		ReactionDeckName: "Riftbound::Reaction Spells",
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
	// Body and Calm only ever see Highlander, at its own cost of 2: a
	// "1 Power or less" band would be empty for both and is skipped. Chaos
	// gets a band at Lunar Boon's own cost of 1 and again at 2, since 2 or
	// less still includes it. A non-Spell card sharing the reaction wording
	// isn't a Reaction spell at all.
	if res.Notes != 4 {
		t.Fatalf("Notes = %d, want 4 (Body@2, Calm@2, Chaos@1, Chaos@2)", res.Notes)
	}

	headers, rows := readDeck(t, res.DeckFile)
	wantHeaders := []string{
		"#separator:Tab", "#html:true", "#notetype:Basic",
		"#deck:Riftbound::Reaction Spells", "#tags column:3",
	}
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d note rows, want 4: %v", len(rows), rows)
	}

	// Rows sort by domain, then by rising Power threshold within it: Body@2,
	// Calm@2, Chaos@1, Chaos@2.
	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if !strings.Contains(front, "Body") || !strings.Contains(front, "cost 2 Power or less") {
		t.Errorf("front = %q, want it to ask about Body at 2 Power or less", front)
	}
	if !strings.Contains(back, "<li>Highlander</li>") {
		t.Errorf("back = %q, want Highlander listed", back)
	}
	if !strings.Contains(tags, "riftbound::domain::body") || !strings.Contains(tags, "riftbound::power::2") {
		t.Errorf("tags = %q", tags)
	}

	chaosLow := rows[2]
	if !strings.Contains(chaosLow[0], "Chaos") || !strings.Contains(chaosLow[0], "cost 1 Power or less") {
		t.Errorf("front = %q, want it to ask about Chaos at 1 Power or less", chaosLow[0])
	}
	if !strings.Contains(chaosLow[1], "<li>Lunar Boon</li>") {
		t.Errorf("back = %q, want Lunar Boon listed", chaosLow[1])
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
	if res.Notes != 3 {
		t.Fatalf("Notes = %d, want 3 (Fury@0, Fury@2, Mind@2)", res.Notes)
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
