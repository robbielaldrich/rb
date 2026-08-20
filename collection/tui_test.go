package collection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func typeKeys(t *testing.T, m *shoppingModel, keys ...string) {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "<enter>":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "<tab>":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		case "<esc>":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "<down>":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		default:
			for _, r := range k {
				m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
			}
			continue
		}
		m.Update(msg)
	}
}

func newTestModel(t *testing.T, path string) *shoppingModel {
	t.Helper()
	cs, err := loadCatalog("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}
	coll, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	return newModel(cs, coll, path)
}

func readColl(t *testing.T, path string) collection {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var c collection
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestAddQuantityThenAddCopies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.json")
	m := newTestModel(t, path)

	// Type a query; digits go into the query, not the list.
	typeKeys(t, m, "astral 44")
	if m.query != "astral 44" {
		t.Fatalf("query = %q, want %q", m.query, "astral 44")
	}
	if len(m.results) == 0 || m.results[0].card.Name != "Astral Heron" {
		t.Fatalf("top hit = %+v", m.results)
	}

	// Tab to the list, then 1 selects.
	typeKeys(t, m, "<tab>")
	if !m.focused {
		t.Fatal("tab did not focus the list")
	}
	typeKeys(t, m, "1")
	if m.mode != modeQuantity {
		t.Fatalf("mode = %v, want modeQuantity", m.mode)
	}

	// Quantity 2, enter.
	typeKeys(t, m, "2", "<enter>")
	if m.mode != modeSearch {
		t.Fatalf("mode = %v, want back to search; status=%q err=%v", m.mode, m.status, m.err)
	}
	got := readColl(t, path)
	if len(got.Cards) != 1 || got.Cards[0].Quantity != 2 {
		t.Fatalf("collection = %+v", got.Cards)
	}
	if got.Cards[0].Number != "44/166" || got.Cards[0].SetID != "VEN" {
		t.Fatalf("entry = %+v", got.Cards[0])
	}
	t.Logf("after first add: %+v", got.Cards[0])

	// Same card again: enter takes the top hit, bare enter means one copy,
	// then the conflict prompt appears.
	typeKeys(t, m, "astral 44", "<enter>")
	if m.mode != modeQuantity {
		t.Fatalf("mode = %v, want modeQuantity", m.mode)
	}
	typeKeys(t, m, "<enter>")
	if m.mode != modeConflict {
		t.Fatalf("mode = %v, want modeConflict (owned=%d)", m.mode, m.picked.owned)
	}
	typeKeys(t, m, "a") // add copies
	got = readColl(t, path)
	if got.Cards[0].Quantity != 3 {
		t.Fatalf("after add copies, quantity = %d, want 3", got.Cards[0].Quantity)
	}
	t.Logf("after add copies: qty=%d status=%q", got.Cards[0].Quantity, m.status)
}

func TestReplaceQuantity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.json")
	m := newTestModel(t, path)

	typeKeys(t, m, "astral 44", "<enter>", "5", "<enter>")
	if q := readColl(t, path).Cards[0].Quantity; q != 5 {
		t.Fatalf("quantity = %d, want 5", q)
	}

	typeKeys(t, m, "astral 44", "<enter>", "2", "<enter>")
	if m.mode != modeConflict {
		t.Fatalf("mode = %v, want modeConflict", m.mode)
	}
	typeKeys(t, m, "r") // replace
	if q := readColl(t, path).Cards[0].Quantity; q != 2 {
		t.Fatalf("after replace, quantity = %d, want 2", q)
	}
	t.Logf("replace ok: %q", m.status)
}

func TestViewRenders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.json")
	m := newTestModel(t, path)
	typeKeys(t, m, "astral")
	out := m.View().Content
	if !strings.Contains(out, "Astral Heron") {
		t.Fatalf("view missing result:\n%s", out)
	}
	t.Logf("\n%s", out)
}
