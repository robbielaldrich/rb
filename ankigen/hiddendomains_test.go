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
	if res.Notes != 3 {
		t.Fatalf("Notes = %d, want one per card in each domain", res.Notes)
	}

	headers, rows := readDeck(t, res.DeckFile)
	checkHeaders(t, headers)
	for _, r := range rows {
		if r[3] != "Riftbound::Hidden by Domain" || r[4] != "Cloze" {
			t.Errorf("deck, notetype = %q, %q, want Riftbound::Hidden by Domain, Cloze", r[3], r[4])
		}
	}

	// Domains come in name order, and the cards within one in theirs.
	want := []string{"Chaos: Bone Skewer", "Chaos: Tideturner", "Order: Guards!"}
	if got := blanks(rows); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("blanks = %v, want %v", got, want)
	}

	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if front != "<b>Hidden</b> cards available to <b>Chaos</b>:<ul><li>{{c1::Bone Skewer <i>(UNL)</i>}}</li><li>Tideturner <i>(OGN)</i></li></ul>" {
		t.Errorf("front = %q, want the Chaos list with Bone Skewer blanked", front)
	}
	if !strings.Contains(tags, "riftbound::domain::chaos") || !strings.Contains(tags, "riftbound::card::bone-skewer") {
		t.Errorf("tags = %q", tags)
	}

	// The scan references the same media the other decks use rather than a
	// second copy.
	if back != "<img src='rb-unl-001-219.jpg' width='250'>" {
		t.Errorf("back = %q, want the blanked card's scan", back)
	}
	if _, err := os.Stat(filepath.Join(res.MediaDir, "rb-unl-001-219.jpg")); err != nil {
		t.Errorf("the scan was not written: %v", err)
	}
}

// A card of two domains is playable out of either, so it belongs to both
// lists, and is a note in each.
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
	want := []string{"Calm: Fox-Fire", "Mind: Fox-Fire", "Mind: Sprite Call"}
	if got := blanks(rows); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("blanks = %v, want %v", got, want)
	}
	if rows[0][5] == rows[1][5] {
		t.Errorf("both Fox-Fire notes have GUID %q, want one per domain", rows[0][5])
	}
}

// A note's GUID is its identity in Anki, so a rerun against a grown catalog has
// to keep it and rewrite only the list around the card.
func TestHiddenDomainsKeepTheirGUIDAsSetsAreAdded(t *testing.T) {
	first := fixture(t, []cards.Card{
		hiddenDomainCard("ogn-001-298", "Tideturner", "ogn", "Chaos"),
	})
	res, err := GenerateHiddenDomains(first)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}
	_, rows := readDeck(t, res.DeckFile)
	before := rows[0][5]

	second := fixture(t, []cards.Card{
		hiddenDomainCard("ogn-001-298", "Tideturner", "ogn", "Chaos"),
		hiddenDomainCard("ven-002-166", "Spiderling", "ven", "Chaos"),
	})
	res, err = GenerateHiddenDomains(second)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}
	_, rows = readDeck(t, res.DeckFile)

	var tide []string
	for _, r := range rows {
		if strings.Contains(r[2], "riftbound::card::tideturner") {
			tide = r
		}
	}
	if tide == nil {
		t.Fatalf("no Tideturner note in %v", blanks(rows))
	}
	if tide[5] != before {
		t.Errorf("GUID changed from %q to %q, which would orphan the old note", before, tide[5])
	}
	if !strings.Contains(tide[0], "<li>Spiderling <i>(VEN)</i></li>") {
		t.Errorf("front = %q, want the new card shown in the list", tide[0])
	}
}

// A signature card belongs to a legend, not a domain, so it is left off the
// list of every domain it is printed in.
func TestHiddenDomainsLeaveOutSignatureCards(t *testing.T) {
	sig := hiddenDomainCard("ogn-003-298", "Fox-Fire", "ogn", "Mind", "Calm")
	supertype := cards.SupertypeSignature
	sig.Classification.Supertype = &supertype
	opts := fixture(t, []cards.Card{
		sig,
		hiddenDomainCard("ogn-002-298", "Sprite Call", "ogn", "Mind"),
	})

	res, err := GenerateHiddenDomains(opts)
	if err != nil {
		t.Fatalf("GenerateHiddenDomains: %v", err)
	}

	_, rows := readDeck(t, res.DeckFile)
	if got := blanks(rows); len(got) != 1 || got[0] != "Mind: Sprite Call" {
		t.Fatalf("blanks = %v, want only Mind's Sprite Call", got)
	}
	if strings.Contains(rows[0][0], "Fox-Fire") {
		t.Errorf("front = %q, want the signature card left out", rows[0][0])
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
