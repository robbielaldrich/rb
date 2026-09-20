package ankigen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/cards"
)

// hiddenDomainCard is a Hidden card in the domains named, the shape this deck
// buckets on.
func hiddenDomainCard(id, name, setID string, domains ...string) cards.Card {
	c := hiddenCard(id, name)
	c.Set = cards.CardSet{SetID: setID}
	c.Classification.Domain = domains
	return c
}

func TestGenerateHiddenDomains(t *testing.T) {
	opts := fixture(t, []cards.Card{
		hiddenDomainCard("unl-001-219", "Bone Skewer", "unl", "Chaos"),
		hiddenDomainCard("ogn-002-298", "Tideturner", "ogn", "Chaos"),
		hiddenDomainCard("sfd-003-221", "Guards!", "sfd", "Order"),
	})

	res, err := GenerateHiddenDomains(opts)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}
	if res.Notes != 2 {
		t.Fatalf("Notes = %d, want one per domain (Chaos, Order)", res.Notes)
	}

	headers, rows := readDeck(t, res.DeckFile)
	wantHeaders := []string{
		"#separator:Tab", "#html:true", "#notetype:Basic",
		"#deck:Riftbound::Hidden by Domain", "#tags column:3",
	}
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d note rows, want 2: %v", len(rows), rows)
	}

	// Domains come in name order, and the cards within one in theirs.
	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if front != "Which Hidden cards are available to Chaos?" {
		t.Errorf("front = %q", front)
	}
	if !strings.Contains(back, "<b>2 cards</b>") {
		t.Errorf("back = %q, want it to count the roster", back)
	}
	if i, j := strings.Index(back, "Bone Skewer"), strings.Index(back, "Tideturner"); i < 0 || j < 0 || i > j {
		t.Errorf("back = %q, want both Chaos cards in name order", back)
	}
	// The set is named beside each card, a roster being the roster of a format.
	if !strings.Contains(back, "<i>(UNL)</i>") || !strings.Contains(back, "<i>(OGN)</i>") {
		t.Errorf("back = %q, want each card's set", back)
	}
	if !strings.Contains(tags, "riftbound::domain::chaos") {
		t.Errorf("tags = %q", tags)
	}

	// The scans are shown small, and reference the same media the other decks
	// use rather than a second copy.
	if !strings.Contains(back, "<img src='rb-unl-001-219.jpg' width='180'>") {
		t.Errorf("back = %q, want the card's scan shown as a thumbnail", back)
	}
	if _, err := os.Stat(filepath.Join(res.MediaDir, "rb-unl-001-219.jpg")); err != nil {
		t.Errorf("the scan was not written: %v", err)
	}
}

// A card of two domains is playable out of either, so it belongs to both
// rosters.
func TestHiddenDomainsListDualDomainCardsUnderBoth(t *testing.T) {
	opts := fixture(t, []cards.Card{
		hiddenDomainCard("ogn-001-298", "Fox-Fire", "ogn", "Mind", "Calm"),
		hiddenDomainCard("ogn-002-298", "Sprite Call", "ogn", "Mind"),
	})

	res, err := GenerateHiddenDomains(opts)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}

	_, rows := readDeck(t, res.DeckFile)
	backs := map[string]string{}
	for _, r := range rows {
		backs[r[0]] = r[1]
	}

	calm, ok := backs["Which Hidden cards are available to Calm?"]
	if !ok {
		t.Fatalf("no Calm note, got %v", backs)
	}
	if !strings.Contains(calm, "Fox-Fire") {
		t.Errorf("Calm back = %q, want Fox-Fire", calm)
	}
	if !strings.Contains(calm, "<b>1 card</b>") {
		t.Errorf("Calm back = %q, want a single-card roster counted in the singular", calm)
	}

	mind := backs["Which Hidden cards are available to Mind?"]
	if !strings.Contains(mind, "Fox-Fire") || !strings.Contains(mind, "Sprite Call") {
		t.Errorf("Mind back = %q, want both Mind cards", mind)
	}
}

// The question is a note's identity in Anki, so a rerun against a grown
// catalog has to leave it alone and rewrite only the roster behind it.
func TestHiddenDomainsKeepTheirQuestionAsSetsAreAdded(t *testing.T) {
	first := fixture(t, []cards.Card{
		hiddenDomainCard("ogn-001-298", "Tideturner", "ogn", "Chaos"),
	})
	res, err := GenerateHiddenDomains(first)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}
	_, rows := readDeck(t, res.DeckFile)
	before := rows[0][0]

	second := fixture(t, []cards.Card{
		hiddenDomainCard("ogn-001-298", "Tideturner", "ogn", "Chaos"),
		hiddenDomainCard("ven-002-166", "Spiderling", "ven", "Chaos"),
	})
	res, err = GenerateHiddenDomains(second)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}
	_, rows = readDeck(t, res.DeckFile)

	if rows[0][0] != before {
		t.Errorf("question changed from %q to %q, which would orphan the old note", before, rows[0][0])
	}
	if !strings.Contains(rows[0][1], "Spiderling") || !strings.Contains(rows[0][1], "<b>2 cards</b>") {
		t.Errorf("back = %q, want the new card folded into the roster", rows[0][1])
	}
}

func TestNoHiddenCardsIsAnErrorForDomains(t *testing.T) {
	opts := fixture(t, []cards.Card{
		{
			Name: "Plain Unit", RiftboundID: "unl-009-219", CollectorNumber: 9,
			Set:            cards.CardSet{SetID: "unl"},
			Classification: cards.Classification{Type: "Unit", Domain: []string{"Fury"}},
			Text:           cards.Text{Plain: "I enter ready."},
		},
	})

	if _, err := GenerateHiddenDomains(opts); err == nil {
		t.Fatal("GenerateHiddenDomains accepted a catalog with no Hidden cards")
	}
}
