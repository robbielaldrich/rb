// Command riftcodex-dl downloads all card data from the Riftcodex API
// (https://riftcodex.com/docs/endpoints/cards/) and writes it to disk as
// sets.json, cards.json, and one image per card under images/. Image
// downloads are idempotent: cards whose image file already exists on disk
// are skipped.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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

// cardMedia captures just the fields of a card needed to download its image.
type cardMedia struct {
	RiftboundID string `json:"riftbound_id"`
	Media       struct {
		ImageURL string `json:"image_url"`
	} `json:"media"`
}

func main() {
	outDir := flag.String("out", "riftcodex_cards", "directory to write downloaded card data into")
	images := flag.Bool("images", true, "also download card images")
	concurrency := flag.Int("concurrency", 8, "number of concurrent image downloads")
	flag.Parse()

	client := &http.Client{Timeout: 30 * time.Second}

	if err := run(client, *outDir, *images, *concurrency); err != nil {
		log.Fatalf("riftcodex-dl: %v", err)
	}
}

func run(client *http.Client, outDir string, downloadImages bool, concurrency int) error {
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
		log.Printf("set %-6s (%s): %d cards", s.SetID, s.Name, len(cards))
	}

	if err := writeJSON(filepath.Join(outDir, "cards.json"), allCards); err != nil {
		return fmt.Errorf("writing cards.json: %w", err)
	}
	log.Printf("wrote %d total cards across %d sets to %s", len(allCards), len(sets), outDir)

	if downloadImages {
		imagesDir := filepath.Join(outDir, "images")
		if err := os.MkdirAll(imagesDir, 0o755); err != nil {
			return fmt.Errorf("creating images dir: %w", err)
		}
		if err := downloadAllImages(client, imagesDir, allCards, concurrency); err != nil {
			return fmt.Errorf("downloading images: %w", err)
		}
	}

	return nil
}

// downloadAllImages fetches each distinct card image into imagesDir,
// skipping any card whose image file is already present so re-runs only
// fetch what's missing. Multiple card entries can share a riftbound_id (e.g.
// foil variants), so images are deduplicated by riftbound_id first.
func downloadAllImages(client *http.Client, imagesDir string, rawCards []json.RawMessage, concurrency int) error {
	var (
		mu               sync.Mutex
		downloaded, skip int
		firstErr         error
	)

	byRiftboundID := make(map[string]string, len(rawCards))
	for _, raw := range rawCards {
		var c cardMedia
		if err := json.Unmarshal(raw, &c); err != nil {
			return fmt.Errorf("parsing card for image download: %w", err)
		}
		if c.RiftboundID == "" || c.Media.ImageURL == "" {
			continue
		}
		byRiftboundID[c.RiftboundID] = c.Media.ImageURL
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for riftboundID, imageURL := range byRiftboundID {
		dest := filepath.Join(imagesDir, riftboundID+imageExt(imageURL))
		if fileExists(dest) {
			mu.Lock()
			skip++
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(imageURL, dest string) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := downloadFile(client, imageURL, dest); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("downloading %s: %w", imageURL, err)
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			downloaded++
			mu.Unlock()
		}(imageURL, dest)
	}

	wg.Wait()

	log.Printf("images: %d downloaded, %d already present", downloaded, skip)
	return firstErr
}

func imageExt(imageURL string) string {
	u, err := url.Parse(imageURL)
	if err != nil {
		return ".png"
	}
	if ext := path.Ext(u.Path); ext != "" {
		return ext
	}
	return ".png"
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

func downloadFile(client *http.Client, url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, dest)
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
