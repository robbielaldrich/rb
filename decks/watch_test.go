package decks

import (
	"os"
	"strings"
	"testing"
)

// clipboard stands in for pbpaste, handing back whatever was last copied.
type clipboard struct{ text string }

func (c *clipboard) read() (string, error) { return c.text, nil }

func newWatcher(t *testing.T) (*watcher, *clipboard) {
	t.Helper()
	c := &clipboard{}
	return &watcher{
		path:      t.TempDir() + "/decks.json",
		pool:      testPool(nil),
		latestSet: "VEN",
		read:      c.read,
	}, c
}

func (w *watcher) mustPoll(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	if err := w.poll(&out); err != nil {
		t.Fatalf("poll: %v", err)
	}
	return out.String()
}

func TestWatchSavesACopiedDeck(t *testing.T) {
	w, c := newWatcher(t)
	c.text = pasted
	out := w.mustPoll(t)

	reg, err := loadRegistry(w.path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Decks) != 1 {
		t.Fatalf("registered %d decks, want the one copied", len(reg.Decks))
	}
	d := reg.Decks[0]
	if d.Name != "Kennen, Heart of the Tempest" {
		t.Errorf("deck is named %q, want its legend", d.Name)
	}
	if len(d.ID) != 8 {
		t.Errorf("deck ID is %q, want eight hex digits", d.ID)
	}
	if d.AddedAt.IsZero() || d.LatestSet != "VEN" {
		t.Errorf("deck was filed at %v under %q, want now under VEN", d.AddedAt, d.LatestSet)
	}
	// Lightning Rush is reprinted in Origins, which puts OGN among the sets;
	// the Chaos Rune is in VEN too, but runes don't count either way.
	if got := strings.Join(d.Sets, " "); got != "OGN VEN" {
		t.Errorf("deck's sets are %q, want OGN VEN", got)
	}
	if !strings.Contains(out, "saved Kennen, Heart of the Tempest · VEN · 17 cards, 3 in the sideboard · OGN VEN as "+d.ID) {
		t.Errorf("the watcher doesn't report what it saved:\n%s", out)
	}
}

func TestWatchFilesACopyOnce(t *testing.T) {
	w, c := newWatcher(t)
	c.text = pasted
	w.mustPoll(t)
	if out := w.mustPoll(t); out != "" {
		t.Errorf("the same clipboard read twice says %q, want nothing", out)
	}

	// Copying the same list again from somewhere else, after something in
	// between, finds it registered.
	c.text = "something else"
	w.mustPoll(t)
	c.text = strings.ReplaceAll(pasted, "\n\n", "\n")
	if out := w.mustPoll(t); !strings.Contains(out, "already registered") {
		t.Errorf("a list copied again says %q, want it found registered", out)
	}
	if w.added != 1 {
		t.Errorf("added %d decks, want the one", w.added)
	}
}

func TestWatchKeepsQuietAboutWhatIsntADeck(t *testing.T) {
	w, c := newWatcher(t)
	for _, text := range []string{"https://riftdecks.com/x", "3 pigs went to market", ""} {
		c.text = text
		if out := w.mustPoll(t); out != "" {
			t.Errorf("copying %q says %q, want nothing", text, out)
		}
	}
	if _, err := os.Stat(w.path); !os.IsNotExist(err) {
		t.Error("the register was written with nothing to save")
	}
}

func TestWatchSaysWhenADecklistWontParse(t *testing.T) {
	w, c := newWatcher(t)
	c.text = "3 Lightning Rush\n2 Star-Crossed\n1 Baron Nashor\n"
	if out := w.mustPoll(t); !strings.Contains(out, "couldn't read it") {
		t.Errorf("a list with no headings says %q, want it reported", out)
	}
}

// A deck renamed by hand while the watcher runs keeps its new name.
func TestWatchDoesntWriteOverAHandEdit(t *testing.T) {
	w, c := newWatcher(t)
	c.text = pasted
	w.mustPoll(t)

	reg, _ := loadRegistry(w.path)
	reg.Decks[0].Name = "worlds T1"
	if err := reg.save(w.path); err != nil {
		t.Fatal(err)
	}

	c.text = "MainDeck:\n1 Shadow\n"
	w.mustPoll(t)
	reg, _ = loadRegistry(w.path)
	if len(reg.Decks) != 2 || reg.Decks[0].Name != "worlds T1" {
		t.Errorf("register holds %+v, want the renamed deck and the new one", reg.Decks)
	}
}

func TestOldDecksAreGivenAnID(t *testing.T) {
	path := t.TempDir() + "/decks.json"
	write(t, path, `{"decks": [{"name": "a"}, {"name": "b", "id": "keepme00"}]}`)

	reg, err := loadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if reg.Decks[0].ID == "" || reg.Decks[0].ID == reg.Decks[1].ID {
		t.Errorf("first deck's ID is %q, want a fresh one", reg.Decks[0].ID)
	}
	if reg.Decks[1].ID != "keepme00" {
		t.Errorf("second deck's ID is %q, want the one it had", reg.Decks[1].ID)
	}
}
