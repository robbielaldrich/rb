package collection

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"rb/cards"
)

const maxResults = 5

type mode int

const (
	modeSearch   mode = iota // typing a query, picking from the results
	modeQuantity             // how many copies
	modeConflict             // card already owned: add copies or replace
)

type model struct {
	index *cardIndex
	coll  *collection
	path  string

	mode    mode
	query   string
	results []result
	focused bool // result list has focus, so digits select instead of typing
	cursor  int
	picked  result
	qty     string
	status  string
	err     error
	quit    bool
}

func newModel(catalog []cards.Card, coll *collection, path string) *model {
	return &model{
		index: buildIndex(catalog),
		coll:  coll,
		path:  path,
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if key.String() == "ctrl+c" {
		m.quit = true
		return m, tea.Quit
	}

	switch m.mode {
	case modeSearch:
		return m, m.updateSearch(key)
	case modeQuantity:
		m.updateQuantity(key)
	case modeConflict:
		m.updateConflict(key)
	}
	return m, nil
}

// updateSearch drives the search prompt. Digits always belong to the query so
// "astral 44" reads naturally; tab hands the digits to the result list
// instead, where they act as select hotkeys.
func (m *model) updateSearch(key tea.KeyPressMsg) tea.Cmd {
	if m.focused {
		return m.updateResultList(key)
	}

	switch key.String() {
	case "esc":
		if m.query == "" {
			m.quit = true
			return tea.Quit
		}
		m.query, m.results, m.status = "", nil, ""
		return nil
	case "tab", "down":
		if len(m.results) > 0 {
			m.focused, m.cursor = true, 0
		}
		return nil
	case "enter":
		if len(m.results) > 0 {
			m.pick(m.results[0])
		}
		return nil
	case "backspace":
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	default:
		if key.Text == "" {
			return nil
		}
		m.query += key.Text
	}

	m.refresh()
	return nil
}

// updateResultList handles keys while the result list has focus.
func (m *model) updateResultList(key tea.KeyPressMsg) tea.Cmd {
	switch k := key.String(); k {
	case "esc", "tab":
		m.focused = false
	case "down":
		m.cursor = min(m.cursor+1, len(m.results)-1)
	case "up":
		if m.cursor == 0 {
			m.focused = false // walking off the top returns to the prompt
			return nil
		}
		m.cursor--
	case "enter":
		m.pick(m.results[m.cursor])
	default:
		if n, err := strconv.Atoi(k); err == nil && n >= 1 && n <= len(m.results) {
			m.pick(m.results[n-1])
			return nil
		}
		// Anything else is the user typing again: resume editing the query.
		m.focused = false
		if key.String() == "backspace" {
			if r := []rune(m.query); len(r) > 0 {
				m.query = string(r[:len(r)-1])
			}
		} else if key.Text != "" {
			m.query += key.Text
		}
		m.refresh()
	}
	return nil
}

// refresh re-runs the search for the current query.
func (m *model) refresh() {
	m.results = m.index.search(m.query, m.coll, maxResults)
	m.cursor, m.focused = 0, false
	m.status = ""
}

func (m *model) pick(r result) {
	m.picked = r
	m.qty = ""
	m.mode = modeQuantity
	m.focused = false
	m.status = ""
}

func (m *model) updateQuantity(key tea.KeyPressMsg) {
	switch key.String() {
	case "esc":
		m.mode = modeSearch
	case "backspace":
		if r := []rune(m.qty); len(r) > 0 {
			m.qty = string(r[:len(r)-1])
		}
	case "enter":
		qty := 1 // a bare enter means one copy
		if m.qty != "" {
			n, err := strconv.Atoi(m.qty)
			if err != nil || n < 0 {
				m.status = "quantity must be a whole number"
				return
			}
			qty = n
		}
		m.qty = strconv.Itoa(qty)
		if m.picked.owned > 0 {
			m.mode = modeConflict // ask before touching an existing count
			return
		}
		m.commit(qty)
	default:
		for _, r := range key.Text {
			if unicode.IsDigit(r) {
				m.qty += string(r)
			}
		}
	}
}

func (m *model) updateConflict(key tea.KeyPressMsg) {
	qty, err := strconv.Atoi(m.qty)
	if err != nil {
		m.mode = modeSearch
		return
	}

	switch key.String() {
	case "esc":
		m.mode = modeSearch
		m.status = "cancelled"
	case "a":
		m.commit(m.picked.owned + qty)
	case "r":
		m.commit(qty)
	}
}

// commit records qty copies of the picked card and writes the file.
func (m *model) commit(qty int) {
	m.coll.set(m.picked.card, qty, time.Now())
	if err := m.coll.save(m.path); err != nil {
		m.err = fmt.Errorf("failed to save collection: %w", err)
		return
	}

	label := cardLabel(m.picked.card)
	switch {
	case qty == 0:
		m.status = fmt.Sprintf("removed %s", label)
	case m.picked.owned > 0:
		m.status = fmt.Sprintf("%s: %d → %d", label, m.picked.owned, qty)
	default:
		m.status = fmt.Sprintf("added %s ×%d", label, qty)
	}

	m.mode = modeSearch
	m.query, m.results = "", nil
	m.cursor, m.focused = 0, false
}

const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiDim   = "\x1b[2m"
	ansiCyan  = "\x1b[36m"
)

func dim(s string) string  { return ansiDim + s + ansiReset }
func bold(s string) string { return ansiBold + s + ansiReset }

func (m *model) View() tea.View {
	if m.quit {
		return tea.NewView("")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n\n",
		bold("rb collection"),
		dim(fmt.Sprintf("%d cards · %d unique / %d total in %s",
			len(m.index.cards), len(m.coll.Cards), m.coll.total(), m.path)))

	switch m.mode {
	case modeSearch:
		m.viewSearch(&b)
	case modeQuantity:
		m.viewQuantity(&b)
	case modeConflict:
		m.viewConflict(&b)
	}

	if m.err != nil {
		fmt.Fprintf(&b, "\n%serror: %v%s\n", ansiCyan, m.err, ansiReset)
	} else if m.status != "" {
		fmt.Fprintf(&b, "\n%s\n", m.status)
	}
	return tea.NewView(b.String())
}

func (m *model) viewSearch(b *strings.Builder) {
	caret := "█"
	if m.focused {
		caret = "" // the cursor lives in the list while it has focus
	}
	fmt.Fprintf(b, "%s %s\n\n", ansiCyan+">"+ansiReset, m.query+caret)

	switch {
	case m.query == "":
		fmt.Fprint(b, dim("  type a name and/or number, e.g. \"astral 44\"\n"))
	case len(m.results) == 0:
		fmt.Fprint(b, dim("  no matches\n"))
	}

	for i, r := range m.results {
		owned := ""
		if r.owned > 0 {
			owned = dim(fmt.Sprintf("  (have %d)", r.owned))
		}
		marker := " "
		if m.focused && i == m.cursor {
			marker = ansiCyan + "▸" + ansiReset
		}
		fmt.Fprintf(b, "%s %s  %-28s %s %s%s\n",
			marker,
			bold(strconv.Itoa(i+1)),
			r.card.Name,
			cardNumber(r.card),
			strings.ToUpper(r.card.Set.SetID),
			owned)
	}

	if m.focused {
		fmt.Fprint(b, "\n"+dim("  1-5 select · ↑↓ move · enter confirms · esc back to typing\n"))
	} else {
		fmt.Fprint(b, "\n"+dim("  tab focuses list · enter takes the top hit · esc clears · ctrl+c quits\n"))
	}
}

func (m *model) viewQuantity(b *strings.Builder) {
	fmt.Fprintf(b, "  %s\n\n", bold(cardLabel(m.picked.card)))
	if m.picked.owned > 0 {
		fmt.Fprint(b, dim(fmt.Sprintf("  you already have %d\n\n", m.picked.owned)))
	}
	fmt.Fprintf(b, "  quantity: %s  %s\n", m.qty+"█", dim("(enter = 1)"))
	fmt.Fprint(b, "\n"+dim("  enter confirms · esc goes back\n"))
}

func (m *model) viewConflict(b *strings.Builder) {
	qty, _ := strconv.Atoi(m.qty)
	fmt.Fprintf(b, "  %s\n\n", bold(cardLabel(m.picked.card)))
	fmt.Fprintf(b, "  already in your collection at %s.\n\n", bold(strconv.Itoa(m.picked.owned)))
	fmt.Fprintf(b, "  %s add copies   %s\n", bold("a"), dim(fmt.Sprintf("%d + %d = %d", m.picked.owned, qty, m.picked.owned+qty)))
	fmt.Fprintf(b, "  %s replace      %s\n", bold("r"), dim(fmt.Sprintf("%d → %d", m.picked.owned, qty)))
	fmt.Fprint(b, "\n"+dim("  esc cancels\n"))
}
