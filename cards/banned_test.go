package cards

import "testing"

// A misspelt entry would ban nothing and fail silently, so every name on the
// list has to match a card the catalog actually holds.
func TestBannedNamesAreInTheCatalog(t *testing.T) {
	cs, err := Load("cards.json")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, c := range cs {
		if c.IsBanned() {
			found[c.BaseName()] = true
		}
	}
	for name := range banned {
		if !found[name] {
			t.Errorf("%q is banned but no card in the catalog has that name", name)
		}
	}
}

func TestLegalDropsEveryPrintingOfABannedCard(t *testing.T) {
	cs := []Card{
		{Name: "Fight or Flight"},
		{Name: "Draven - Vanquisher (Alternate Art)"},
		{Name: "Teemo - Scout"},
	}
	got := Legal(cs)
	if len(got) != 1 || got[0].Name != "Teemo - Scout" {
		t.Errorf("Legal = %v, want only Teemo - Scout", got)
	}
}
