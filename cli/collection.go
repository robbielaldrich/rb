package cli

import (
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"rb/cards"
)

// RunCollection opens the interactive collection tracker: a search prompt
// that fuzzy-matches cards by name and collector number, and records how many
// copies of each you own in a JSON file.
func RunCollection(args []string) error {
	fs := flag.NewFlagSet("collection", flag.ContinueOnError)
	cardsDir := fs.String("cards", "cards", "directory holding cards.json")
	file := fs.String("file", "collection.json", "collection file to read and write")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	catalog, err := loadCatalog(filepath.Join(*cardsDir, "cards.json"))
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	coll, err := loadCollection(*file)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	m := newCollectionModel(catalog, coll, *file)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return fmt.Errorf("failed to run collection UI: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// catalog
// ---------------------------------------------------------------------------

// loadCatalog reads cards.json and deduplicates it by riftbound_id. The API
// returns more than one record for some cards (a stale one and a refreshed
// one); the most recently updated record wins.
func loadCatalog(path string) ([]cards.Card, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s not found, run `rb download-cards` first: %w", path, err)
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var all []cards.Card
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	newest := make(map[string]cards.Card, len(all))
	for _, c := range all {
		if prev, ok := newest[c.RiftboundID]; ok && prev.Metadata.UpdatedOn >= c.Metadata.UpdatedOn {
			continue
		}
		newest[c.RiftboundID] = c
	}

	out := make([]cards.Card, 0, len(newest))
	for _, c := range newest {
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b cards.Card) int {
		if n := cmp.Compare(a.Set.SetID, b.Set.SetID); n != 0 {
			return n
		}
		return cmp.Compare(a.CollectorNumber, b.CollectorNumber)
	})
	return out, nil
}

// riftIDRe matches the common riftbound_id shape, e.g. ven-044-166 or
// ven-044a-166, whose parts are the set, the collector number (with an
// optional variant letter), and the printed set size.
var riftIDRe = regexp.MustCompile(`^([a-z]+)-([0-9]+[a-z*]?)-([0-9]+)$`)

// cardNumber renders a card's printed number, e.g. "44/166". Cards whose
// riftbound_id doesn't carry a set size (runes, some promos) fall back to
// their collector number alone.
func cardNumber(c cards.Card) string {
	if m := riftIDRe.FindStringSubmatch(c.RiftboundID); m != nil {
		num := strings.TrimLeft(m[2], "0")
		if num == "" {
			num = "0"
		}
		return num + "/" + m[3]
	}
	return strconv.Itoa(c.CollectorNumber)
}

// cardLabel renders a card the way it is written on paper, e.g.
// "Astral Heron 44/166 VEN".
func cardLabel(c cards.Card) string {
	return fmt.Sprintf("%s %s %s", c.Name, cardNumber(c), strings.ToUpper(c.Set.SetID))
}

// ---------------------------------------------------------------------------
// collection file
// ---------------------------------------------------------------------------

// collectionEntry is one owned card. Cards are keyed by riftbound_id; the
// name, number and set are denormalised so the file stays readable on its own.
type collectionEntry struct {
	RiftboundID string    `json:"riftbound_id"`
	Name        string    `json:"name"`
	Number      string    `json:"number"`
	SetID       string    `json:"set_id"`
	Quantity    int       `json:"quantity"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type collection struct {
	Cards []collectionEntry `json:"cards"`
}

func loadCollection(path string) (*collection, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &collection{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var c collection
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &c, nil
}

func (c *collection) find(riftboundID string) (collectionEntry, bool) {
	for _, e := range c.Cards {
		if e.RiftboundID == riftboundID {
			return e, true
		}
	}
	return collectionEntry{}, false
}

// set records qty copies of a card, dropping the entry entirely if qty is
// zero or less.
func (c *collection) set(card cards.Card, qty int, now time.Time) {
	for i, e := range c.Cards {
		if e.RiftboundID != card.RiftboundID {
			continue
		}
		if qty <= 0 {
			c.Cards = slices.Delete(c.Cards, i, i+1)
			return
		}
		c.Cards[i].Quantity = qty
		c.Cards[i].UpdatedAt = now
		return
	}
	if qty <= 0 {
		return
	}
	c.Cards = append(c.Cards, collectionEntry{
		RiftboundID: card.RiftboundID,
		Name:        card.Name,
		Number:      cardNumber(card),
		SetID:       strings.ToUpper(card.Set.SetID),
		Quantity:    qty,
		UpdatedAt:   now,
	})
}

func (c *collection) total() int {
	var n int
	for _, e := range c.Cards {
		n += e.Quantity
	}
	return n
}

// save writes the collection atomically so an interrupted write can't leave a
// truncated file behind.
func (c *collection) save(path string) error {
	sorted := slices.Clone(c.Cards)
	slices.SortFunc(sorted, func(a, b collectionEntry) int {
		if n := cmp.Compare(a.SetID, b.SetID); n != 0 {
			return n
		}
		return cmp.Compare(a.RiftboundID, b.RiftboundID)
	})

	data, err := json.MarshalIndent(collection{Cards: sorted}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal collection: %w", err)
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", tmp, path, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// search index: a trie over name words, walked with a Levenshtein DP row
// ---------------------------------------------------------------------------

type trieNode struct {
	children map[rune]*trieNode
	cards    []int // cards whose name contains the word ending at this node
}

type cardIndex struct {
	root   *trieNode
	cards  []cards.Card
	setIDs map[string]bool
}

func buildIndex(cs []cards.Card) *cardIndex {
	ix := &cardIndex{root: &trieNode{}, cards: cs, setIDs: map[string]bool{}}
	for i, c := range cs {
		ix.setIDs[strings.ToLower(c.Set.SetID)] = true
		for _, w := range nameWords(c.Name) {
			ix.insert(w, i)
		}
	}
	return ix
}

// nameWords splits a card name into lowercase alphanumeric words, e.g.
// "Annie - Fiery" becomes ["annie", "fiery"].
func nameWords(name string) []string {
	fields := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return slices.Compact(slices.Sorted(slices.Values(fields)))
}

func (ix *cardIndex) insert(word string, card int) {
	n := ix.root
	for _, r := range word {
		if n.children == nil {
			n.children = map[rune]*trieNode{}
		}
		child, ok := n.children[r]
		if !ok {
			child = &trieNode{}
			n.children[r] = child
		}
		n = child
	}
	if len(n.cards) == 0 || n.cards[len(n.cards)-1] != card {
		n.cards = append(n.cards, card)
	}
}

// searchToken walks the trie carrying one Levenshtein DP row per node, and
// returns the best edit distance found for every card whose name contains a
// close-enough word. A node whose row shows the whole token consumed is a
// prefix hit, so its entire subtree matches at that distance: typing "astr"
// finds "Astral" for free, while "astrel" costs one edit.
func (ix *cardIndex) searchToken(token string, maxDist int) map[int]int {
	out := map[int]int{}
	q := []rune(token)

	row := make([]int, len(q)+1)
	for i := range row {
		row[i] = i
	}
	for r, child := range ix.root.children {
		ix.walk(child, r, q, row, maxDist, out)
	}
	return out
}

func (ix *cardIndex) walk(n *trieNode, r rune, q []rune, prev []int, maxDist int, out map[int]int) {
	cur := make([]int, len(prev))
	cur[0] = prev[0] + 1
	best := cur[0]
	for i := 1; i <= len(q); i++ {
		cost := 1
		if q[i-1] == r {
			cost = 0
		}
		cur[i] = min(cur[i-1]+1, prev[i]+1, prev[i-1]+cost)
		best = min(best, cur[i])
	}

	if d := cur[len(q)]; d <= maxDist {
		collectSubtree(n, d, out)
	}
	if best > maxDist { // no descendant can improve on this row
		return
	}
	for cr, child := range n.children {
		ix.walk(child, cr, q, cur, maxDist, out)
	}
}

func collectSubtree(n *trieNode, d int, out map[int]int) {
	for _, c := range n.cards {
		if prev, ok := out[c]; !ok || d < prev {
			out[c] = d
		}
	}
	for _, child := range n.children {
		collectSubtree(child, d, out)
	}
}

// maxEdits scales the edit budget to the token length, so short tokens stay
// precise instead of matching most of the catalog.
func maxEdits(token string) int {
	switch n := len([]rune(token)); {
	case n <= 3:
		return 0
	case n <= 6:
		return 1
	default:
		return 2
	}
}

// query is a parsed search line: name words, an optional collector number,
// and an optional set code.
type query struct {
	words  []string
	number int
	hasNum bool
	setID  string
}

func (ix *cardIndex) parseQuery(line string) query {
	var q query
	for _, tok := range strings.Fields(strings.ToLower(line)) {
		tok = strings.Trim(tok, ".,")
		// "44/166" and "44" both mean collector number 44.
		numPart, _, _ := strings.Cut(tok, "/")
		if n, err := strconv.Atoi(numPart); err == nil && numPart != "" {
			q.number, q.hasNum = n, true
			continue
		}
		if ix.setIDs[tok] {
			q.setID = tok
			continue
		}
		if tok != "" {
			q.words = append(q.words, tok)
		}
	}
	return q
}

type result struct {
	card     cards.Card
	dist     int
	numMatch bool
	setMatch bool
	owned    int
	recency  time.Time
}

// search ranks candidates by relevance first (an exact collector-number hit,
// then edit distance, then set), and breaks ties by recency so the cards you
// most recently touched float to the top of an otherwise equal field.
func (ix *cardIndex) search(line string, coll *collection, limit int) []result {
	q := ix.parseQuery(line)
	if len(q.words) == 0 && !q.hasNum && q.setID == "" {
		return nil
	}

	// Intersect per-word matches: every typed word has to match something.
	var candidates map[int]int
	for _, w := range q.words {
		hits := ix.searchToken(w, maxEdits(w))
		if candidates == nil {
			candidates = hits
			continue
		}
		merged := make(map[int]int, len(hits))
		for card, d := range hits {
			if prev, ok := candidates[card]; ok {
				merged[card] = prev + d
			}
		}
		candidates = merged
	}
	if candidates == nil { // number and/or set only
		candidates = map[int]int{}
		for i := range ix.cards {
			candidates[i] = 0
		}
	}

	recency := make(map[string]time.Time, len(coll.Cards))
	owned := make(map[string]int, len(coll.Cards))
	for _, e := range coll.Cards {
		recency[e.RiftboundID] = e.UpdatedAt
		owned[e.RiftboundID] = e.Quantity
	}

	out := make([]result, 0, len(candidates))
	for i, d := range candidates {
		c := ix.cards[i]
		r := result{
			card:     c,
			dist:     d,
			numMatch: q.hasNum && c.CollectorNumber == q.number,
			setMatch: q.setID != "" && strings.EqualFold(q.setID, c.Set.SetID),
			owned:    owned[c.RiftboundID],
			recency:  recency[c.RiftboundID],
		}
		// A bare number or set code is a filter, not a hint: without name
		// words there is nothing else to rank on.
		if len(q.words) == 0 {
			if q.hasNum && !r.numMatch {
				continue
			}
			if q.setID != "" && !r.setMatch {
				continue
			}
		}
		out = append(out, r)
	}

	slices.SortFunc(out, func(a, b result) int {
		if n := cmp.Compare(boolRank(b.numMatch), boolRank(a.numMatch)); n != 0 {
			return n
		}
		if n := cmp.Compare(a.dist, b.dist); n != 0 {
			return n
		}
		if n := cmp.Compare(boolRank(b.setMatch), boolRank(a.setMatch)); n != 0 {
			return n
		}
		if n := b.recency.Compare(a.recency); n != 0 {
			return n
		}
		if n := cmp.Compare(a.card.Name, b.card.Name); n != 0 {
			return n
		}
		return cmp.Compare(a.card.RiftboundID, b.card.RiftboundID)
	})

	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------
// TUI
// ---------------------------------------------------------------------------

const maxResults = 5

type mode int

const (
	modeSearch   mode = iota // typing a query, picking from the results
	modeQuantity             // how many copies
	modeConflict             // card already owned: add copies or replace
)

type collectionModel struct {
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

func newCollectionModel(catalog []cards.Card, coll *collection, path string) *collectionModel {
	return &collectionModel{
		index: buildIndex(catalog),
		coll:  coll,
		path:  path,
	}
}

func (m *collectionModel) Init() tea.Cmd { return nil }

func (m *collectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m *collectionModel) updateSearch(key tea.KeyPressMsg) tea.Cmd {
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
func (m *collectionModel) updateResultList(key tea.KeyPressMsg) tea.Cmd {
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
func (m *collectionModel) refresh() {
	m.results = m.index.search(m.query, m.coll, maxResults)
	m.cursor, m.focused = 0, false
	m.status = ""
}

func (m *collectionModel) pick(r result) {
	m.picked = r
	m.qty = ""
	m.mode = modeQuantity
	m.focused = false
	m.status = ""
}

func (m *collectionModel) updateQuantity(key tea.KeyPressMsg) {
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

func (m *collectionModel) updateConflict(key tea.KeyPressMsg) {
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
func (m *collectionModel) commit(qty int) {
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

func (m *collectionModel) View() tea.View {
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

func (m *collectionModel) viewSearch(b *strings.Builder) {
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

func (m *collectionModel) viewQuantity(b *strings.Builder) {
	fmt.Fprintf(b, "  %s\n\n", bold(cardLabel(m.picked.card)))
	if m.picked.owned > 0 {
		fmt.Fprint(b, dim(fmt.Sprintf("  you already have %d\n\n", m.picked.owned)))
	}
	fmt.Fprintf(b, "  quantity: %s  %s\n", m.qty+"█", dim("(enter = 1)"))
	fmt.Fprint(b, "\n"+dim("  enter confirms · esc goes back\n"))
}

func (m *collectionModel) viewConflict(b *strings.Builder) {
	qty, _ := strconv.Atoi(m.qty)
	fmt.Fprintf(b, "  %s\n\n", bold(cardLabel(m.picked.card)))
	fmt.Fprintf(b, "  already in your collection at %s.\n\n", bold(strconv.Itoa(m.picked.owned)))
	fmt.Fprintf(b, "  %s add copies   %s\n", bold("a"), dim(fmt.Sprintf("%d + %d = %d", m.picked.owned, qty, m.picked.owned+qty)))
	fmt.Fprintf(b, "  %s replace      %s\n", bold("r"), dim(fmt.Sprintf("%d → %d", m.picked.owned, qty)))
	fmt.Fprint(b, "\n"+dim("  esc cancels\n"))
}
