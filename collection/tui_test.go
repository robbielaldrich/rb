package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/cards"
)

// typing drives the editor the way a terminal would, feeding decoded
// keystrokes one at a time. Named keys are written as "<enter>".
func (e *editor) typing(t *testing.T, s string) {
	t.Helper()
	for len(s) > 0 {
		var k key
		if strings.HasPrefix(s, "<") {
			end := strings.Index(s, ">")
			k, s = key{name: s[1:end]}, s[end+1:]
		} else {
			r := []rune(s)[0]
			k, s = key{r: r}, s[len(string(r)):]
		}
		if _, err := e.handle(k); err != nil {
			t.Fatalf("handle(%v): %v", k, err)
		}
	}
}

func newTestEditor(t *testing.T) *editor {
	t.Helper()
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}
	e := newEditor(&collection{}, cs, filepath.Join(t.TempDir(), "collection.json"))
	e.refresh()
	return e
}

func (e *editor) render(t *testing.T) string {
	t.Helper()
	lines, _, _ := e.frame(72)
	return strings.Join(lines, "\n")
}

func TestAddCardAndEditQuantity(t *testing.T) {
	e := newTestEditor(t)

	e.typing(t, "astral her")
	if len(e.results) == 0 {
		t.Fatal("no matches for \"astral her\"")
	}
	t.Logf("search:\n%s", e.render(t))

	e.typing(t, "1")
	if e.mode != modeQuantity {
		t.Fatalf("pressing 1 left mode %v, want quantity", e.mode)
	}
	if e.qty != "1" {
		t.Fatalf("qty = %q, want 1", e.qty)
	}
	t.Logf("quantity:\n%s", e.render(t))

	// A digit replaces the count the editor chose; the next appends to it.
	e.typing(t, "12")
	if e.qty != "12" {
		t.Fatalf("qty = %q, want 12", e.qty)
	}
	e.typing(t, "<up>")
	if e.qty != "13" {
		t.Fatalf("qty = %q after up, want 13", e.qty)
	}

	card := e.card
	e.typing(t, "<enter>")
	if e.mode != modeSearch || e.query != "" {
		t.Fatalf("enter left mode %v query %q, want search with an empty query", e.mode, e.query)
	}
	if !strings.Contains(e.status, "13") {
		t.Fatalf("status = %q, want it to report 13", e.status)
	}

	// The count is on disk already, and picking the card again offers one more.
	data, err := os.ReadFile(e.path)
	if err != nil {
		t.Fatalf("collection was not written: %v", err)
	}
	if !strings.Contains(string(data), card.RiftboundID) {
		t.Fatalf("collection file lacks %s:\n%s", card.RiftboundID, data)
	}
	e.typing(t, "astral her1")
	if e.card.RiftboundID != card.RiftboundID || e.qty != "14" {
		t.Fatalf("re-picking gave %s qty %q, want %s qty 14", e.card.RiftboundID, e.qty, card.RiftboundID)
	}
	t.Logf("re-picked:\n%s", e.render(t))
}

func TestQuantityModeTypingSearchesAgain(t *testing.T) {
	e := newTestEditor(t)
	e.typing(t, "astral<enter>")
	if e.mode != modeQuantity {
		t.Fatalf("mode = %v, want quantity", e.mode)
	}

	e.typing(t, "s")
	if e.mode != modeSearch {
		t.Fatalf("a letter left mode %v, want search", e.mode)
	}
	if e.query != "s" {
		t.Fatalf("query = %q, want the typed letter", e.query)
	}
}

func TestZeroQuantityRemovesCard(t *testing.T) {
	e := newTestEditor(t)
	e.typing(t, "astral<enter>0")
	if n := len(e.collection.Cards); n != 0 {
		t.Fatalf("collection holds %d cards, want none", n)
	}
	e.typing(t, "<enter>")
	if !strings.Contains(e.status, "removed") {
		t.Fatalf("status = %q, want it to report a removal", e.status)
	}
}

func TestNumberEntryEscapesTheSelectionKeys(t *testing.T) {
	e := newTestEditor(t)
	e.typing(t, "astral #44")
	if e.mode != modeSearch {
		t.Fatalf("digits after # left mode %v, want search", e.mode)
	}
	if e.query != "astral #44" {
		t.Fatalf("query = %q, want the number typed into it", e.query)
	}
	if len(e.results) == 0 || !e.results[0].numMatch {
		t.Fatalf("top match did not hit collector number 44: %s", e.render(t))
	}
	t.Logf("number search:\n%s", e.render(t))

	// A space closes the number, handing the digits back to selection.
	e.typing(t, " 1")
	if e.mode != modeQuantity {
		t.Fatalf("mode = %v after closing the number, want quantity", e.mode)
	}
}

func TestBackspaceAndWordDelete(t *testing.T) {
	e := newTestEditor(t)
	e.typing(t, "astral heron<backspace><backspace>")
	if e.query != "astral her" {
		t.Fatalf("query = %q, want \"astral her\"", e.query)
	}
	e.typing(t, "<ctrl+w>")
	if e.query != "astral " {
		t.Fatalf("query = %q after ctrl+w, want \"astral \"", e.query)
	}
	e.typing(t, "<ctrl+u>")
	if e.query != "" {
		t.Fatalf("query = %q after ctrl+u, want empty", e.query)
	}
}

func TestEscapeQuits(t *testing.T) {
	e := newTestEditor(t)
	if quit, _ := e.handle(key{name: "esc"}); !quit {
		t.Fatal("esc in search mode did not quit")
	}
	e.typing(t, "astral<enter>")
	quit, _ := e.handle(key{name: "esc"})
	if quit || e.mode != modeSearch {
		t.Fatalf("esc in quantity mode gave quit=%v mode=%v, want a return to search", quit, e.mode)
	}
}

func TestDecodeKeys(t *testing.T) {
	got := decodeKeys([]byte("ab\x1b[A\x1b\r\x7f\x1b[3~c"))
	want := []key{{r: 'a'}, {r: 'b'}, {name: "up"}, {name: "esc"}, {name: "enter"},
		{name: "backspace"}, {name: "unknown"}, {r: 'c'}}
	if len(got) != len(want) {
		t.Fatalf("decoded %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("key %d = %v, want %v (all: %v)", i, got[i], want[i], got)
		}
	}
}
