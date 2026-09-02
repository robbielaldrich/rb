package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyIdentifiesEveryRuling(t *testing.T) {
	rs, err := Load("rulings.json")
	if err != nil {
		t.Fatalf("Load(rulings.json) = %v", err)
	}

	seen := map[string]Ruling{}
	for _, r := range rs {
		if clash, ok := seen[r.Key()]; ok {
			t.Errorf("%q and %q share the key %s", clash.Question, r.Question, r.Key())
		}
		seen[r.Key()] = r
	}
}

func TestKeyIgnoresWording(t *testing.T) {
	r := Ruling{Slug: "alpha-strike", Anchor: "targets", Question: "How are targets chosen?"}
	reworded := r
	reworded.Question = "In what order are targets chosen?"

	if r.Key() != reworded.Key() {
		t.Errorf("rewording changed the key: %s -> %s", r.Key(), reworded.Key())
	}
	if want := "alpha-strike#targets"; r.Key() != want {
		t.Errorf("Key() = %s, want %s", r.Key(), want)
	}
}

// The Reddit notes carry no anchor to tell them apart, so their keys have to
// come from the question.
func TestKeyWithoutAnchor(t *testing.T) {
	a := Ruling{Slug: "reddit", Question: "Does damage lower a unit's Might?"}
	b := Ruling{Slug: "reddit", Question: "Does healing happen after a showdown?"}

	if a.Key() == b.Key() {
		t.Errorf("both notes keyed as %s", a.Key())
	}
	if a.Key() != (Ruling{Slug: "reddit", Question: a.Question}).Key() {
		t.Error("the same note keyed differently twice")
	}
}

func TestLoadRejectsAnEmptyDataset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rulings.json")
	if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("Load of an empty dataset succeeded")
	}
}
