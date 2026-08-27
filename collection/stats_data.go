package collection

import (
	"encoding/json"
	"fmt"
)

// The summary is written twice: once as the text a terminal shows, and once as
// the JSON behind it. The text is a tabwriter grid whose columns are sized to
// the widest set label, so it can't be parsed back reliably — anything that
// wants the numbers as data reads this instead of the table it mirrors.
type statsData struct {
	Sets  []setData `json:"sets"`
	Total setData   `json:"total"`
}

type setData struct {
	// The totals row stands for every set at once, so it carries no name.
	SetID string `json:"set,omitempty"`
	Label string `json:"label,omitempty"`

	Cards        tallyData  `json:"cards"`
	AlternateArt *tallyData `json:"alternate_art"`
	Overnumbered *tallyData `json:"overnumbered"`
	Signature    *tallyData `json:"signature"`
}

type tallyData struct {
	Owned int `json:"owned"`
	Total int `json:"total"`
}

// chase reports an extra-printing tally, or nil for a set that prints none of
// that kind at all. The page then shows the same "-" the text report does,
// rather than a 0/0 it would have to render as 0%.
func chase(t tally) *tallyData {
	if t.total == 0 {
		return nil
	}
	return &tallyData{Owned: t.owned, Total: t.total}
}

func row(s setStats) setData {
	return setData{
		SetID:        s.setID,
		Label:        s.label,
		Cards:        tallyData{Owned: s.named.owned, Total: s.named.total},
		AlternateArt: chase(s.alt),
		Overnumbered: chase(s.over),
		Signature:    chase(s.sig),
	}
}

func writeStatsData(path string, sets []setStats) error {
	out := statsData{Sets: make([]setData, len(sets)), Total: row(totals(sets))}
	for i, s := range sets {
		out.Sets[i] = row(s)
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal the summary data: %w", err)
	}
	return writeFile(path, append(data, '\n'))
}
