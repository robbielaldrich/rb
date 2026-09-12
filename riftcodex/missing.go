package riftcodex

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// missingFile is the hand-maintained set of cards that are in print but absent
// from the Riftcodex dataset, kept in the shape the API returns so they can be
// merged into cards.json as they are.
type missingFile struct {
	Note    string            `json:"note"`
	Sources []string          `json:"sources"`
	Cards   []json.RawMessage `json:"cards"`
}

// cardID reads just enough of a card to recognise it.
type cardID struct {
	RiftboundID string `json:"riftbound_id"`
	Name        string `json:"name"`
	Set         struct {
		SetID string `json:"set_id"`
	} `json:"set"`
}

// riftIDRe matches the riftbound_id shape, e.g. opp-069b-166, whose parts are
// the set, the collector number, and the size of the set being printed or
// reprinted. The middle part is left loose where package cards pins it to a
// number, so that the runes and promos numbered outside the main run — r01b,
// sp1 — are read for their set size too rather than falling through with none.
var riftIDRe = regexp.MustCompile(`^([a-z]+)-([0-9a-z*]+)-([0-9]+)$`)

// printing identifies the card a record is a printing of, rather than the
// record itself: the set it is filed under, the card it shows, and the set it
// reprints. Riftcodex files organized-play promos under OPP and numbers them
// with the size of the set they reprint, so this is what tells an Origins
// promo of a card from a Vendetta promo of the same card.
type printing struct{ setID, name, reprints string }

func (c cardID) printing() printing {
	reprints := ""
	if m := riftIDRe.FindStringSubmatch(c.RiftboundID); m != nil {
		reprints = m[3]
	}
	return printing{
		setID:    strings.ToUpper(c.Set.SetID),
		name:     strings.ToLower(baseName(c.Name)),
		reprints: reprints,
	}
}

// variantRe matches the parenthesised suffix that tells reprints of one card
// apart, e.g. "Fury Rune (Nexus Night Promo)".
var variantRe = regexp.MustCompile(`\s*\([^()]*\)$`)

// baseName is a card's name without that suffix, so an entry here matches the
// card however Riftcodex ends up labelling its printing of it.
func baseName(name string) string { return variantRe.ReplaceAllString(name, "") }

// loadMissing reads the overlay file. No file is not an error: it only means
// there is nothing to add.
func loadMissing(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var f missingFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return f.Cards, nil
}

// mergeMissing appends the hand-maintained cards to the downloaded dump, after
// asserting that the dump still carries none of them.
//
// The assertion is the point of the file: each entry is only there because
// Riftcodex lacks it, so one that has since appeared upstream is no longer
// ours to supply. Letting the two coexist would leave two records for the one
// piece of cardboard, so the download stops instead and names what to delete.
//
// What is matched is the printing rather than the record, since the id an
// entry carries can only be a guess until Riftcodex assigns one: a Vendetta
// Nexus Night promo of Mel is the OPP printing of Mel numbered against
// Vendetta's 166 cards, whatever collector number it is eventually given. The
// exact id is checked too, for the case where the guess was right.
func mergeMissing(dump, extra []json.RawMessage, path string) ([]json.RawMessage, error) {
	if len(extra) == 0 {
		return dump, nil
	}

	ids := make(map[string]bool, len(dump))
	printings := make(map[printing]bool, len(dump))
	for _, raw := range dump {
		var c cardID
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to parse a downloaded card: %w", err)
		}
		ids[c.RiftboundID] = true
		printings[c.printing()] = true
	}

	var found []string
	for _, raw := range extra {
		var c cardID
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("failed to parse a card in %s: %w", path, err)
		}
		if ids[c.RiftboundID] || printings[c.printing()] {
			found = append(found, fmt.Sprintf("%s (%s)", c.RiftboundID, c.Name))
		}
	}
	if len(found) > 0 {
		return nil, fmt.Errorf("Riftcodex now carries %d of the %d cards in %s, which should be deleted from it: %s",
			len(found), len(extra), path, strings.Join(found, ", "))
	}

	return append(dump, extra...), nil
}
