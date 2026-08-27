package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"rb/cards"
)

// A set that prints no extras at all has to reach the page as null, not as a
// 0/0 it would have to render as "0%" — owning none of something is a
// different statement from there being none to own.
func TestAbsentPrintingsAreNull(t *testing.T) {
	cs := []cards.Card{printing("ven-001-166", "Astral Heron", "ven", cards.Metadata{})}
	got := row(summarise(cs, &collection{})[0])

	if got.Cards != (tallyData{Owned: 0, Total: 1}) {
		t.Errorf("cards = %+v, want 0/1", got.Cards)
	}
	for _, c := range []struct {
		name  string
		tally *tallyData
	}{
		{"alternate_art", got.AlternateArt},
		{"overnumbered", got.Overnumbered},
		{"signature", got.Signature},
	} {
		if c.tally != nil {
			t.Errorf("%s = %+v, want null", c.name, *c.tally)
		}
	}
}

func TestTotalRowAddsTheSetsUp(t *testing.T) {
	cs := []cards.Card{
		printing("ven-001-166", "Astral Heron", "ven", cards.Metadata{}),
		printing("unl-001-219", "Mesmerize", "unl", cards.Metadata{}),
	}
	total := row(totals(summarise(cs, &collection{
		Cards: []collectedCard{{RiftboundID: "unl-001-219", Quantity: 1}},
	})))

	if total.Cards != (tallyData{Owned: 1, Total: 2}) {
		t.Errorf("total cards = %+v, want 1/2", total.Cards)
	}
	// The row stands for every set at once, so it carries no name of its own.
	if total.SetID != "" || total.Label != "" {
		t.Errorf("total row named %q %q, want no name", total.SetID, total.Label)
	}
}

func TestWriteStatsDataRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	if err := writeStatsData(path, summarise(statsCards(), &collection{
		Cards: []collectedCard{{RiftboundID: "ven-001-166", Quantity: 2}},
	})); err != nil {
		t.Fatalf("failed to write the summary data: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read the summary data back: %v", err)
	}
	var got statsData
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal the summary data: %v", err)
	}

	if len(got.Sets) != 1 || got.Sets[0].SetID != "VEN" {
		t.Fatalf("sets = %+v, want the one VEN row", got.Sets)
	}
	if got.Sets[0].Label != "Vendetta" {
		t.Errorf("label = %q, want Vendetta", got.Sets[0].Label)
	}
	// The same 1/3 the text report puts in its "all" row.
	if got.Total.Cards != (tallyData{Owned: 1, Total: 3}) {
		t.Errorf("total cards = %+v, want 1/3", got.Total.Cards)
	}
}
