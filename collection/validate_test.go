package collection

import (
	"path/filepath"
	"strings"
	"testing"
)

// typing drives the validator the same way it drives the editor.
func (v *validator) typing(t *testing.T, s string) {
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

func newTestValidator(t *testing.T) *validator {
	t.Helper()
	coll := &collection{Cards: []collectedCard{
		{RiftboundID: "ven-001-166", Name: "Baccai Sandspinner", Number: "1/166", SetID: "VEN", Quantity: 3},
		{RiftboundID: "ven-002-166", Name: "Blade Twirler", Number: "2/166", SetID: "VEN", Quantity: 2},
		{RiftboundID: "ven-003-166", Name: "Brittle Steel", Number: "3/166", SetID: "VEN", Quantity: 1},
	}}
	return newValidator(coll, filepath.Join(t.TempDir(), "collection.json"), "")
}

func quantity(t *testing.T, v *validator, id string) int {
	t.Helper()
	for _, e := range v.collection.Cards {
		if e.RiftboundID == id {
			return e.Quantity
		}
	}
	return 0
}

func TestValidatorStartsOnTheFirstCardWithItsCount(t *testing.T) {
	v := newTestValidator(t)
	if v.entry().Name != "Baccai Sandspinner" {
		t.Errorf("started on %q, want the first card on file", v.entry().Name)
	}
	if v.qty != "3" {
		t.Errorf("qty = %q, want the recorded 3", v.qty)
	}
}

func TestValidatorEnterAcceptsAndAdvances(t *testing.T) {
	v := newTestValidator(t)
	v.typing(t, "<enter>")
	if v.entry().Name != "Blade Twirler" {
		t.Errorf("enter left us on %q, want the next card", v.entry().Name)
	}
	if v.qty != "2" {
		t.Errorf("qty = %q, want the next card's count", v.qty)
	}
	if v.changed() != 0 {
		t.Errorf("accepting a card counted as %d changes, want none", v.changed())
	}
}

func TestValidatorCorrectsACount(t *testing.T) {
	v := newTestValidator(t)
	v.typing(t, "5<enter>")
	if got := quantity(t, v, "ven-001-166"); got != 5 {
		t.Errorf("quantity = %d, want the typed 5", got)
	}
	if v.changed() != 1 {
		t.Errorf("changed = %d, want 1", v.changed())
	}
}

func TestValidatorZeroRemovesThenTypingRestores(t *testing.T) {
	v := newTestValidator(t)

	v.typing(t, "0")
	if got := quantity(t, v, "ven-001-166"); got != 0 {
		t.Fatalf("quantity = %d, want the entry dropped", got)
	}

	v.typing(t, "2")
	if got := quantity(t, v, "ven-001-166"); got != 2 {
		t.Errorf("quantity = %d, want the entry back at 2", got)
	}
	if e := v.entry(); e.Name != "Baccai Sandspinner" || e.Number != "1/166" {
		t.Errorf("restored entry is %+v, want the card it was filed as", e)
	}
}

func TestValidatorBackReturnsToThePreviousCard(t *testing.T) {
	v := newTestValidator(t)
	v.typing(t, "<enter><enter><ctrl+z>")
	if v.entry().Name != "Blade Twirler" {
		t.Errorf("ctrl+z left us on %q, want the previous card", v.entry().Name)
	}

	v.typing(t, "<ctrl+z><ctrl+z>")
	if v.at != 0 {
		t.Errorf("at = %d, want to be held at the first card", v.at)
	}
	if !strings.Contains(v.status, "first card") {
		t.Errorf("status = %q, want it to say we're at the start", v.status)
	}
}

func TestValidatorPassEndsAfterTheLastCard(t *testing.T) {
	v := newTestValidator(t)

	for i := range 2 {
		if stop, _ := v.handle(key{name: "enter"}); stop {
			t.Fatalf("the pass stopped on card %d of 3", i+1)
		}
	}
	if stop, _ := v.handle(key{name: "enter"}); !stop {
		t.Error("the pass ran past its last card")
	}
	if v.at != len(v.ids) {
		t.Errorf("at = %d, want %d so the summary counts every card", v.at, len(v.ids))
	}
}

func TestValidatorBumpKeys(t *testing.T) {
	v := newTestValidator(t)
	v.typing(t, "+")
	if got := quantity(t, v, "ven-001-166"); got != 4 {
		t.Errorf("after +, quantity = %d, want 4", got)
	}
	v.typing(t, "<down>")
	if got := quantity(t, v, "ven-001-166"); got != 3 {
		t.Errorf("after down, quantity = %d, want 3", got)
	}
}

func mixedSetValidator(t *testing.T, setID string) *validator {
	t.Helper()
	coll := &collection{Cards: []collectedCard{
		{RiftboundID: "ogn-045-298", Name: "Defy", Number: "45/298", SetID: "OGN", Quantity: 1},
		{RiftboundID: "ven-001-166", Name: "Baccai Sandspinner", Number: "1/166", SetID: "VEN", Quantity: 3},
		{RiftboundID: "ven-002-166", Name: "Blade Twirler", Number: "2/166", SetID: "VEN", Quantity: 2},
	}}
	return newValidator(coll, filepath.Join(t.TempDir(), "collection.json"), setID)
}

func TestValidatorWalksOneSet(t *testing.T) {
	v := mixedSetValidator(t, "ven")
	if len(v.ids) != 2 {
		t.Fatalf("walking %d cards, want the 2 filed under VEN", len(v.ids))
	}
	if v.entry().Name != "Baccai Sandspinner" {
		t.Errorf("started on %q, want the first VEN card", v.entry().Name)
	}

	// The pass ends after the set's own cards, without falling into OGN.
	v.typing(t, "<enter>")
	if stop, _ := v.handle(key{name: "enter"}); !stop {
		t.Error("the pass ran past the end of the set")
	}
}

// Correcting inside one set must leave the rest of the collection alone.
func TestValidatorLeavesOtherSetsAlone(t *testing.T) {
	v := mixedSetValidator(t, "VEN")
	v.typing(t, "0<enter>9")
	if got := quantity(t, v, "ogn-045-298"); got != 1 {
		t.Errorf("the OGN card is now %d, want it untouched at 1", got)
	}
	if got := quantity(t, v, "ven-001-166"); got != 0 {
		t.Errorf("the zeroed VEN card is %d, want it dropped", got)
	}
	if got := quantity(t, v, "ven-002-166"); got != 9 {
		t.Errorf("the corrected VEN card is %d, want 9", got)
	}
}

func TestValidatorRejectsASetWithNothingFiledUnderIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.json")
	coll := &collection{Cards: []collectedCard{
		{RiftboundID: "ven-001-166", Name: "Baccai Sandspinner", Number: "1/166", SetID: "VEN", Quantity: 3},
	}}
	if err := coll.save(path); err != nil {
		t.Fatal(err)
	}

	err := RunValidator(path, "OGN")
	if err == nil {
		t.Fatal("want an error for a set with nothing filed under it")
	}
	for _, want := range []string{`"OGN"`, "VEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q doesn't mention %s", err, want)
		}
	}
}
