package riftcodex

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL  = "https://api.riftcodex.com"
	pageSize = 100
)

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

func fetchSets(client *http.Client) ([]Set, error) {
	url := fmt.Sprintf("%s/sets?size=100", baseURL)
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

	type cardsResponse struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
		Page  int               `json:"page"`
		Size  int               `json:"size"`
		Pages int               `json:"pages"`
	}

	page := 1
	for {
		url := fmt.Sprintf("%s/cards?size=%d&page=%d&set_id=%s&sort=collector_number", baseURL, pageSize, page, setID)
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
