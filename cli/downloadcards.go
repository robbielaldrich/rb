// Package cli implements the rb subcommands. Each command exposes a single
// Run* entry point that parses its own flags and returns an error rather than
// exiting, leaving the exit decision to main.
package cli

import (
	"encoding/json"
	"errors"
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

// RunDownloadCards downloads all card data from the Riftcodex API
// (https://riftcodex.com/docs/endpoints/cards/) and writes it to disk as
// sets.json, cards.json, and one image per card under images/. Image
// downloads are idempotent: cards whose image file already exists on disk
// are skipped.
func RunDownloadCards(args []string) error {
	fs := flag.NewFlagSet("download-cards", flag.ContinueOnError)
	outDir := fs.String("out", "cards", "directory to write downloaded card data into")
	images := fs.Bool("images", true, "also download card images")
	concurrency := fs.Int("concurrency", 8, "number of concurrent image downloads")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	if err := downloadCards(client, *outDir, *images, *concurrency); err != nil {
		return fmt.Errorf("failed to download cards: %w", err)
	}
	return nil
}

type setsResponse struct {
	Items []Set `json:"items"`
	Total int   `json:"total"`
	Pages int   `json:"pages"`
}

// Set is a Riftcodex card set, as returned by GET /sets.
type Set struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SetID        string `json:"set_id"`
	CardCount    int    `json:"card_count"`
	TCGPlayerID  any    `json:"tcgplayer_id"`
	CardmarketID any    `json:"cardmarket_id"`
	PublishedOn  string `json:"published_on"`
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

func downloadCards(client *http.Client, outDir string, downloadImages bool, concurrency int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir %s: %w", outDir, err)
	}

	sets, err := fetchSets(client)
	if err != nil {
		return fmt.Errorf("failed to fetch sets: %w", err)
	}
	if err := writeJSON(filepath.Join(outDir, "sets.json"), sets); err != nil {
		return fmt.Errorf("failed to write sets.json: %w", err)
	}
	log.Printf("fetched %d sets", len(sets))

	var allCards []json.RawMessage
	for _, s := range sets {
		setCards, err := fetchAllCards(client, s.SetID)
		if err != nil {
			return fmt.Errorf("failed to fetch cards for set %s: %w", s.SetID, err)
		}
		allCards = append(allCards, setCards...)
		log.Printf("set %-6s (%s): %d cards", s.SetID, s.Name, len(setCards))
	}

	if err := writeJSON(filepath.Join(outDir, "cards.json"), allCards); err != nil {
		return fmt.Errorf("failed to write cards.json: %w", err)
	}
	log.Printf("wrote %d total cards across %d sets to %s", len(allCards), len(sets), outDir)

	if downloadImages {
		imagesDir := filepath.Join(outDir, "images")
		if err := os.MkdirAll(imagesDir, 0o755); err != nil {
			return fmt.Errorf("failed to create images dir %s: %w", imagesDir, err)
		}
		if err := downloadAllImages(client, imagesDir, allCards, concurrency); err != nil {
			return fmt.Errorf("failed to download images: %w", err)
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
			return fmt.Errorf("failed to parse card for image download: %w", err)
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
					firstErr = fmt.Errorf("failed to download %s: %w", imageURL, err)
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
	return retry(func() error {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to build request for %s: %w", url, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to GET %s: %w", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return fmt.Errorf("GET %s: unexpected status %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
		}

		tmp := dest + ".tmp"
		f, err := os.Create(tmp)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", tmp, err)
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("failed to write %s: %w", tmp, err)
		}
		if err := f.Close(); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("failed to close %s: %w", tmp, err)
		}

		if err := os.Rename(tmp, dest); err != nil {
			return fmt.Errorf("failed to rename %s to %s: %w", tmp, dest, err)
		}
		return nil
	})
}

func fetchSets(client *http.Client) ([]Set, error) {
	url := fmt.Sprintf("%s/sets?size=100", apiBase)
	var resp setsResponse
	if err := getJSON(client, url, &resp); err != nil {
		return nil, fmt.Errorf("failed to fetch sets: %w", err)
	}
	if resp.Pages > 1 {
		return nil, fmt.Errorf("unexpectedly more than one page of sets (pages=%d); pagination for /sets is not implemented", resp.Pages)
	}
	return resp.Items, nil
}

func fetchAllCards(client *http.Client, setID string) ([]json.RawMessage, error) {
	var out []json.RawMessage

	page := 1
	for {
		url := fmt.Sprintf("%s/cards?size=%d&page=%d&set_id=%s&sort=collector_number", apiBase, pageSize, page, setID)
		var resp cardsResponse
		if err := getJSON(client, url, &resp); err != nil {
			return nil, fmt.Errorf("failed to fetch page %d for set %s: %w", page, setID, err)
		}
		out = append(out, resp.Items...)

		if page >= resp.Pages || resp.Pages == 0 {
			break
		}
		page++
	}

	return out, nil
}

func getJSON(client *http.Client, url string, out any) error {
	return retry(func() error {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to build request for %s: %w", url, err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to GET %s: %w", url, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response from %s: %w", url, err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("GET %s: unexpected status %s: %s", url, resp.Status, body)
		}

		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("failed to parse response from %s: %w", url, err)
		}
		return nil
	})
}

// retry runs fn up to maxAttempts times, backing off between failures. The
// Riftcodex API intermittently stalls past the client timeout on larger
// pages, which would otherwise abort a download that is most of the way done.
func retry(fn func() error) error {
	const maxAttempts = 4

	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt < maxAttempts {
			backoff := time.Duration(attempt) * 2 * time.Second
			log.Printf("attempt %d/%d failed (%v); retrying in %s", attempt, maxAttempts, err, backoff)
			time.Sleep(backoff)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, err)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
