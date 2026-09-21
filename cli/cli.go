package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"rb/ankigen"
	"rb/collection"
	"rb/decks"
	"rb/riftcodex"
)

var commands = []string{"download-cards", "collect", "validate", "collection-stats", "add-decks", "match-decks", "gen-anki", "gen-rules-anki", "add-anki-media", "missing"}

func bind(cmd string, fs *flag.FlagSet) func() error {
	switch cmd {
	case "download-cards":
		outDir := fs.String("out", "cards", "directory to write downloaded card data into")
		missingFile := fs.String("missing-file", "cards/missing-from-riftcodex.json", "cards Riftcodex does not carry, to add to the download")
		images := fs.Bool("images", true, "also download card images")
		concurrency := fs.Int("concurrency", 8, "number of concurrent image downloads")
		return func() error { return downloadCards(*outDir, *missingFile, *images, *concurrency) }

	case "collect":
		catalogFile := fs.String("catalog-file", "cards/cards.json", "card catalog to search (cards.json)")
		collectionFile := fs.String("collection-file", "collection/collection.json", "collection file to read and write")
		return func() error { return collect(*collectionFile, *catalogFile, fs.Args()) }

	case "validate":
		collectionFile := fs.String("collection-file", "collection/collection.json", "collection file to check and correct")
		return func() error {
			if fs.NArg() > 1 {
				return fmt.Errorf("validate takes one optional set label, got %d arguments", fs.NArg())
			}
			return validate(*collectionFile, fs.Arg(0))
		}

	case "collection-stats":
		catalogFile := fs.String("catalog-file", "cards/cards.json", "card catalog to measure the collection against")
		collectionFile := fs.String("collection-file", "collection/collection.json", "collection file to read")
		dataFile := fs.String("json-out", "collection/collection-stats-result.json", "file to leave the summary data in for the collection page, or \"\" to leave none")
		return func() error { return collectionStats(*collectionFile, *catalogFile, *dataFile) }

	case "missing":
		catalogFile := fs.String("catalog-file", "cards/cards.json", "card catalog to measure the collection against")
		collectionFile := fs.String("collection-file", "collection/collection.json", "collection file to read")
		axisFlag := fs.String("axis", "playset", "which axis to measure short against: \"set\" (own a copy at all) or \"playset\" (own three)")
		return func() error {
			axis, err := collection.ParseAxis(*axisFlag)
			if err != nil {
				return err
			}
			return missing(*collectionFile, *catalogFile, fs.Args(), axis)
		}

	case "add-decks":
		catalogFile := fs.String("catalog-file", "cards/cards.json", "card catalog to check the pasted card names against")
		setsFile := fs.String("sets-file", "cards/sets.json", "set list to date each deck by the newest set in print")
		decksFile := fs.String("decks-file", "decks/decks.json", "deck register to append to")
		return func() error { return addDecks(*decksFile, *catalogFile, *setsFile) }

	case "match-decks":
		var opts decks.Options
		fs.StringVar(&opts.CatalogPath, "catalog-file", "cards/cards.json", "card catalog to read the deck names through")
		fs.StringVar(&opts.CollectionPath, "collection-file", "collection/collection.json", "collection to build the decks out of")
		fs.StringVar(&opts.DecksPath, "decks-file", "decks/decks.json", "deck register to measure")
		fs.StringVar(&opts.ReportPath, "out", "decks/match-decks-result.txt", "file to keep a copy of the report in, or \"\" to keep none")
		fs.BoolVar(&opts.Sideboard, "sideboard", false, "count the sideboard as part of the deck")
		return func() error { return matchDecks(opts) }

	case "gen-anki":
		var opts ankigen.Options
		fs.StringVar(&opts.CatalogPath, "catalog-file", "cards/cards.json", "card catalog to build the deck from")
		fs.StringVar(&opts.ImageDir, "image-dir", "cards/images", "directory holding the downloaded card scans")
		fs.StringVar(&opts.OutDir, "out", "anki", "directory to write the deck files and their media into")
		fs.StringVar(&opts.EffectDeckName, "effect-deck", "Riftbound::Hidden Effects", "name of the companion deck asking what a card does")
		fs.StringVar(&opts.ReactionDeckName, "reaction-deck", "Riftbound::Reaction Cards", "name of the deck listing each domain's Reaction cards")
		fs.StringVar(&opts.ActionDeckName, "action-deck", "Riftbound::Action Cards", "name of the deck listing each domain's Action cards")
		fs.StringVar(&opts.HiddenDomainDeckName, "hidden-domain-deck", "Riftbound::Hidden by Domain", "name of the deck asking which Hidden cards each domain holds")
		fs.Float64Var(&opts.EffectMaskFraction, "effect-mask", 0.4, "fraction of the card height to paint out from the bottom, for cards whose keyword badge can't be found")
		fs.IntVar(&opts.ImageWidth, "image-width", 500, "width to scale card images to, or 0 to keep them full size")
		fs.BoolVar(&opts.AllPrintings, "all-printings", false, "make a note per printing rather than per card")
		only := fs.String("only", "", "comma-separated blocks to generate, or empty for all (available: "+ankigen.BlockNames()+")")
		return func() error { return genAnki(opts, *only) }

	case "add-anki-media":
		defaultDir, _ := ankigen.DefaultAnkiDir()
		mediaDir := fs.String("media-dir", "anki/media", "directory holding the generated card images")
		ankiDir := fs.String("anki-dir", defaultDir, "Anki's data folder, holding one folder per profile")
		profile := fs.String("profile", "", "Anki profile to copy into, or empty to use the only one")
		return func() error { return addAnkiMedia(*mediaDir, *ankiDir, *profile) }

	case "gen-rules-anki":
		var opts ankigen.RulesOptions
		fs.StringVar(&opts.RulingsPath, "rulings-file", "rules/rulings.json", "ruling dataset to draft the notes from")
		fs.StringVar(&opts.ReviewPath, "review-file", "rules/anki-review.json", "record of what has been approved, reworded or skipped")
		fs.StringVar(&opts.OutDir, "out", "anki", "directory to write the deck file into")
		fs.StringVar(&opts.DeckName, "deck", "Riftbound::Rulings", "name of the deck to import into")
		fs.BoolVar(&opts.Revisit, "revisit", false, "offer the rulings already decided on again")
		return func() error { return genRulesAnki(opts) }
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
	fmt.Fprint(w, "usage: rb <command> [flags] [args]\n")
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

func downloadCards(outDir, missingFile string, images bool, concurrency int) error {
	if err := riftcodex.DownloadCards(outDir, missingFile, images, concurrency); err != nil {
		return fmt.Errorf("failed to download cards: %w", err)
	}
	return nil
}

func collect(collectionFile, catalogFile string, filters []string) error {
	if err := collection.RunEditor(collectionFile, catalogFile, filters); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}
	return nil
}

func validate(collectionFile, setID string) error {
	if err := collection.RunValidator(collectionFile, setID); err != nil {
		return fmt.Errorf("failed to validate the collection: %w", err)
	}
	return nil
}

func collectionStats(collectionFile, catalogFile, dataFile string) error {
	if err := collection.RunStats(collectionFile, catalogFile, dataFile); err != nil {
		return fmt.Errorf("failed to show the collection stats: %w", err)
	}
	return nil
}

func missing(collectionFile, catalogFile string, filters []string, axis collection.Axis) error {
	if err := collection.Missing(collectionFile, catalogFile, filters, axis, os.Stdout); err != nil {
		return fmt.Errorf("failed to list the missing cards: %w", err)
	}
	return nil
}

func addDecks(decksFile, catalogFile, setsFile string) error {
	if err := decks.RunAdder(decksFile, catalogFile, setsFile); err != nil {
		return fmt.Errorf("failed to register the decks: %w", err)
	}
	return nil
}

func matchDecks(opts decks.Options) error {
	if err := decks.Match(opts, os.Stdout); err != nil {
		return fmt.Errorf("failed to match the decks against the collection: %w", err)
	}
	if opts.ReportPath != "" {
		fmt.Printf("\nkept a copy in %s\n", opts.ReportPath)
	}
	return nil
}

func genAnki(opts ankigen.Options, only string) error {
	if opts.EffectMaskFraction <= 0 || opts.EffectMaskFraction > 1 {
		return fmt.Errorf("-effect-mask must be between 0 and 1, got %v", opts.EffectMaskFraction)
	}

	blocks := ankigen.Blocks
	if only != "" {
		blocks = nil
		for _, name := range strings.Split(only, ",") {
			b, err := ankigen.BlockByName(strings.TrimSpace(name))
			if err != nil {
				return err
			}
			blocks = append(blocks, b)
		}
	}

	var files []string
	notes := 0
	for _, b := range blocks {
		res, err := b.Run(opts)
		if err != nil {
			return fmt.Errorf("failed to generate the %s block: %w", b.Name, err)
		}
		files = append(files, res.Files...)
		notes += res.Notes
	}

	fmt.Printf("wrote %d notes across %d deck files:\n", notes, len(files))
	for _, f := range files {
		fmt.Printf("  %s\n", f)
	}
	fmt.Printf(`
to import:
  1. run: just add-anki-media   (copies %s/media/* into Anki's collection.media)
  2. in Anki, File > Import, and choose whichever deck files above you want —
     each imports independently, so pick and choose
`, opts.OutDir)
	return nil
}

func addAnkiMedia(mediaDir, ankiDir, profile string) error {
	if err := ankigen.AddMedia(mediaDir, ankiDir, profile, os.Stdout); err != nil {
		return fmt.Errorf("failed to add the media to Anki: %w", err)
	}
	return nil
}

func genRulesAnki(opts ankigen.RulesOptions) error {
	res, err := ankigen.ReviewRulings(opts, os.Stdin, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to review the rulings: %w", err)
	}

	fmt.Printf(`
%d %s this pass · %d approved · %d skipped · %d left

wrote %s

to import:
  in Anki, File > Import and choose %s
`, res.Decided, plural(res.Decided, "ruling"), res.Notes, res.Skipped, res.Left, res.DeckFile, res.DeckFile)
	return nil
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func IsUsage(err error) bool {
	var u usageError
	return errors.As(err, &u)
}
