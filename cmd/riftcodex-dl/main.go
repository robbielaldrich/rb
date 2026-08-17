// Command riftcodex-dl downloads all card data from the Riftcodex API
// (https://riftcodex.com/docs/endpoints/cards/) and writes it to disk as one
// JSON file per set, plus a combined cards.json and sets.json.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	apiBase  = "https://api.riftcodex.com"
	pageSize = 100
)

type setsResponse struct {
	Items []Set `json:"items"`
	Total int   `json:"total"`
	Pages int   `json:"pages"`
}

// Set is a Riftcodex card set, as returned by GET /sets.
type Set struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	SetID        string      `json:"set_id"`
	CardCount    int         `json:"card_count"`
	TCGPlayerID  interface{} `json:"tcgplayer_id"`
	CardmarketID interface{} `json:"cardmarket_id"`
	PublishedOn  string      `json:"published_on"`
}

type cardsResponse struct {
	Items []json.RawMessage `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
	Pages int               `json:"pages"`
}

func main() {
	outDir := flag.String("out", "riftcodex_cards", "directory to write downloaded card data into")
	flag.Parse()

	client := &http.Client{Timeout: 30 * time.Second}

	if err := run(client, *outDir); err != nil {
		log.Fatalf("riftcodex-dl: %v", err)
	}
}

func run(client *http.Client, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	sets, err := fetchSets(client)
	if err != nil {
		return fmt.Errorf("fetching sets: %w", err)
	}
	if err := writeJSON(filepath.Join(outDir, "sets.json"), sets); err != nil {
		return fmt.Errorf("writing sets.json: %w", err)
	}
	log.Printf("fetched %d sets", len(sets))

	var allCards []json.RawMessage
	for _, s := range sets {
		cards, err := fetchAllCards(client, s.SetID)
		if err != nil {
			return fmt.Errorf("fetching cards for set %s: %w", s.SetID, err)
		}
		allCards = append(allCards, cards...)

		setFile := filepath.Join(outDir, fmt.Sprintf("%s.json", normalizeSetID(s.SetID)))
		if err := writeJSON(setFile, cards); err != nil {
			return fmt.Errorf("writing %s: %w", setFile, err)
		}
		log.Printf("set %-6s (%s): %d cards", s.SetID, s.Name, len(cards))
	}

	if err := writeJSON(filepath.Join(outDir, "cards.json"), allCards); err != nil {
		return fmt.Errorf("writing cards.json: %w", err)
	}
	log.Printf("done: %d total cards across %d sets written to %s", len(allCards), len(sets), outDir)

	return nil
}

func normalizeSetID(id string) string {
	out := make([]rune, 0, len(id))
	for _, r := range id {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		out = append(out, r)
	}
	return string(out)
}

func fetchSets(client *http.Client) ([]Set, error) {
	url := fmt.Sprintf("%s/sets?size=100", apiBase)
	var resp setsResponse
	if err := getJSON(client, url, &resp); err != nil {
		return nil, err
	}
	if resp.Pages > 1 {
		return nil, fmt.Errorf("unexpectedly more than one page of sets (pages=%d); pagination for /sets is not implemented", resp.Pages)
	}
	return resp.Items, nil
}

func fetchAllCards(client *http.Client, setID string) ([]json.RawMessage, error) {
	var cards []json.RawMessage

	page := 1
	for {
		url := fmt.Sprintf("%s/cards?size=%d&page=%d&set_id=%s&sort=collector_number", apiBase, pageSize, page, setID)
		var resp cardsResponse
		if err := getJSON(client, url, &resp); err != nil {
			return nil, err
		}
		cards = append(cards, resp.Items...)

		if page >= resp.Pages || resp.Pages == 0 {
			break
		}
		page++
	}

	return cards, nil
}

func getJSON(client *http.Client, url string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s: %s", url, resp.Status, body)
	}

	return json.Unmarshal(body, out)
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
