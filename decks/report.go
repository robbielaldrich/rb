package decks

import (
	"encoding/json"
	"fmt"
)

// The report is written twice: once as the text a terminal shows, and once as
// the JSON behind it. The text columns are sized to the longest name in each
// deck, so they can't be parsed back reliably — anything that wants the
// results as data reads this instead of the report it mirrors.
type reportData struct {
	Sideboard bool       `json:"sideboard"`
	Decks     []deckData `json:"decks"`
}

type deckData struct {
	Name      string        `json:"name"`
	Size      int           `json:"size"`
	Short     int           `json:"short"`
	Buildable bool          `json:"buildable"`
	Missing   []missingData `json:"missing"`
}

type missingData struct {
	Name  string `json:"name"`
	Need  int    `json:"need"`
	Have  int    `json:"have"`
	Short int    `json:"short"`
	// Unknown and Nearest carry the same warning the text report prints for a
	// name no card is filed under, so the page can flag a likely typo too.
	Unknown bool     `json:"unknown,omitempty"`
	Nearest string   `json:"nearest,omitempty"`
	Sets    []string `json:"sets,omitempty"`
	Domains []string `json:"domains,omitempty"`
}

func writeData(path string, res []Result, sideboard bool) error {
	out := reportData{Sideboard: sideboard, Decks: make([]deckData, len(res))}
	for i, r := range res {
		d := deckData{
			Name:      r.Deck.Name,
			Size:      r.Size,
			Short:     r.Short(),
			Buildable: r.Buildable(),
			// An empty list marshals as null unless it is made, and a page
			// that reads it shouldn't have to guard every deck.
			Missing: make([]missingData, len(r.Missing)),
		}
		for j, m := range r.Missing {
			d.Missing[j] = missingData{
				Name:    m.Name,
				Need:    m.Need,
				Have:    m.Have,
				Short:   m.short(),
				Unknown: m.Unknown,
				Nearest: m.Nearest,
				Sets:    m.Sets,
				Domains: m.Domains,
			}
		}
		out.Decks[i] = d
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal the report data: %w", err)
	}
	return writeFile(path, append(data, '\n'))
}
