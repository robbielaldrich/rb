package riftboundgg

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"rb/cards"
)

// idRe reads a riftbound.gg card id, e.g. RAD-020A, VEN-R01b-P or
// UNL-058-P-Champion: the set, a series letter for the printings numbered
// outside the main run (SP special showcases, R runes, T tokens), the number,
// a variant letter, and whatever tag follows (-STAR for a signed printing,
// and for a promo -P with some mark of which promo it is).
var idRe = regexp.MustCompile(`^([A-Z0-9]+)-(SP|R|T)?(\d+)([A-Za-z]?)((?:-[A-Za-z0-9]+)*)$`)

// parsedID is a riftbound.gg card id taken apart by idRe.
type parsedID struct {
	set, series, number, letter, tag string
}

func parseID(id string) (parsedID, bool) {
	m := idRe.FindStringSubmatch(id)
	if m == nil {
		return parsedID{}, false
	}
	return parsedID{set: m[1], series: m[2], number: m[3], letter: strings.ToLower(m[4]), tag: m[5]}, true
}

// mainRun reports whether a card is the plain printing at its own number,
// rather than a variant, a signed copy or a card numbered outside the run.
func (p parsedID) mainRun() bool { return p.series == "" && p.letter == "" && p.tag == "" }

// updatedOn stamps each converted card, as the API keeps no date of its own;
// cards.Load only compares it between records sharing an id, which the
// conversion never produces.
const updatedOn = "1970-01-01T00:00:00+00:00"

// setInfo is what converting a set's cards needs to know about the set.
type setInfo struct {
	code, label string
	// size is the number the main run is printed out of, e.g. 166 for
	// Vendetta, which every id in the set is quoted against.
	size int
	// specials counts the set's SP printings, which are numbered out of
	// their own run, e.g. ven-sp3-006.
	specials int
	// plain is the main-run printing of each card, by name, whose rarity the
	// showcase reprints of it take. It spans every set, since a set's
	// overnumbered showcases are often of champions first printed in an
	// earlier one.
	plain map[string]row
}

// isPlain reports whether a row is the plain printing of a card in its set's
// main run.
func isPlain(r row) bool {
	p, ok := parseID(r.ID)
	return ok && p.mainRun() && r.Promo != "1" && r.Rarity != "Showcase"
}

// plainPrintings indexes the plain printings of every set by card name,
// taking the first met where a card is printed in more than one set; a
// reprint keeps its rarity.
func plainPrintings(rows []row) map[string]row {
	plain := map[string]row{}
	for _, r := range rows {
		if _, seen := plain[r.NameNormal]; !seen && isPlain(r) {
			plain[r.NameNormal] = r
		}
	}
	return plain
}

// newSetInfo works out a set's numbering from its cards.
//
// The printed set size is not in the API, but every set's main run ends with
// its battlefields, so it is the number of the last battlefield that isn't a
// showcase. A set with no battlefields falls back to its last main-run card.
func newSetInfo(code, label string, rows []row, plain map[string]row) setInfo {
	s := setInfo{code: code, label: label, plain: plain}
	lastAny := 0
	for _, r := range rows {
		p, ok := parseID(r.ID)
		if !ok {
			continue
		}
		if p.series == "SP" {
			s.specials++
		}
		if !isPlain(r) {
			continue
		}
		n, _ := strconv.Atoi(p.number)
		lastAny = max(lastAny, n)
		if len(r.Type) > 0 && r.Type[0] == cards.TypeBattlefield {
			s.size = max(s.size, n)
		}
	}
	if s.size == 0 {
		s.size = lastAny
	}
	return s
}

// convert turns one riftbound.gg row into the record package cards reads.
// It reports false for the cards the catalog leaves out: tokens, which are
// not cards a deck is built from.
//
// Ids follow the scheme Riftcodex, the source this replaced, gave the main
// sets — ven-044-166, ven-044a-166 for an alternate art, unl-226*-219 for a
// signed legend, ven-r01 for a rune, ven-sp3-006 for a special showcase — so
// that collections and decks recorded against it still line up. Promos are
// filed under the set they reprint, marked with a "p" and whatever tells one
// promo of the card from another: rad-015p-166, rad-015p2-166, ogn-007bp-298.
// The "p" keeps a promo's id from ending in the "a" that marks an alternate
// art.
func convert(r row, p parsedID, s setInfo) (cards.Card, bool) {
	if p.series == "T" {
		return cards.Card{}, false
	}

	n, _ := strconv.Atoi(p.number)
	set := strings.ToLower(s.code)
	series := strings.ToLower(p.series)
	name := cardName(r.NameNormal)
	plain, hasPlain := s.plain[r.NameNormal]
	rarity := r.Rarity
	var alt, over, signed bool
	var id string

	switch {
	case r.Promo == "1":
		tag := strings.ToLower(strings.ReplaceAll(p.tag, "-", ""))
		if !strings.Contains(tag, "p") {
			tag += "p"
		}
		id = fmt.Sprintf("%s-%s%s%s%s-%03d", set, series, p.number, p.letter, tag, s.size)
		name, rarity = cardName(r.Name), cards.RarityPromo
	case p.series == "R":
		id = fmt.Sprintf("%s-r%s%s", set, p.number, p.letter)
	case p.series == "SP":
		id = fmt.Sprintf("%s-sp%d-%03d", set, n, s.specials)
	default:
		star := ""
		if strings.EqualFold(p.tag, "-STAR") {
			star = "*"
		}
		id = fmt.Sprintf("%s-%s%s%s-%03d", set, p.number, p.letter, star, s.size)
	}

	if r.Promo != "1" {
		switch {
		case strings.EqualFold(p.tag, "-STAR"):
			signed, name = true, name+" (Signature)"
		case p.letter == "a" || p.series == "SP":
			alt, name = true, name+" (Alternate Art)"
		case p.series == "" && n > s.size && r.Rarity == "Showcase":
			over, name = true, name+" (Overnumbered)"
		}
		// A showcase reprint is pulled at the rarity of the card it reprints;
		// only the runes' alternate arts are a rarity of their own.
		if rarity == "Showcase" && hasPlain && !(len(r.Type) > 0 && r.Type[0] == "Rune") {
			rarity = plain.Rarity
		}
	}

	var typ string
	if len(r.Type) > 0 {
		typ = r.Type[0]
	}
	effect := string(r.Effect)

	return cards.Card{
		Name:            name,
		RiftboundID:     id,
		TCGPlayerID:     optional(string(r.MarketIDs)),
		CollectorNumber: n,
		Attributes: cards.Attributes{
			Energy: number(r.Cost),
			Might:  number(r.Might),
			Power:  positive(number(r.Power)),
		},
		Classification: cards.Classification{
			Type:      typ,
			Supertype: optional(string(r.Supertype)),
			Rarity:    rarity,
			Domain:    nonNil(r.Color),
		},
		Text: cards.Text{
			Rich:    rich(effect),
			Plain:   tagRe.ReplaceAllString(effect, ""),
			Flavour: optional(string(r.Flavor)),
		},
		Set:         cards.CardSet{SetID: s.code, Label: s.label},
		Media:       cards.Media{ImageURL: string(r.Image)},
		Tags:        nonNil(r.Tags),
		Orientation: orientation(typ),
		Metadata: cards.Metadata{
			UpdatedOn:    updatedOn,
			AlternateArt: alt,
			Overnumbered: over,
			Signature:    signed,
		},
		New: new(bool),
	}, true
}

// cardName drops the " - Starter" riftbound.gg appends to the legends of the
// Proving Grounds starter decks, which is no part of the printed name and
// would keep them from matching the decklists that name them.
func cardName(name string) string { return strings.Replace(name, " - Starter", "", 1) }

var tagRe = regexp.MustCompile(`<[^>]+>`)

// rich wraps the effect text in a paragraph, closing it before a list of
// modes rather than nesting the list inside, as the rest of the catalog's
// rich text does.
func rich(effect string) string {
	if effect == "" {
		return ""
	}
	if head, tail, ok := strings.Cut(effect, "<ul>"); ok {
		return "<p>" + head + "</p><ul>" + tail
	}
	return "<p>" + effect + "</p>"
}

func orientation(typ string) string {
	if typ == cards.TypeBattlefield {
		return "landscape"
	}
	return "portrait"
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func number(f flex) *int {
	n, err := strconv.Atoi(string(f))
	if err != nil {
		return nil
	}
	return &n
}

// positive drops a zero, which the API writes for cards with no power where
// the catalog leaves the field out.
func positive(n *int) *int {
	if n == nil || *n == 0 {
		return nil
	}
	return n
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// published renders a set's release date, a Unix timestamp in the API, the
// way sets.json spells it.
func published(ts flex) (string, error) {
	secs, err := strconv.ParseInt(string(ts), 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to read release date %q: %w", ts, err)
	}
	return time.Unix(secs, 0).UTC().Format("2006-01-02T15:04:05"), nil
}
