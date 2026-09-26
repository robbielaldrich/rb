package ankigen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/cards"
)

func reactionCard(id, name string, energy int, domains ...string) cards.Card {
	return cards.Card{
		Name: name, RiftboundID: id, CollectorNumber: 3,
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Spell", Domain: domains},
		Attributes:     cards.Attributes{Energy: &energy},
		Text:           cards.Text{Plain: "[Reaction] (Play any time, even before spells and abilities resolve.)Discard 1, then draw 2."},
	}
}

// reactionCardNoEnergy is a Reaction card whose Energy attribute the API left
// null.
func reactionCardNoEnergy(id, name string, domains ...string) cards.Card {
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
		ReactionDeckName: "Riftbound::Reaction Cards",
		ActionDeckName:   "Riftbound::Action Cards",
		ImageWidth:       60,
	}
}

func actionCard(id, name string, energy int, domains ...string) cards.Card {
	c := reactionCard(id, name, energy, domains...)
	c.Text.Plain = "[Action] (Play on your turn, even while something is being resolved.)Draw 1."
	return c
}

func fronts(rows [][]string) []string {
	var out []string
	for _, r := range rows {
		out = append(out, r[0])
	}
	return out
}

// Each domain gets a band for every Energy cost from 1 to its dearest card,
// one per Energy cost the domain prints at, each holding only that cost.
func TestGenerateReactionCards(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCard("unl-001-219", "Zap", 1, "Body"),
		reactionCard("unl-002-219", "Anvil", 4, "Body"),
		reactionCard("unl-003-219", "Bash", 1, "Body"),
		reactionCardNoEnergy("unl-004-219", "Free Swing", "Body"),
		reactionCard("unl-005-219", "Lunar Boon", 2, "Chaos"),
		{
			Name: "Discipline", RiftboundID: "unl-006-219", CollectorNumber: 6,
			Set:            cards.CardSet{SetID: "unl"},
			Classification: cards.Classification{Type: "Unit", Domain: []string{"Calm"}},
			Text:           cards.Text{Plain: "I enter ready."},
		},
	})

	res, err := GenerateReactionCards(opts)
	if err != nil {
		t.Fatalf("GenerateReactionCards: %v", err)
	}

	headers, rows := readDeck(t, res.DeckFile)
	wantHeaders := []string{
		"#separator:Tab", "#html:true", "#notetype:Basic",
		"#deck:Riftbound::Reaction Cards", "#tags column:3",
	}
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}

	want := []string{
		// A free card is asked for at nothing rather than folded into every
		// band above it, and 2 and 3 go unasked because Body prints nothing
		// at either.
		"What are all the Reaction cards in Body that cost 0 Energy?",
		"What are all the Reaction cards in Body that cost 1 Energy?",
		"What are all the Reaction cards in Body that cost 4 Energy?",
		"What are all the Reaction cards in Chaos that cost 2 Energy?",
	}
	if strings.Join(fronts(rows), "\n") != strings.Join(want, "\n") {
		t.Fatalf("fronts =\n%s\nwant\n%s", strings.Join(fronts(rows), "\n"), strings.Join(want, "\n"))
	}

	free, one, four := rows[0], rows[1], rows[2]
	if !strings.Contains(free[1], "<b>1 card</b>") || !strings.Contains(free[1], "Free Swing") {
		t.Errorf("0-Energy back = %q, want the free card alone", free[1])
	}
	// The cheap cards are not repeated in the bands above them, which is the
	// whole of the change: each is read once, where it costs.
	if !strings.Contains(one[1], "<b>2 cards</b>") || strings.Contains(one[1], "Free Swing") {
		t.Errorf("1-Energy back = %q, want the two 1-Energy cards and nothing cheaper", one[1])
	}
	if !strings.Contains(four[1], "<b>1 card</b>") || !strings.Contains(four[1], "Anvil") {
		t.Errorf("4-Energy back = %q, want Anvil alone rather than the whole roster", four[1])
	}
	// Within a cost, by name.
	if strings.Index(one[1], "<li>Bash ") > strings.Index(one[1], "<li>Zap ") {
		t.Errorf("back = %q, want the two named in name order", one[1])
	}
	if !strings.Contains(four[1], "<i>(UNL)</i>") || !strings.Contains(four[1], "<img src='rb-unl-002-219.jpg' width='180'>") {
		t.Errorf("back = %q, want the set named and the scan shown", four[1])
	}
	if !strings.Contains(four[2], "riftbound::domain::body") || !strings.Contains(four[2], "riftbound::energy::4") {
		t.Errorf("tags = %q", four[2])
	}
}

func TestActionCardsShareThePathAndKeepToTheirOwnKeyword(t *testing.T) {
	gear := actionCard("unl-002-219", "Long Sword", 2, "Body")
	gear.Classification.Type = "Gear"
	legend := actionCard("unl-003-219", "Eye of Twilight", 0, "Body")
	legend.Classification.Type = cards.TypeLegend
	opts := reactionOpts(t, []cards.Card{
		actionCard("unl-001-219", "Void Seeker", 1, "Body"),
		gear,
		legend,
		reactionCard("unl-004-219", "Flash", 1, "Body"),
	})

	res, err := GenerateActionCards(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)
	want := []string{
		"What are all the Action cards in Body that cost 1 Energy?",
		"What are all the Action cards in Body that cost 2 Energy?",
	}
	if strings.Join(fronts(rows), "\n") != strings.Join(want, "\n") {
		t.Fatalf("fronts = %v, want %v", fronts(rows), want)
	}
	// The gear costs 2 and the spell 1, so each is now in its own band.
	if back := rows[1][1]; !strings.Contains(back, "Long Sword") {
		t.Errorf("2-Energy back = %q, want the gear as well as the spells", back)
	}
	if back := rows[0][1]; !strings.Contains(back, "Void Seeker") {
		t.Errorf("1-Energy back = %q, want the spell", back)
	}
	for _, r := range rows {
		if strings.Contains(r[1], "Flash") || strings.Contains(r[1], "Eye of Twilight") {
			t.Errorf("%q answers %q, want no Reaction card and no legend", r[0], r[1])
		}
	}
}

func TestSpeedCardsCollapsePrintings(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{
		reactionCard("unl-001-219", "Lunar Boon", 1, "Chaos"),
		reactionCard("unl-001a-219", "Lunar Boon (Alternate Art)", 1, "Chaos"),
	})

	res, err := GenerateReactionCards(opts)
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

func TestNoSpeedCardsIsAnError(t *testing.T) {
	opts := reactionOpts(t, []cards.Card{{
		Name: "Arena Kingpin", RiftboundID: "unl-001-219",
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Unit"},
		Text:           cards.Text{Plain: "I enter ready."},
	}})
	if _, err := GenerateReactionCards(opts); err == nil {
		t.Fatal("want an error when the catalog holds no Reaction cards")
	}
	if _, err := GenerateActionCards(opts); err == nil {
		t.Fatal("want an error when the catalog holds no Action cards")
	}
}
