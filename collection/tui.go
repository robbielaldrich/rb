package collection

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"rb/cards"
)

// maxMatches is both how many matches are listed and how many digit keys
// select one, so it can't go past 9.
const maxMatches = 5

type mode int

const (
	// modeSearch types a card name; digits pick one of the listed matches.
	modeSearch mode = iota
	// modeQuantity edits how many copies of the card just picked you own.
	modeQuantity
)

type editor struct {
	index      *cardIndex
	collection *collection
	path       string
	// scope names the set and domain the search is confined to, upper case,
	// or is empty when the whole catalog is in play.
	scope string

	mode    mode
	query   string
	results []result
	status  string

	card cards.Card
	// undo holds what each edited card was worth before it was picked, most
	// recent last, so an edit can be taken back after it has been made.
	undo []change
	// qty is held as text so backspacing to an empty box is representable.
	qty string
	// qtyFresh marks a quantity the editor chose rather than the user, so the
	// first digit typed replaces it instead of appending to it.
	qtyFresh bool
}

// change is the quantity a card carried before an edit.
type change struct {
	card cards.Card
	qty  int
}

func newEditor(coll *collection, cs []cards.Card, path, scope string) *editor {
	return &editor{index: buildIndex(cs), collection: coll, path: path, scope: strings.ToUpper(scope)}
}

func (e *editor) run(in, out *os.File) error {
	return runLoop(in, out, e.frame, e.handle)
}

func (e *editor) handle(k key) (quit bool, err error) {
	switch k.name {
	case "ctrl+c", "ctrl+d":
		return true, nil
	case "ctrl+z":
		return false, e.undoLast()
	case "esc":
		if e.mode == modeQuantity {
			e.done()
			return false, nil
		}
		return true, nil
	}
	if e.mode == modeQuantity {
		return false, e.handleQuantity(k)
	}
	return false, e.handleSearch(k)
}

func (e *editor) handleSearch(k key) error {
	switch k.name {
	case "backspace":
		e.query = dropLastRune(e.query)
	case "ctrl+u":
		e.query = ""
	case "ctrl+w":
		e.query = dropLastWord(e.query)
	case "enter":
		if len(e.results) > 0 {
			return e.pick(0)
		}
	case "":
		// Digits select a match, so a collector number has to be entered
		// behind a '#'. Once such a token is open, digits go into it.
		if r := k.r; r >= '1' && r <= '9' && !inNumberEntry(e.query) {
			if i := int(r - '1'); i < len(e.results) {
				return e.pick(i)
			}
			return nil
		}
		e.query += string(k.r)
	}
	e.refresh()
	return nil
}

func (e *editor) handleQuantity(k key) error {
	switch k.name {
	case "backspace":
		if e.qtyFresh {
			e.qty = ""
		} else {
			e.qty = dropLastRune(e.qty)
		}
		e.qtyFresh = false
		return e.apply()
	case "enter":
		e.done()
		return nil
	case "up":
		return e.bump(1)
	case "down":
		return e.bump(-1)
	case "":
		switch r := k.r; {
		case r >= '0' && r <= '9':
			if e.qtyFresh {
				e.qty, e.qtyFresh = "", false
			}
			if len(e.qty) < 4 { // nobody owns ten thousand of one card
				e.qty += string(r)
			}
			return e.apply()
		case r == '+' || r == '=':
			return e.bump(1)
		case r == '-' || r == '_':
			return e.bump(-1)
		case unicode.IsSpace(r):
			return nil
		default:
			// Anything else is the user moving on to the next card.
			e.done()
			e.query = string(r)
			e.refresh()
			return nil
		}
	}
	return nil
}

// pick records one more copy of the i'th match straight away, then hands the
// quantity to the user to adjust.
func (e *editor) pick(i int) error {
	r := e.results[i]
	e.undo = append(e.undo, change{card: r.card, qty: r.owned})
	e.card, e.mode = r.card, modeQuantity
	e.qty, e.qtyFresh = strconv.Itoa(r.owned+1), true
	e.status = ""
	return e.apply()
}

// undoLast puts the most recently picked card back to the quantity it had
// before it was picked, whether or not that edit was finished, and drops the
// user back at the search prompt.
func (e *editor) undoLast() error {
	if len(e.undo) == 0 {
		e.status = "nothing to undo"
		return nil
	}
	c := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]

	e.collection.set(c.card, c.qty, time.Now())
	if err := e.collection.save(e.path); err != nil {
		return fmt.Errorf("failed to save collection: %w", err)
	}

	if c.qty == 0 {
		e.status = fmt.Sprintf("undid %s — none owned", c.card.Label())
	} else {
		e.status = fmt.Sprintf("undid %s — back to %d", c.card.Label(), c.qty)
	}
	e.mode, e.query = modeSearch, ""
	e.refresh()
	return nil
}

// apply writes the quantity being edited through to disk on every keystroke,
// so an interrupted session never loses what you already typed.
func (e *editor) apply() error {
	n, err := strconv.Atoi(e.qty)
	if err != nil { // an empty box means none owned, for now
		n = 0
	}
	e.collection.set(e.card, n, time.Now())
	if err := e.collection.save(e.path); err != nil {
		return fmt.Errorf("failed to save collection: %w", err)
	}
	return nil
}

func (e *editor) bump(d int) error {
	n, _ := strconv.Atoi(e.qty)
	e.qty, e.qtyFresh = strconv.Itoa(max(n+d, 0)), false
	return e.apply()
}

// done leaves quantity editing, reporting the count that was stored so it
// stays on screen while the next card is searched for.
func (e *editor) done() {
	n, _ := strconv.Atoi(e.qty)
	switch {
	case n == 0:
		e.status = fmt.Sprintf("removed %s", e.card.Label())
	default:
		e.status = fmt.Sprintf("%s — you now have %d", e.card.Label(), n)
	}
	e.mode, e.query = modeSearch, ""
	e.refresh()
}

func (e *editor) refresh() {
	// '#' is only there to keep a collector number out of the selection keys;
	// the search itself reads a bare number.
	e.results = e.index.search(strings.ReplaceAll(e.query, "#", ""), e.collection, maxMatches)
}

// inNumberEntry reports whether the query ends in an open '#' token, i.e.
// whether the next digit typed belongs to a collector number.
func inNumberEntry(query string) bool {
	i := strings.LastIndexFunc(query, unicode.IsSpace)
	return strings.HasPrefix(query[i+1:], "#")
}

func dropLastRune(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i]&0xc0 != 0x80 { // start of the last rune
			return s[:i]
		}
	}
	return ""
}

func dropLastWord(s string) string {
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	if i := strings.LastIndexFunc(s, unicode.IsSpace); i >= 0 {
		return s[:i+1]
	}
	return ""
}
