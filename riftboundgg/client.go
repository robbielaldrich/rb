package riftboundgg

import (
	"bytes"
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
	cardsURL = "https://api.dotgg.gg/cgfw/getcards?game=riftbound&mode=indexed"
	setsURL  = "https://api.dotgg.gg/cgfw/getsets?game=riftbound"
)

// apiSet is one set as the sets endpoint lists it. Only the sets a booster
// box is opened from are listed; the bundles and promo runs riftbound.gg also
// files cards under are not.
type apiSet struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	CardsCount  flex   `json:"cardsCount"`
	ReleaseDate flex   `json:"ReleaseDate"`
}

// cardsResponse is the card endpoint's table: one list of column names, and
// a row of values per card in that column order.
type cardsResponse struct {
	Names []string            `json:"names"`
	Data  [][]json.RawMessage `json:"data"`
}

// row is one card as the card endpoint describes it.
type row struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	NameNormal string   `json:"name_normal"`
	Effect     flex     `json:"effect"`
	Flavor     flex     `json:"flavor"`
	Color      []string `json:"color"`
	Cost       flex     `json:"cost"`
	Power      flex     `json:"power"`
	Might      flex     `json:"might"`
	Type       []string `json:"type"`
	Supertype  flex     `json:"supertype"`
	Tags       []string `json:"tags"`
	Rarity     string   `json:"rarity"`
	Image      flex     `json:"image"`
	Promo      flex     `json:"promo"`
	MarketIDs  flex     `json:"marketIds"`
}

// flex is a scalar the API sends as a string, a number or null depending on
// the card, read as the string it spells.
type flex string

func (f *flex) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*f = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flex(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("failed to read %s as a string or number: %w", data, err)
	}
	*f = flex(n.String())
	return nil
}

func fetchSets(client *http.Client) ([]apiSet, error) {
	var sets []apiSet
	if err := getJSON(client, setsURL, &sets); err != nil {
		return nil, err
	}
	return sets, nil
}

func fetchRows(client *http.Client) ([]row, error) {
	var resp cardsResponse
	if err := getJSON(client, cardsURL, &resp); err != nil {
		return nil, err
	}

	rows := make([]row, 0, len(resp.Data))
	for i, values := range resp.Data {
		if len(values) != len(resp.Names) {
			return nil, fmt.Errorf("card %d has %d values for %d columns", i, len(values), len(resp.Names))
		}
		fields := make(map[string]json.RawMessage, len(values))
		for j, name := range resp.Names {
			fields[name] = values[j]
		}
		data, err := json.Marshal(fields)
		if err != nil {
			return nil, fmt.Errorf("failed to regroup card %d: %w", i, err)
		}
		var r row
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("failed to parse card %d: %w", i, err)
		}
		rows = append(rows, r)
	}
	return rows, nil
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

// retry runs fn up to maxAttempts times, backing off between failures, so
// that one dropped request among the thousand-odd image downloads doesn't
// abort a run that is most of the way done.
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
