package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"rb/ankigen"
	"rb/collection"
	"rb/decks"
	"rb/riftcodex"
	"rb/rules"
)

var commands = []string{"download-cards", "collect", "validate", "collection-stats", "watch-decks", "gen-anki", "gen-rules-cards", "add-anki-media", "missing"}

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

	case "watch-decks":
		opts := decks.WatchOptions{Notify: true}
		fs.StringVar(&opts.CatalogPath, "catalog-file", "cards/cards.json", "card catalog to check the copied card names against")
		fs.StringVar(&opts.SetsPath, "sets-file", "cards/sets.json", "set list to date each deck by the newest set in print")
		fs.StringVar(&opts.DecksPath, "decks-file", "decks/decks.json", "deck register to append to")
		fs.DurationVar(&opts.Interval, "interval", 500*time.Millisecond, "how often to read the clipboard")
		fs.BoolVar(&opts.Notify, "notify", true, "raise a notification for each deck saved")
		return func() error { return watchDecks(opts) }

	case "gen-anki":
		var opts ankigen.Options
		fs.StringVar(&opts.CatalogPath, "catalog-file", "cards/cards.json", "card catalog to build the deck from")
		fs.StringVar(&opts.ImageDir, "image-dir", "cards/images", "directory holding the downloaded card scans")
		fs.StringVar(&opts.OutDir, "out", "anki", "directory to write the deck files and their media into")
		fs.StringVar(&opts.EffectDeckName, "effect-deck", "Riftbound::Hidden Effects", "name of the companion deck asking what a card does")
		fs.StringVar(&opts.ReactionDeckName, "reaction-deck", "Riftbound::Reaction Cards", "name of the deck listing each domain's Reaction cards")
		fs.StringVar(&opts.ActionDeckName, "action-deck", "Riftbound::Action Cards", "name of the deck listing each domain's Action cards")
		fs.StringVar(&opts.SignatureDeckName, "signature-deck", "Riftbound::Signature Cards", "name of the deck asking which card each legend brings")
		fs.StringVar(&opts.HiddenDomainDeckName, "hidden-domain-deck", "Riftbound::Hidden by Domain", "name of the deck asking which Hidden cards each domain holds")
		fs.Float64Var(&opts.EffectMaskFraction, "effect-mask", 0.4, "fraction of the card height to paint out from the bottom, for cards whose keyword badge can't be found")
		fs.IntVar(&opts.ImageWidth, "image-width", 500, "width to scale card images to, or 0 to keep them full size")
		fs.BoolVar(&opts.AllPrintings, "all-printings", false, "make a note per printing rather than per card")
		fs.StringVar(&opts.Rules.RulingsPath, "rulings-file", "rules/rulings.json", "ruling dataset to draft the rulings notes from")
		fs.StringVar(&opts.Rules.ReviewPath, "review-file", "rules/anki-review.json", "record of which rulings have been approved, reworded or skipped")
		fs.StringVar(&opts.Rules.DeckName, "rulings-deck", "Riftbound::Rulings", "name of the deck the rulings import into")
		fs.BoolVar(&opts.Rules.Revisit, "revisit", false, "offer the rulings already decided on again")
		only := fs.String("only", "", "comma-separated blocks to generate, or empty for all (available: "+ankigen.BlockNames()+")")
		return func() error { return genAnki(opts, *only) }

	case "add-anki-media":
		defaultDir, _ := ankigen.DefaultAnkiDir()
		mediaDir := fs.String("media-dir", "anki/media", "directory holding the generated card images")
		ankiDir := fs.String("anki-dir", defaultDir, "Anki's data folder, holding one folder per profile")
		profile := fs.String("profile", "", "Anki profile to copy into, or empty to use the only one")
		return func() error { return addAnkiMedia(*mediaDir, *ankiDir, *profile) }

	case "gen-rules-cards":
		rulingsFile := fs.String("rulings-file", "rules/rulings.json", "ruling dataset to read the questions from")
		catalogFile := fs.String("catalog-file", "cards/cards.json", "card catalog to find the named cards in")
		outFile := fs.String("out", "rules/ruling-cards.json", "file to write the cards each question names into")
		return func() error { return genRulesCards(*rulingsFile, *catalogFile, *outFile) }

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

func watchDecks(opts decks.WatchOptions) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := decks.RunWatcher(ctx, opts); err != nil {
		return fmt.Errorf("failed to watch the clipboard for decks: %w", err)
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

func genRulesCards(rulingsFile, catalogFile, outFile string) error {
	if err := rules.LinkCards(rulingsFile, catalogFile, outFile); err != nil {
		return fmt.Errorf("failed to link the cards each ruling names: %w", err)
	}
	fmt.Printf("wrote %s\n", outFile)
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
