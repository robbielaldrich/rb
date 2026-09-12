package riftcodex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func raw(t *testing.T, riftboundID, setID, name string) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"riftbound_id": riftboundID,
		"name":         name,
		"set":          map[string]string{"set_id": setID},
	})
	if err != nil {
		t.Fatalf("failed to build a card: %v", err)
	}
	return data
}

// The Vendetta promo of a card, and the Vendetta printing it reprints, are two
// different printings: only the second is in the dump.
func TestMergeMissingAddsCardsTheDumpLacks(t *testing.T) {
	dump := []json.RawMessage{raw(t, "ven-069-166", "VEN", "Mel, Newly Awakened")}
	extra := []json.RawMessage{raw(t, "opp-069b-166", "OPP", "Mel, Newly Awakened (Nexus Night Promo)")}

	got, err := mergeMissing(dump, extra, "missing.json")
	if err != nil {
		t.Fatalf("mergeMissing: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("merged %d cards, want 2", len(got))
	}
}

// The file exists only to cover what Riftcodex lacks, so an entry it has
// caught up with stops the download rather than being quietly duplicated.
func TestMergeMissingRefusesCardsTheDumpNowCarries(t *testing.T) {
	dump := []json.RawMessage{raw(t, "opp-069b-166", "OPP", "Mel, Newly Awakened")}
	extra := []json.RawMessage{raw(t, "opp-069b-166", "OPP", "Mel, Newly Awakened (Nexus Night Promo)")}

	_, err := mergeMissing(dump, extra, "cards/missing.json")
	if err == nil {
		t.Fatal("mergeMissing accepted a card the dump already carries")
	}
	for _, want := range []string{"opp-069b-166", "cards/missing.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// The id each entry carries is a guess at the one Riftcodex will assign, so a
// promo that turns up under a different collector number is still caught: the
// set, the card and the set being reprinted are what identify it.
func TestMergeMissingCatchesAPromoNumberedDifferently(t *testing.T) {
	dump := []json.RawMessage{raw(t, "opp-241-166", "OPP", "Mel, Newly Awakened")}
	extra := []json.RawMessage{raw(t, "opp-069b-166", "OPP", "Mel, Newly Awakened (Nexus Night Promo)")}

	if _, err := mergeMissing(dump, extra, "missing.json"); err == nil {
		t.Fatal("mergeMissing accepted a promo the dump carries under another number")
	}
}

// A promo of the same card from a different set is a different printing: OPP's
// Origins Fury Rune doesn't stand in for the Vendetta one.
func TestMergeMissingSeparatesPromosBySetReprinted(t *testing.T) {
	dump := []json.RawMessage{
		raw(t, "ven-r01", "VEN", "Fury Rune"),
		raw(t, "opp-007b-298", "OPP", "Fury Rune"),
	}
	extra := []json.RawMessage{raw(t, "opp-r01b-166", "OPP", "Fury Rune (Nexus Night Promo)")}

	if _, err := mergeMissing(dump, extra, "missing.json"); err != nil {
		t.Fatalf("mergeMissing rejected a Vendetta promo over an Origins one: %v", err)
	}
}

func TestLoadMissingWithoutAFileAddsNothing(t *testing.T) {
	got, err := loadMissing(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("loadMissing: %v", err)
	}
	if got != nil {
		t.Errorf("loadMissing returned %d cards, want none", len(got))
	}
}

func TestLoadMissingReadsTheCards(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	body := `{"note":"why these are here","cards":[{"riftbound_id":"ven-r01b","name":"Fury Rune (Nexus Night Promo)"}]}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("failed to write the file: %v", err)
	}

	got, err := loadMissing(path)
	if err != nil {
		t.Fatalf("loadMissing: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("loadMissing returned %d cards, want 1", len(got))
	}
}

// The checked-in file has to stay loadable and stay absent from the dataset it
// supplements, since a typo in either would only surface on the next download.
func TestCheckedInMissingFileIsUsable(t *testing.T) {
	cards, err := loadMissing("../cards/missing-from-riftcodex.json")
	if err != nil {
		t.Fatalf("loadMissing: %v", err)
	}
	if len(cards) == 0 {
		t.Skip("no cards are currently missing from Riftcodex")
	}

	for _, raw := range cards {
		var c cardID
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatalf("failed to parse a card: %v", err)
		}
		if c.RiftboundID == "" || c.Name == "" {
			t.Errorf("card %s is missing an id or a name", raw)
		}
	}
}
