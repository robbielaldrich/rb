package cli

import (
	"errors"
	"flag"
	"fmt"

	"rb/ankigen"
	"rb/collection"
	"rb/riftcodex"
)

// RunDownloadCards mirrors the Riftcodex card database onto local disk.
func RunDownloadCards(args []string) error {
	fs := flag.NewFlagSet("download-cards", flag.ContinueOnError)
	outDir := fs.String("out", "catalog", "directory to write downloaded card data into")
	images := fs.Bool("images", true, "also download card images")
	concurrency := fs.Int("concurrency", 8, "number of concurrent image downloads")
	if run, err := parse(fs, args); !run {
		return err
	}

	if err := riftcodex.DownloadCards(*outDir, *images, *concurrency); err != nil {
		return fmt.Errorf("failed to download cards: %w", err)
	}

	return nil
}

// RunCollection opens the interactive collection tracker.
func RunCollection(args []string) error {
	fs := flag.NewFlagSet("collection", flag.ContinueOnError)
	catalogFile := fs.String("catalog-file", "catalog/cards.json", "directory holding card catalog (cards.json)")
	collectionFile := fs.String("collection-file", "collection/collection.json", "collection file to read and write")
	if run, err := parse(fs, args); !run {
		return err
	}

	if err := collection.RunEditor(*collectionFile, *catalogFile); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}
	return nil
}

// RunAnkiGen writes an Anki deck built from the card catalog.
func RunAnkiGen(args []string) error {
	fs := flag.NewFlagSet("anki-gen", flag.ContinueOnError)
	opts := ankigen.Options{}
	fs.StringVar(&opts.CatalogPath, "catalog-file", "catalog/cards.json", "card catalog to build the deck from")
	fs.StringVar(&opts.ImageDir, "image-dir", "catalog/images", "directory holding the downloaded card scans")
	fs.StringVar(&opts.OutDir, "out", "anki", "directory to write the deck file and its media into")
	fs.StringVar(&opts.DeckName, "deck", "Riftbound::Hidden Costs", "name of the deck to import into")
	fs.Float64Var(&opts.MaskFraction, "mask", 1.0/3.0, "fraction of the card height to paint out, from the top")
	fs.IntVar(&opts.ImageWidth, "image-width", 500, "width to scale card images to, or 0 to keep them full size")
	fs.BoolVar(&opts.AllPrintings, "all-printings", false, "make a note per printing rather than per card")
	if run, err := parse(fs, args); !run {
		return err
	}
	if opts.MaskFraction <= 0 || opts.MaskFraction > 1 {
		return fmt.Errorf("-mask must be between 0 and 1, got %v", opts.MaskFraction)
	}

	res, err := ankigen.GenerateHiddenCosts(opts)
	if err != nil {
		return fmt.Errorf("failed to generate deck: %w", err)
	}

	fmt.Printf(`wrote %d notes and %d images

to import:
  1. copy %s/* into your Anki collection.media folder
     (Anki: Tools > Check Media > View Files)
  2. in Anki, File > Import and choose %s
`, res.Notes, res.Images, res.MediaDir, res.DeckFile)
	return nil
}

// parse reports flag failures to the caller, already wrapped, rather than
// exiting behind its back, and says whether the command should go on to run.
// -h is not a failure: the flag package has printed the usage, so there is
// nothing left to do and the returned error is nil.
func parse(fs *flag.FlagSet, args []string) (run bool, err error) {
	switch err := fs.Parse(args); {
	case errors.Is(err, flag.ErrHelp):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to parse flags: %w", err)
	}
	return true, nil
}
