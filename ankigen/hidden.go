package ankigen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"rb/catalog"
)

// Options configure a generated deck.
type Options struct {
	CatalogPath string
	ImageDir    string
	OutDir      string
	DeckName    string
	// MaskFraction is how much of the card's height to paint out, measured
	// from the top edge.
	MaskFraction float64
	// ImageWidth caps the pixel width of the generated images; 0 keeps the
	// scans at their original size.
	ImageWidth int
	// AllPrintings keeps every printing of a card. By default one note is
	// made per card, since alternate arts and reprints of the same card would
	// otherwise ask the same question several times.
	AllPrintings bool
}

// Result reports what a run produced, so the caller can tell the user where
// things landed.
type Result struct {
	DeckFile string
	MediaDir string
	Notes    int
	Images   int
}

// mediaPrefix namespaces the generated files inside Anki's collection.media,
// which is one flat folder shared by every deck.
const mediaPrefix = "rb-"

// GenerateHiddenCosts builds a deck for memorising what the cards with the
// Hidden keyword cost. The front is the card with its top band painted out,
// hiding the printed cost; the back is the same card intact.
func GenerateHiddenCosts(opts Options) (Result, error) {
	cards, err := catalog.Load(opts.CatalogPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	hidden := selectHidden(cards, opts.AllPrintings)
	if len(hidden) == 0 {
		return Result{}, fmt.Errorf("no cards with the Hidden keyword in %s", opts.CatalogPath)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}

	d := deck{name: opts.DeckName, notetype: "Basic"}
	images := 0
	for _, c := range hidden {
		masked, full, err := renderCard(c, mediaDir, opts)
		if err != nil {
			return Result{}, err
		}
		images += 2

		d.notes = append(d.notes, note{
			front: img(masked) + "<br>What is the cost?",
			back:  img(full),
			tags:  tagsFor(c),
		})
	}

	deckFile := filepath.Join(opts.OutDir, "riftbound-hidden-costs.txt")
	if err := d.write(deckFile); err != nil {
		return Result{}, fmt.Errorf("failed to write deck: %w", err)
	}

	return Result{DeckFile: deckFile, MediaDir: mediaDir, Notes: len(d.notes), Images: images}, nil
}

// selectHidden picks the cards that actually carry the Hidden keyword, in
// printed order. Unless every printing is wanted, one representative printing
// stands in for each card: the same cost asked five times over teaches
// nothing extra.
func selectHidden(cards []catalog.Card, allPrintings bool) []catalog.Card {
	var out []catalog.Card
	seen := map[string]bool{}
	for _, c := range cards {
		if !c.HasKeyword("Hidden") {
			continue
		}
		if !allPrintings {
			if seen[c.BaseName()] {
				continue
			}
			seen[c.BaseName()] = true
		}
		out = append(out, c)
	}
	// Studying in name order beats studying in set order: the deck reads as a
	// list of cards rather than a walk through a release.
	slices.SortFunc(out, func(a, b catalog.Card) int {
		return strings.Compare(a.Label(), b.Label())
	})
	return out
}

// renderCard writes the masked and intact images for one card and returns the
// names to reference them by.
func renderCard(c catalog.Card, mediaDir string, opts Options) (masked, full string, err error) {
	src := filepath.Join(opts.ImageDir, c.RiftboundID+".png")
	img, err := loadCardImage(src)
	if err != nil {
		return "", "", fmt.Errorf("failed to read the scan of %s: %w", c.Label(), err)
	}
	img = scaleToWidth(img, opts.ImageWidth)

	full = mediaName(c, "")
	masked = mediaName(c, "-cost-masked")
	if err := writeJPEG(filepath.Join(mediaDir, full), img); err != nil {
		return "", "", fmt.Errorf("failed to write the image of %s: %w", c.Label(), err)
	}
	if err := writeJPEG(filepath.Join(mediaDir, masked), maskTop(img, opts.MaskFraction)); err != nil {
		return "", "", fmt.Errorf("failed to write the masked image of %s: %w", c.Label(), err)
	}
	return masked, full, nil
}

// mediaName builds a filename safe for collection.media. The '*' that marks a
// signature printing's riftbound_id is not legal on every filesystem.
func mediaName(c catalog.Card, suffix string) string {
	return mediaPrefix + strings.ReplaceAll(c.RiftboundID, "*", "s") + suffix + ".jpg"
}

func tagsFor(c catalog.Card) []string {
	return []string{
		"riftbound::hidden",
		"riftbound::set::" + strings.ToUpper(c.Set.SetID),
		"riftbound::type::" + strings.ToLower(c.Classification.Type),
	}
}
