package collection

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"rb/catalog"
)

// The search index is a trie over the words of every card name, walked with a
// Levenshtein DP row so a query tolerates typos.

type trieNode struct {
	children map[rune]*trieNode
	cards    []int // cards whose name contains the word ending at this node
}

type cardIndex struct {
	root   *trieNode
	cards  []catalog.Card
	setIDs map[string]bool
}

func buildIndex(cs []catalog.Card) *cardIndex {
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
	card     catalog.Card
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
