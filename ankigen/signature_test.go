package ankigen

import (
	"strings"
	"testing"

	"rb/cards"
)

// legendCard is a legend at a collector number, tagged with its champion.
func legendCard(id, name, setID string, number int, tags ...string) cards.Card {
	return cards.Card{
		Name: name, RiftboundID: id, CollectorNumber: number,
		Set:            cards.CardSet{SetID: setID},
		Classification: cards.Classification{Type: cards.TypeLegend, Domain: []string{"Fury"}},
		Tags:           tags,
	}
}

// signatureCard is a champion's signature card, printed at the number after
// their legend the way the sets lay them out.
func signatureCard(id, name, setID string, number int, cardType string, tags ...string) cards.Card {
	sig := cards.SupertypeSignature
	return cards.Card{
		Name: name, RiftboundID: id, CollectorNumber: number,
		Set:            cards.CardSet{SetID: setID},
		Classification: cards.Classification{Type: cardType, Supertype: &sig, Domain: []string{"Fury"}},
		Tags:           tags,
		Text:           cards.Text{Plain: "[Action] (Play on your turn or in showdowns.)Deal 2."},
	}
}

func TestGenerateSignatureCards(t *testing.T) {
	opts := fixture(t, []cards.Card{
		legendCard("unl-001-219", "Ahri - Nine-Tailed Fox", "unl", 1, "Ahri"),
		signatureCard("unl-002-219", "Fox-Fire", "unl", 2, "Spell", "Ahri"),
	})

	res, err := GenerateSignatureCards(opts)
	if err != nil {
		t.Fatalf("GenerateSignatureCards: %v", err)
	}
	if res.Notes != 1 {
		t.Fatalf("Notes = %d, want one per legend", res.Notes)
	}

	_, rows := readDeck(t, res.DeckFile)
	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if front != "What is the signature card of Ahri - Nine-Tailed Fox?" {
		t.Errorf("front = %q", front)
	}
	if !strings.Contains(back, "<b>1 card</b>") || !strings.Contains(back, "Fox-Fire") {
		t.Errorf("back = %q, want a counted roster naming Fox-Fire", back)
	}
	if !strings.Contains(back, "<img src='rb-unl-002-219.jpg' width='180'>") {
		t.Errorf("back = %q, want the signature card's scan", back)
	}
	if !strings.Contains(tags, "riftbound::signature-cards") {
		t.Errorf("tags = %q", tags)
	}
}

// A signature card is tied to its champion rather than to one printing of
// their legend, so a champion with two legends shares the cards between both.
func TestSignatureCardsAreSharedBetweenALegendsPrintings(t *testing.T) {
	opts := fixture(t, []cards.Card{
		legendCard("ogs-019-024", "Master Yi - Wuju Bladesman", "ogs", 19, "Master Yi"),
		signatureCard("ogs-020-024", "Highlander", "ogs", 20, "Spell", "Master Yi"),
		legendCard("unl-191-219", "Master Yi - Wuju Master", "unl", 191, "Master Yi"),
		signatureCard("unl-192-219", "Alpha Strike", "unl", 192, "Spell", "Master Yi"),
	})

	res, err := GenerateSignatureCards(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 2 {
		t.Fatalf("Notes = %d, want a note for each of the two legends", res.Notes)
	}

	_, rows := readDeck(t, res.DeckFile)
	for _, r := range rows {
		if !strings.Contains(r[1], "Alpha Strike") || !strings.Contains(r[1], "Highlander") {
			t.Errorf("%q answers %q, want both of the champion's signature cards", r[0], r[1])
		}
	}
}

// Ornn brings three gears rather than a card, so the question asks for a card
// and the count on the answer does the correcting.
func TestALegendCanBringSeveralSignatureCards(t *testing.T) {
	opts := fixture(t, []cards.Card{
		legendCard("sfd-189-221", "Ornn - Fire Below the Mountain", "sfd", 189, "Ornn"),
		signatureCard("sfd-190-221", "Forgefire Cape", "sfd", 190, "Gear", "Ornn"),
		signatureCard("sfd-191-221", "Rabadon's Deathcrown", "sfd", 191, "Gear", "Ornn"),
		signatureCard("sfd-192-221", "Shurelya's Requiem", "sfd", 192, "Gear", "Ornn"),
	})

	res, err := GenerateSignatureCards(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)
	if len(rows) != 1 {
		t.Fatalf("got %d notes, want one", len(rows))
	}
	if !strings.Contains(rows[0][1], "<b>3 cards</b>") {
		t.Errorf("back = %q, want all three gears counted", rows[0][1])
	}
}

// A reprint of a legend is printed away from its signature card. Vendetta's
// overnumbered Jayce reaches the catalog with its champion prefix and its
// overnumbered flag both missing, so only where it sits tells it from a first
// printing — and it must not become a second note asking the same thing.
func TestReprintedLegendsAreNotAskedAgain(t *testing.T) {
	opts := fixture(t, []cards.Card{
		legendCard("ven-149-166", "Jayce - Defender of Tomorrow", "ven", 149, "Jayce"),
		signatureCard("ven-150-166", "Acceleration Gate", "ven", 150, "Spell", "Jayce"),
		legendCard("ven-194-166", "Defender of Tomorrow", "ven", 194, "Jayce"),
		legendCard("ven-195-166", "Mel - Soul's Reflection (Overnumbered)", "ven", 195, "Mel"),
	})

	res, err := GenerateSignatureCards(opts)
	if err != nil {
		t.Fatal(err)
	}
	_, rows := readDeck(t, res.DeckFile)
	if len(rows) != 1 {
		t.Fatalf("got %d notes, want only the first-printed Jayce: %v", len(rows), rows)
	}
	if rows[0][0] != "What is the signature card of Jayce - Defender of Tomorrow?" {
		t.Errorf("front = %q, want the legend printed beside its signature card", rows[0][0])
	}
}

// Signature cards are never a general answer, so the speed decks leave them
// out: only a deck whose legend is that champion may run one.
func TestSpeedCardsExcludeSignatureCards(t *testing.T) {
	sig := cards.SupertypeSignature
	plain := cards.Card{
		Name: "Gust", RiftboundID: "unl-010-219", CollectorNumber: 10,
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Spell", Domain: []string{"Fury"}},
		Text:           cards.Text{Plain: "[Reaction] (Play any time.)Deal 1."},
	}
	signature := plain
	signature.Name, signature.RiftboundID = "Fox-Fire", "unl-011-219"
	signature.Classification.Supertype = &sig

	got := speedCards([]cards.Card{plain, signature}, "Reaction")
	if len(got) != 1 || got[0].BaseName() != "Gust" {
		t.Errorf("speedCards returned %v, want only the card any deck can run", got)
	}
}

// Guarded against the real catalog, where the legends with more than one
// signature card and the mis-flagged reprints actually live.
func TestEveryLegendInTheCatalogHasASignatureCard(t *testing.T) {
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}

	legends := firstPrintedLegends(cs)
	if len(legends) < 40 {
		t.Fatalf("found %d first-printed legends, want the catalog's full roster", len(legends))
	}

	signatures := signatureCards(cs)
	for _, l := range legends {
		if len(signaturesOf(signatures, l)) == 0 {
			t.Errorf("%s brings no signature card", l.BaseName())
		}
	}

	// The bare "Defender of Tomorrow" is Vendetta's overnumbered Jayce with
	// its name and flag lost in the catalog; asking about it would repeat the
	// first printing's question.
	for _, l := range legends {
		if l.BaseName() == "Defender of Tomorrow" {
			t.Error("the mis-flagged Jayce reprint was taken for a legend of its own")
		}
	}
}
