package ankigen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func staleLines(t *testing.T, dir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, staleFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}

	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

// A question the deck stops asking is recorded, since re-importing can't
// remove the note it left in the collection.
func TestWriteRecordsDroppedQuestions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.txt")

	before := deck{name: "Riftbound::Test", notetype: "Basic", notes: []note{
		{front: "kept question", back: "a"},
		{front: "dropped question", back: "b"},
	}}
	if err := before.write(path); err != nil {
		t.Fatal(err)
	}
	if lines := staleLines(t, dir); len(lines) != 0 {
		t.Fatalf("a first run recorded %v, want nothing dropped", lines)
	}

	after := deck{name: "Riftbound::Test", notetype: "Basic", notes: []note{
		{front: "kept question", back: "a"},
	}}
	if err := after.write(path); err != nil {
		t.Fatal(err)
	}

	lines := staleLines(t, dir)
	if len(lines) != 1 {
		t.Fatalf("recorded %v, want just the dropped question", lines)
	}
	if !strings.HasSuffix(lines[0], "\tRiftbound::Test\tdropped question") {
		t.Errorf("line = %q, want the deck and question tab-separated after a date", lines[0])
	}
}

// Running again without the catalog changing must not list the same question
// twice.
func TestStaleQuestionsAreNotRecordedTwice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.txt")

	full := deck{name: "Riftbound::Test", notetype: "Basic", notes: []note{
		{front: "one", back: "a"}, {front: "two", back: "b"},
	}}
	if err := full.write(path); err != nil {
		t.Fatal(err)
	}

	trimmed := deck{name: "Riftbound::Test", notetype: "Basic", notes: []note{{front: "one", back: "a"}}}
	for range 3 {
		if err := trimmed.write(path); err != nil {
			t.Fatal(err)
		}
	}

	if lines := staleLines(t, dir); len(lines) != 1 {
		t.Fatalf("recorded %v after three identical runs, want one line", lines)
	}
}

// A new set can make a dropped question worth asking again, and the run that
// starts asking it takes it off the list.
func TestStaleQuestionComingBackIsCleared(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.txt")

	full := deck{name: "Riftbound::Test", notetype: "Basic", notes: []note{
		{front: "one", back: "a"}, {front: "two", back: "b"},
	}}
	if err := full.write(path); err != nil {
		t.Fatal(err)
	}
	trimmed := deck{name: "Riftbound::Test", notetype: "Basic", notes: []note{{front: "one", back: "a"}}}
	if err := trimmed.write(path); err != nil {
		t.Fatal(err)
	}
	if lines := staleLines(t, dir); len(lines) != 1 {
		t.Fatalf("recorded %v, want the dropped question listed", lines)
	}

	if err := full.write(path); err != nil {
		t.Fatal(err)
	}
	if lines := staleLines(t, dir); len(lines) != 0 {
		t.Errorf("recorded %v, want the returning question cleared", lines)
	}
}

// Every deck written into one directory shares the list, so a run of one must
// not clear another's lines.
func TestStaleListIsSharedBetweenDecks(t *testing.T) {
	dir := t.TempDir()
	costs := filepath.Join(dir, "costs.txt")
	effects := filepath.Join(dir, "effects.txt")

	for _, d := range []deck{
		{name: "Riftbound::Costs", notetype: "Basic", notes: []note{{front: "cost one"}, {front: "cost two"}}},
	} {
		if err := d.write(costs); err != nil {
			t.Fatal(err)
		}
	}
	if err := (&deck{name: "Riftbound::Effects", notetype: "Basic", notes: []note{{front: "effect one"}, {front: "effect two"}}}).write(effects); err != nil {
		t.Fatal(err)
	}

	if err := (&deck{name: "Riftbound::Costs", notetype: "Basic", notes: []note{{front: "cost one"}}}).write(costs); err != nil {
		t.Fatal(err)
	}
	if err := (&deck{name: "Riftbound::Effects", notetype: "Basic", notes: []note{{front: "effect one"}}}).write(effects); err != nil {
		t.Fatal(err)
	}

	lines := staleLines(t, dir)
	if len(lines) != 2 {
		t.Fatalf("recorded %v, want one dropped question from each deck", lines)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Riftbound::Costs\tcost two") || !strings.Contains(joined, "Riftbound::Effects\teffect two") {
		t.Errorf("recorded %v, want both decks' dropped questions", lines)
	}
}
