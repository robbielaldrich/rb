package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"rb/ankigen"
	"rb/collection"
	"rb/riftcodex"
)

var commands = []string{"download-cards", "collect", "gen-anki"}

func bind(cmd string, fs *flag.FlagSet) func() error {
	switch cmd {
	case "download-cards":
		outDir := fs.String("out", "cards", "directory to write downloaded card data into")
		images := fs.Bool("images", true, "also download card images")
		concurrency := fs.Int("concurrency", 8, "number of concurrent image downloads")
		return func() error { return downloadCards(*outDir, *images, *concurrency) }

	case "collect":
		catalogFile := fs.String("catalog-file", "cards/cards.json", "card catalog to search (cards.json)")
		collectionFile := fs.String("collection-file", "collection/collection.json", "collection file to read and write")
		return func() error { return collect(*collectionFile, *catalogFile) }

	case "gen-anki":
		var opts ankigen.Options
		fs.StringVar(&opts.CatalogPath, "catalog-file", "cards/cards.json", "card catalog to build the deck from")
		fs.StringVar(&opts.ImageDir, "image-dir", "cards/images", "directory holding the downloaded card scans")
		fs.StringVar(&opts.OutDir, "out", "anki", "directory to write the deck file and its media into")
		fs.StringVar(&opts.DeckName, "deck", "Riftbound::Hidden Costs", "name of the deck to import into")
		fs.Float64Var(&opts.MaskFraction, "mask", 1.0/3.0, "fraction of the card height to paint out, from the top")
		fs.IntVar(&opts.ImageWidth, "image-width", 500, "width to scale card images to, or 0 to keep them full size")
		fs.BoolVar(&opts.AllPrintings, "all-printings", false, "make a note per printing rather than per card")
		return func() error { return genAnki(opts) }
	}
	return nil
}

func Run(args []string) error {
	if len(args) == 0 {
		return usageError{"no command given"}
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "help", "-h", "-help", "--help":
		Usage(os.Stdout)
		return nil
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	run := bind(cmd, fs)
	if run == nil {
		return usageError{fmt.Sprintf("unknown command %q", cmd)}
	}

	switch err := fs.Parse(args); {
	case errors.Is(err, flag.ErrHelp):
		return nil
	case err != nil:
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	return run()
}

func Usage(w io.Writer) {
	fmt.Fprint(w, "usage: rb <command> [flags]\n")
	for _, cmd := range commands {
		fmt.Fprintf(w, "\n%s\n", cmd)

		fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
		bind(cmd, fs)
		for _, f := range flagLines(fs) {
			fmt.Fprintf(w, "%s\n", f)
		}
	}
}

func flagLines(fs *flag.FlagSet) []string {
	var names, helps []string
	fs.VisitAll(func(f *flag.Flag) {
		typ, help := flag.UnquoteUsage(f)
		name := "-" + f.Name
		if typ != "" {
			name += " " + typ
		}
		if f.DefValue != "false" {
			if typ == "string" {
				help += fmt.Sprintf(" (default %q)", f.DefValue)
			} else {
				help += fmt.Sprintf(" (default %s)", f.DefValue)
			}
		}
		names, helps = append(names, name), append(helps, help)
	})

	width := 0
	for _, n := range names {
		width = max(width, len(n))
	}
	lines := make([]string, len(names))
	for i := range names {
		lines[i] = fmt.Sprintf("  %-*s   %s", width, names[i], helps[i])
	}
	return lines
}

func downloadCards(outDir string, images bool, concurrency int) error {
	if err := riftcodex.DownloadCards(outDir, images, concurrency); err != nil {
		return fmt.Errorf("failed to download cards: %w", err)
	}
	return nil
}

func collect(collectionFile, catalogFile string) error {
	if err := collection.RunEditor(collectionFile, catalogFile); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}
	return nil
}

func genAnki(opts ankigen.Options) error {
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

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func IsUsage(err error) bool {
	var u usageError
	return errors.As(err, &u)
}
