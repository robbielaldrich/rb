package collection

import (
	"slices"
	"strings"
	"testing"

	"rb/cards"
)

func testCards() []cards.Card {
	return []cards.Card{
		{Name: "Astral Heron", RiftboundID: "ven-044-166", Set: cards.CardSet{SetID: "ven"},
			Classification: cards.Classification{Domain: []string{"Chaos"}}},
		{Name: "Fleetfeather Trapper", RiftboundID: "ven-021-166", Set: cards.CardSet{SetID: "ven"},
			Classification: cards.Classification{Domain: []string{"Fury", "Body"}}},
		{Name: "Defy", RiftboundID: "ogn-045-298", Set: cards.CardSet{SetID: "ogn"},
			Classification: cards.Classification{Domain: []string{"Chaos"}}},
		{Name: "Mischievous Marai", RiftboundID: "unl-003-219", Set: cards.CardSet{SetID: "unl"},
			Classification: cards.Classification{Domain: []string{"Mind"}}},
	}
}

func ids(cs []cards.Card) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.RiftboundID
	}
	return out
}

func TestNarrowFiltersOnSetsAndDomains(t *testing.T) {
	for _, tc := range []struct {
		filters   []string
		wantIDs   []string
		wantScope string
	}{
		{nil, []string{"ven-044-166", "ven-021-166", "ogn-045-298", "unl-003-219"}, ""},
		{[]string{"VEN"}, []string{"ven-044-166", "ven-021-166"}, "VEN"},
		{[]string{"ven"}, []string{"ven-044-166", "ven-021-166"}, "VEN"},
		{[]string{"chaos"}, []string{"ven-044-166", "ogn-045-298"}, "CHAOS"},
		{[]string{"ven", "chaos"}, []string{"ven-044-166"}, "VEN CHAOS"},
		// Either order narrows to the same cards.
		{[]string{"chaos", "ven"}, []string{"ven-044-166"}, "CHAOS VEN"},
		// A card counts as being in every one of its domains.
		{[]string{"body"}, []string{"ven-021-166"}, "BODY"},
	} {
		got, scope, err := narrow(testCards(), tc.filters)
		if err != nil {
			t.Fatalf("narrow(%v): %v", tc.filters, err)
		}
		if !slices.Equal(ids(got), tc.wantIDs) {
			t.Errorf("narrow(%v) = %v, want %v", tc.filters, ids(got), tc.wantIDs)
		}
		if scope != tc.wantScope {
			t.Errorf("narrow(%v) scope = %q, want %q", tc.filters, scope, tc.wantScope)
		}
	}
}

func TestNarrowRejectsUnknownLabel(t *testing.T) {
	_, _, err := narrow(testCards(), []string{"nope"})
	if err == nil {
		t.Fatal("want an error for a label that is neither a set nor a domain")
	}
	for _, want := range []string{`"NOPE"`, "OGN", "UNL", "VEN", "CHAOS", "MIND"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q doesn't mention %s", err, want)
		}
	}
}

func TestNarrowRejectsAnEmptyCombination(t *testing.T) {
	_, _, err := narrow(testCards(), []string{"unl", "chaos"})
	if err == nil {
		t.Fatal("want an error when no card is in both the set and the domain")
	}
}

func TestSetOnlySearchesThatSet(t *testing.T) {
	cs, _, err := narrow(testCards(), []string{"ven"})
	if err != nil {
		t.Fatalf("narrow(ven): %v", err)
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
