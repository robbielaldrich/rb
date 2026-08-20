package collection

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"rb/cards"
)

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
