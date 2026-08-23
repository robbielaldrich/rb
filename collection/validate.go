package collection

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// RunValidator walks the collection one card at a time so the counts on file
// can be checked against the cards in hand. It reads no catalog: every entry
// already carries the name, number and set it was recorded under.
//
// An empty setID walks the whole collection; naming a set walks only what is
// filed under it, for checking one binder at a time.
func RunValidator(collectionPath, setID string) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}
	if len(coll.Cards) == 0 {
		return fmt.Errorf("%s holds no cards yet, run `rb collect` first", collectionPath)
	}

	v := newValidator(coll, collectionPath, setID)
	if len(v.ids) == 0 {
		filed := map[string]bool{}
		for _, e := range coll.Cards {
			filed[e.SetID] = true
		}
		return fmt.Errorf("no cards from set %q in the collection, sets on file: %s",
			setID, strings.Join(slices.Sorted(maps.Keys(filed)), " "))
	}
	if err := v.run(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("failed to run the validation pass: %w", err)
	}

	fmt.Printf("checked %d of %d cards, changed %d\n", v.at, len(v.ids), v.changed())
	return nil
}

type validator struct {
	collection *collection
	path       string

	// setID is the set being checked, upper case, or empty for the lot.
	setID string
	// ids are the cards to walk, in the order they are filed, and start is
	// what each was worth when the pass began. Both are fixed up front so
	// that setting a count to zero can't shift the ground underfoot.
	ids   []string
	start map[string]collectedCard

	at     int
	qty    string
	fresh  bool
	status string
}

func newValidator(coll *collection, path, setID string) *validator {
	v := &validator{collection: coll, path: path, setID: strings.ToUpper(setID), start: map[string]collectedCard{}}
	for _, e := range coll.Cards {
		if v.setID != "" && e.SetID != v.setID {
			continue
		}
		v.ids = append(v.ids, e.RiftboundID)
		v.start[e.RiftboundID] = e
	}
	if len(v.ids) > 0 {
		v.load()
	}
	return v
}

func (v *validator) run(in, out *os.File) error {
	return runLoop(in, out, v.frame, v.handle)
}

// load reads the count of the card now under the cursor into the edit box.
func (v *validator) load() {
	v.qty, v.fresh = strconv.Itoa(v.owned()), true
}

func (v *validator) entry() collectedCard { return v.start[v.ids[v.at]] }

// owned is what the card is worth right now, which is not what it was worth
// when the pass began if it has already been corrected.
func (v *validator) owned() int {
	if i := v.index(); i >= 0 {
		return v.collection.Cards[i].Quantity
	}
	return 0
}

func (v *validator) index() int {
	id := v.ids[v.at]
	return slices.IndexFunc(v.collection.Cards, func(e collectedCard) bool {
		return e.RiftboundID == id
	})
}

// changed counts the cards whose quantity no longer matches what it was when
// the pass began.
func (v *validator) changed() int {
	n := 0
	for _, e := range v.start {
		i := slices.IndexFunc(v.collection.Cards, func(c collectedCard) bool {
			return c.RiftboundID == e.RiftboundID
		})
		switch {
		case i < 0 && e.Quantity != 0, i >= 0 && v.collection.Cards[i].Quantity != e.Quantity:
			n++
		}
	}
	return n
}

func (v *validator) handle(k key) (stop bool, err error) {
	switch k.name {
	case "ctrl+c", "ctrl+d", "esc":
		return true, nil
	case "enter":
		return v.next(), nil
	case "ctrl+z":
		v.back()
		return false, nil
	case "backspace":
		if v.fresh {
			v.qty = ""
		} else {
			v.qty = dropLastRune(v.qty)
		}
		v.fresh = false
		return false, v.apply()
	case "up":
		return false, v.bump(1)
	case "down":
		return false, v.bump(-1)
	case "":
		switch r := k.r; {
		case r >= '0' && r <= '9':
			if v.fresh {
				v.qty, v.fresh = "", false
			}
			if len(v.qty) < 4 {
				v.qty += string(r)
			}
			return false, v.apply()
		case r == '+' || r == '=':
			return false, v.bump(1)
		case r == '-' || r == '_':
			return false, v.bump(-1)
		case r == 'y' || r == 'Y':
			return v.next(), nil
		case unicode.IsSpace(r):
			return v.next(), nil
		}
	}
	return false, nil
}

// next moves to the card after this one, reporting whether the pass is over.
func (v *validator) next() bool {
	if v.at+1 >= len(v.ids) {
		v.at = len(v.ids)
		return true
	}
	v.at++
	v.status = ""
	v.load()
	return false
}

func (v *validator) back() {
	if v.at == 0 {
		v.status = " · already at the first card"
		return
	}
	v.at--
	v.status = ""
	v.load()
}

func (v *validator) bump(d int) error {
	n, _ := strconv.Atoi(v.qty)
	v.qty, v.fresh = strconv.Itoa(max(n+d, 0)), false
	return v.apply()
}

// apply writes the count under the cursor through to disk on every keystroke,
// so an interrupted pass keeps whatever was already corrected.
func (v *validator) apply() error {
	n, _ := strconv.Atoi(v.qty)
	i := v.index()
	switch {
	case n <= 0 && i >= 0:
		v.collection.Cards = slices.Delete(v.collection.Cards, i, i+1)
	case n > 0 && i >= 0:
		v.collection.Cards[i].Quantity = n
		v.collection.Cards[i].UpdatedAt = time.Now()
	case n > 0:
		e := v.entry()
		e.Quantity, e.UpdatedAt = n, time.Now()
		v.collection.Cards = append(v.collection.Cards, e)
	}

	if err := v.collection.save(v.path); err != nil {
		return fmt.Errorf("failed to save collection: %w", err)
	}
	return nil
}
