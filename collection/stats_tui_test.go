package collection

import (
	"bytes"
	"strings"
	"testing"
)

// typing drives the viewer the way a terminal would, feeding decoded
// keystrokes one at a time. Named keys are written as "<esc>".
func (v *statsViewer) typing(t *testing.T, s string) {
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
		if _, err := v.handle(k); err != nil {
			t.Fatalf("handle(%v): %v", k, err)
		}
	}
}

func newTestStatsViewer(t *testing.T) *statsViewer {
	t.Helper()
	cs := statsCards()
	coll := &collection{Cards: []collectedCard{{RiftboundID: "ven-001-166", Quantity: 1}}}
	sets := summarise(cs, coll)

	var buf bytes.Buffer
	writeStats(&buf, sets)
	table := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	return &statsViewer{coll: coll, cards: cs, sets: sets, table: table, axis: AxisPlayset}
}

func TestStatsViewerOpensMissingForSelectedSet(t *testing.T) {
	v := newTestStatsViewer(t)
	v.typing(t, "p")
	if v.mode != statsMissing {
		t.Fatalf("mode = %v, want statsMissing", v.mode)
	}
	if !strings.Contains(v.title, "VEN") {
		t.Errorf("title = %q, want it to name the selected set", v.title)
	}
	if joined := strings.Join(v.missing, "\n"); !strings.Contains(joined, "Oasis Raider") {
		t.Errorf("missing report doesn't mention Oasis Raider:\n%s", joined)
	}
}

// On the set axis a card already owned drops out of the report, however far
// its playset is from full.
func TestStatsViewerSetAxisDropsOwnedCards(t *testing.T) {
	v := newTestStatsViewer(t)
	v.typing(t, "s")
	joined := strings.Join(v.missing, "\n")
	if strings.Contains(joined, "Astral Heron") {
		t.Errorf("set-axis missing still lists an owned card:\n%s", joined)
	}
	if !strings.Contains(joined, "Oasis Raider") {
		t.Errorf("set-axis missing dropped an unowned card:\n%s", joined)
	}
}

func TestStatsViewerEscReturnsToList(t *testing.T) {
	v := newTestStatsViewer(t)
	v.typing(t, "p<esc>")
	if v.mode != statsList {
		t.Fatalf("mode = %v after esc, want statsList", v.mode)
	}
}

// The row past the last set stands for the whole catalog.
func TestStatsViewerCursorPastLastRowIsEverything(t *testing.T) {
	v := newTestStatsViewer(t)
	v.cursor = len(v.sets)
	v.typing(t, "p")
	if v.title != "everything" {
		t.Errorf("title = %q, want everything", v.title)
	}
}

func TestStatsViewerQuits(t *testing.T) {
	v := newTestStatsViewer(t)
	quit, err := v.handle(key{r: 'q'})
	if err != nil || !quit {
		t.Errorf("handle('q') = %v, %v, want quit", quit, err)
	}
}
