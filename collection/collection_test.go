package collection

import (
	"strings"
	"testing"

	"rb/cards"
)

func testCards() []cards.Card {
	return []cards.Card{
		{Name: "Astral Heron", RiftboundID: "ven-044-166", Set: cards.CardSet{SetID: "ven"}},
		{Name: "Defy", RiftboundID: "ogn-045-298", Set: cards.CardSet{SetID: "ogn"}},
		{Name: "Mischievous Marai", RiftboundID: "unl-003-219", Set: cards.CardSet{SetID: "unl"}},
	}
}

func TestFilterSetIsCaseInsensitive(t *testing.T) {
	got, err := filterSet(testCards(), "VEN")
	if err != nil {
		t.Fatalf("filterSet(VEN): %v", err)
	}
	if len(got) != 1 || got[0].RiftboundID != "ven-044-166" {
		t.Fatalf("filterSet(VEN) = %v, want just the Vendetta card", got)
	}
}

func TestFilterSetRejectsUnknownLabel(t *testing.T) {
	_, err := filterSet(testCards(), "nope")
	if err == nil {
		t.Fatal("want an error for a set nothing is printed in")
	}
	for _, want := range []string{`"nope"`, "OGN", "UNL", "VEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q doesn't mention %s", err, want)
		}
	}
}

func TestSetOnlySearchesThatSet(t *testing.T) {
	cs, err := filterSet(testCards(), "ven")
	if err != nil {
		t.Fatalf("filterSet(ven): %v", err)
	}

	e := newEditor(&collection{}, cs, t.TempDir()+"/collection.json", "ven")
	e.typing(t, "defy")
	if len(e.results) != 0 {
		t.Errorf("searching \"defy\" in VEN found %d matches, want none", len(e.results))
	}

	e.typing(t, "<ctrl+u>astral")
	if len(e.results) != 1 {
		t.Fatalf("searching \"astral\" in VEN found %d matches, want one", len(e.results))
	}

	if got := e.render(t); !strings.Contains(got, "VEN > astral") {
		t.Errorf("the prompt doesn't show the set:\n%s", got)
	}
}
