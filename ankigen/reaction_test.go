package ankigen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/cards"
)

func reactionCard(id, name string, domains ...string) cards.Card {
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
		reactionCard("unl-001-219", "Lunar Boon", "Chaos"),
		reactionCard("unl-002-219", "Highlander", "Calm", "Body"),
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
	// Highlander counts once in Calm and once in Body; a non-Spell card with
	// the same shape of text is not a Reaction spell at all.
	if res.Notes != 3 {
		t.Fatalf("Notes = %d, want 3 (Body, Calm, Chaos)", res.Notes)
	}

	headers, rows := readDeck(t, res.DeckFile)
	wantHeaders := []string{
		"#separator:Tab", "#html:true", "#notetype:Basic",
		"#deck:Riftbound::Reaction Spells", "#tags column:3",
	}
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d note rows, want 3: %v", len(rows), rows)
	}

	// Rows sort by domain name: Body, Calm, Chaos.
	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if !strings.Contains(front, "Reaction spells in Body") {
		t.Errorf("front = %q, want it to ask about Body", front)
	}
	if !strings.Contains(back, "<li>Highlander</li>") {
		t.Errorf("back = %q, want Highlander listed", back)
	}
	if !strings.Contains(tags, "riftbound::domain::body") {
		t.Errorf("tags = %q", tags)
	}

	calmBack := rows[1][1]
	if !strings.Contains(calmBack, "<li>Highlander</li>") {
		t.Errorf("Calm back = %q, want Highlander listed there too", calmBack)
	}
}

func TestReactionSpellsCollapsePrintings(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCard("unl-001-219", "Lunar Boon", "Chaos"),
		reactionCard("unl-001a-219", "Lunar Boon (Alternate Art)", "Chaos"),
	})

	res, err := GenerateReactionSpells(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want the two printings collapsed into one domain note", len(rows))
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
