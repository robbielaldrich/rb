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

// Each domain gets a list for every Energy cost it prints at, each holding
// only that cost, and each card in a list gets a note blanking it out.
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
	checkHeaders(t, headers)
	for _, r := range rows {
		if r[3] != "Riftbound::Reaction Cards" || r[4] != "Cloze" {
			t.Errorf("deck, notetype = %q, %q, want Riftbound::Reaction Cards, Cloze", r[3], r[4])
		}
	}

	// A free card is asked for at nothing rather than folded into every list
	// above it, 2 and 3 go unasked because Body prints nothing at either, and
	// within a list the cards come in name order.
	want := []string{
		"Body 0: Free Swing",
		"Body 1: Bash", "Body 1: Zap",
		"Body 4: Anvil",
		"Chaos 2: Lunar Boon",
	}
	if got := blanks(rows); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("blanks =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// The Bash note shows the rest of its list, and nothing from the lists
	// beside it: each card is read once, where it costs.
	bash := rows[1]
	if !strings.HasPrefix(bash[0], "<b>Reaction</b> cards in <b>Body</b> that cost <b>1 Energy</b>:") {
		t.Errorf("front = %q, want the list named", bash[0])
	}
	if !strings.Contains(bash[0], "<li>{{c1::Bash <i>(UNL)</i>}}</li><li>Zap <i>(UNL)</i></li>") {
		t.Errorf("front = %q, want Bash blanked with its set and Zap shown", bash[0])
	}
	if strings.Contains(bash[0], "Free Swing") || strings.Contains(bash[0], "Anvil") {
		t.Errorf("front = %q, want only the 1-Energy cards", bash[0])
	}
	if bash[1] != "<img src='rb-unl-003-219.jpg' width='250'>" {
		t.Errorf("back = %q, want the blanked card's scan", bash[1])
	}
	for _, tag := range []string{"riftbound::reaction-cards", "riftbound::domain::body", "riftbound::energy::1", "riftbound::card::bash"} {
		if !strings.Contains(bash[2], tag) {
			t.Errorf("tags = %q, want %s", bash[2], tag)
		}
	}

	seen := map[string]bool{}
	for _, r := range rows {
		if r[5] == "" || seen[r[5]] {
			t.Errorf("guid = %q, want every note its own", r[5])
		}
		seen[r[5]] = true
	}
}

// blanks names each note by its list's domain and cost and the card it
// blanks, read back from its tags.
func blanks(rows [][]string) []string {
	var out []string
	for _, r := range rows {
		var domain, energy, card string
		for _, tag := range strings.Fields(r[2]) {
			if v, ok := strings.CutPrefix(tag, "riftbound::domain::"); ok {
				domain = strings.ToUpper(v[:1]) + v[1:]
			} else if v, ok := strings.CutPrefix(tag, "riftbound::energy::"); ok {
				energy = v
			}
		}
		if _, rest, ok := strings.Cut(r[0], "{{c1::"); ok {
			card, _, _ = strings.Cut(rest, " <i>(")
		}
		out = append(out, strings.TrimSpace(domain+" "+energy)+": "+card)
	}
	return out
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
	// The gear costs 2 and the spell 1, so each is in its own list, and
	// neither the Reaction card nor the legend is in either.
	want := []string{"Body 1: Void Seeker", "Body 2: Long Sword"}
	if got := blanks(rows); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("blanks = %v, want %v", got, want)
	}
	if !strings.HasPrefix(rows[0][0], "<b>Action</b> cards in <b>Body</b>") || !strings.Contains(rows[0][2], "riftbound::action-cards") {
		t.Errorf("front, tags = %q, %q, want an Action list", rows[0][0], rows[0][2])
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
	if strings.Count(rows[0][0], "<li>") != 1 {
		t.Errorf("front = %q, want Lunar Boon listed once despite two printings", rows[0][0])
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

// A unit whose Reaction keyword only marks a mana ability isn't a card the
// domain can answer with, so it stays off the roster.
func TestSpeedCardsLeaveOutManaAbilities(t *testing.T) {
	sage := reactionCard("unl-002-219", "Dragonsoul Sage", 2, "Body")
	sage.Classification.Type = "Unit"
	sage.Text.Plain = "[Reaction][&gt;] :rb_exhaust:: [Add] :rb_energy_1:. (Abilities that add resources can't be reacted to.)"
	// An activated Reaction ability that does something else still counts.
	guard := reactionCard("unl-003-219", "Watchful Guard", 2, "Body")
	guard.Classification.Type = "Unit"
	guard.Text.Plain = "[Reaction][&gt;] :rb_exhaust:: Give a unit +1 :rb_might: this turn. [Add] nothing."

	opts := reactionOpts(t, []cards.Card{
		sage, guard,
		reactionCard("unl-001-219", "Defy", 2, "Body"),
	})
	res, err := GenerateReactionCards(opts)
	if err != nil {
		t.Fatalf("GenerateReactionCards: %v", err)
	}

	_, rows := readDeck(t, res.DeckFile)
	want := []string{"Body 2: Defy", "Body 2: Watchful Guard"}
	if got := blanks(rows); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("blanks = %v, want %v, the mana ability left out", got, want)
	}
}
