package riftboundgg

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"rb/cards"
)

const requestTimeout = 30 * time.Second

// DownloadCards downloads all card data from riftbound.gg and writes it to
// outDir as sets.json, cards.json, and one image per card under images/.
// Image downloads are idempotent: cards whose image file already exists on
// disk are skipped.
func DownloadCards(outDir string, downloadImages bool, concurrency int) error {
	client := &http.Client{Timeout: requestTimeout}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output dir %s: %w", outDir, err)
	}

	apiSets, err := fetchSets(client)
	if err != nil {
		return fmt.Errorf("failed to fetch sets: %w", err)
	}
	rows, err := fetchRows(client)
	if err != nil {
		return fmt.Errorf("failed to fetch cards: %w", err)
	}

	sets, allCards, err := build(apiSets, rows)
	if err != nil {
		return err
	}

	if err := writeJSON(filepath.Join(outDir, "sets.json"), sets); err != nil {
		return fmt.Errorf("failed to write sets.json: %w", err)
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

// build converts the API's sets and cards into the catalog.
//
// Only cards filed under a listed set are kept. riftbound.gg also files a
// handful of bundle and event printings under codes of their own (ARC, SGN,
// T1A, ...), and oversized copies under ids like OGN-279/298; none are cards
// a deck or a booster holds, so they are left out, and named in the log in
// case one ever should be.
func build(apiSets []apiSet, rows []row) ([]cards.Set, []cards.Card, error) {
	bySet := map[string][]row{}
	var unlisted []string
	listed := map[string]bool{}
	for _, s := range apiSets {
		listed[s.Code] = true
	}
	for _, r := range rows {
		p, ok := parseID(r.ID)
		if !ok || !listed[p.set] {
			unlisted = append(unlisted, r.ID)
			continue
		}
		bySet[p.set] = append(bySet[p.set], r)
	}
	if len(unlisted) > 0 {
		log.Printf("left out %d cards outside the listed sets, or with ids it can't read: %s", len(unlisted), strings.Join(unlisted, ", "))
	}

	plain := plainPrintings(rows)
	var sets []cards.Set
	var out []cards.Card
	seen := map[string]string{}
	for _, as := range apiSets {
		at, err := published(as.ReleaseDate)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read set %s: %w", as.Code, err)
		}

		s := newSetInfo(as.Code, as.Name, bySet[as.Code], plain)
		if s.size == 0 {
			return nil, nil, fmt.Errorf("set %s has no main-run cards to number it by", as.Code)
		}
		n := 0
		for _, r := range bySet[as.Code] {
			p, _ := parseID(r.ID)
			c, ok := convert(r, p, s)
			if !ok {
				continue
			}
			// Two rows landing on one id would leave cards.Load keeping
			// whichever it met last, so a scheme that can't tell them apart
			// is an error rather than a silent loss.
			if prev, dup := seen[c.RiftboundID]; dup {
				return nil, nil, fmt.Errorf("%s and %s both convert to %s", prev, r.ID, c.RiftboundID)
			}
			seen[c.RiftboundID] = r.ID
			out = append(out, c)
			n++
		}

		count, _ := strconv.Atoi(string(as.CardsCount))
		sets = append(sets, cards.Set{
			ID:          as.Code,
			Name:        as.Name,
			SetID:       as.Code,
			CardCount:   count,
			PublishedOn: at,
		})
		log.Printf("set %-4s (%s): %d cards, numbered out of %d", as.Code, as.Name, n, s.size)
	}
	return sets, out, nil
}

// downloadAllImages fetches each card's image into imagesDir, skipping any
// card that already has one, in whichever format it was saved: the scans
// downloaded from Riftcodex before it was replaced are PNGs, and are kept so
// that the images made from them don't all change.
func downloadAllImages(client *http.Client, imagesDir string, all []cards.Card, concurrency int) error {
	var (
		mu               sync.Mutex
		downloaded, skip int
		firstErr         error
	)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, c := range all {
		if c.Media.ImageURL == "" {
			continue
		}
		// Counted on the dispatching goroutine, so unlike the tallies the
		// workers keep it needs no lock of its own.
		if fileExists(filepath.Join(imagesDir, c.RiftboundID+".png")) {
			skip++
			continue
		}
		dest := filepath.Join(imagesDir, c.RiftboundID+imageExt(c.Media.ImageURL))
		if fileExists(dest) {
			skip++
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
		}(c.Media.ImageURL, dest)
	}

	wg.Wait()

	log.Printf("images: %d downloaded, %d already present", downloaded, skip)
	return firstErr
}

func imageExt(imageURL string) string {
	if ext := path.Ext(imageURL); ext != "" && !strings.ContainsAny(ext, "?/") {
		return ext
	}
	return ".webp"
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
