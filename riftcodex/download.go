package riftcodex

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

const requestTimeout = 30 * time.Second

// cardMedia captures just the fields of a card needed to download its image.
type cardMedia struct {
	RiftboundID string `json:"riftbound_id"`
	Media       struct {
		ImageURL string `json:"image_url"`
	} `json:"media"`
}

// DownloadCards downloads all card data from the Riftcodex API and writes it
// to outDir as sets.json, cards.json, and one image per card under images/.
// Image downloads are idempotent: cards whose image file already exists on
// disk are skipped.
func DownloadCards(outDir string, downloadImages bool, concurrency int) error {
	client := &http.Client{Timeout: requestTimeout}

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
