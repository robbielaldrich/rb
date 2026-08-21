package cli

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"rb/collection"
	"rb/riftcodex"
)

// RunDownloadCards mirrors the Riftcodex card database onto local disk.
func RunDownloadCards(args []string) error {
	fs := flag.NewFlagSet("download-cards", flag.ContinueOnError)
	outDir := fs.String("out", "cards", "directory to write downloaded card data into")
	images := fs.Bool("images", true, "also download card images")
	concurrency := fs.Int("concurrency", 8, "number of concurrent image downloads")
	if run, err := parse(fs, args); !run {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	if err := riftcodex.DownloadCards(*outDir, *images, *concurrency); err != nil {
		return fmt.Errorf("failed to download cards: %w", err)
	}

	return nil
}

// RunCollection opens the interactive collection tracker.
func RunCollection(args []string) error {
	fs := flag.NewFlagSet("collection", flag.ContinueOnError)
	cardsDir := fs.String("cards", "cards", "directory holding cards.json")
	file := fs.String("file", "collection.json", "collection file to read and write")
	if run, err := parse(fs, args); !run {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	if err := collection.RunEditor(*file, filepath.Join(*cardsDir, "cards.json")); err != nil {
	}
	return nil
}

// parse reports flag failures to the caller rather than exiting behind its
// back, and says whether the command should go on to run. -h is not a
// failure: the flag package has printed the usage, so there is nothing left
// to do.
func parse(fs *flag.FlagSet, args []string) (run bool, err error) {
	switch err := fs.Parse(args); {
	case errors.Is(err, flag.ErrHelp):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to parse flags: %w", err)
	}
	return true, nil
}
