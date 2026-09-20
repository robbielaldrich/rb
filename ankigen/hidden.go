package ankigen

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"rb/cards"
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
	// EffectDeckName is the companion deck that asks what a Hidden card does
	// rather than what it costs.
	EffectDeckName string
	// EffectMaskFraction is how much of the card's height to paint out for
	// that deck, measured from the bottom edge. It is only a fallback: the
	// mask normally starts just under the card's first keyword badge, wherever
	// the printing happens to put it.
	EffectMaskFraction float64
	// ImageWidth caps the pixel width of the generated images; 0 keeps the
	// scans at their original size.
	ImageWidth int
	// AllPrintings keeps every printing of a card. By default one note is
	// made per card, since alternate arts and reprints of the same card would
	// otherwise ask the same question several times.
	AllPrintings bool
	// ReactionDeckName is the deck the reaction-spells block imports into.
	ReactionDeckName string
	// HiddenDomainDeckName is the deck asking which Hidden cards each domain
	// holds, rather than what any one of them does.
	HiddenDomainDeckName string
}

// Result reports what a run produced, so the caller can tell the user where
// things landed.
type Result struct {
	CostDeckFile   string
	EffectDeckFile string
	MediaDir       string
	Notes          int
}

// mediaPrefix namespaces the generated files inside Anki's collection.media,
// which is one flat folder shared by every deck.
const mediaPrefix = "rb-"

// GenerateHiddenCosts builds two decks for memorising the cards with the
// Hidden keyword, since a Hidden card has to be recognised from either half:
// one asks what a card costs, from a scan with its top band painted out; the
// other asks what it does, from a scan with its rules text painted out. Both
// answer with the card intact, and both draw on the same media folder.
func GenerateHiddenCosts(opts Options) (Result, error) {
	cs, err := cards.Load(opts.CatalogPath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load catalog: %w", err)
	}

	hidden := selectHidden(cs, opts.AllPrintings)
	if len(hidden) == 0 {
		return Result{}, fmt.Errorf("no cards with the Hidden keyword in %s", opts.CatalogPath)
	}

	mediaDir := filepath.Join(opts.OutDir, "media")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("failed to create %s: %w", mediaDir, err)
	}

	costs := deck{name: opts.DeckName, notetype: "Basic"}
	effects := deck{name: opts.EffectDeckName, notetype: "Basic"}
	for _, c := range hidden {
		r, err := renderCard(c, mediaDir, opts)
		if err != nil {
			return Result{}, err
		}
		costs.notes = append(costs.notes, note{
			front: img(r.costMasked) + "<br>What is the cost?",
			back:  img(r.full),
			tags:  tagsFor(c, "riftbound::hidden::cost"),
		})
		effects.notes = append(effects.notes, note{
			front: img(r.textMasked) + "<br>What does this hidden card do?",
			back:  img(r.full),
			tags:  tagsFor(c, "riftbound::hidden::effect"),
		})
	}

	costFile := filepath.Join(opts.OutDir, "riftbound-hidden-costs.txt")
	if err := costs.write(costFile); err != nil {
		return Result{}, fmt.Errorf("failed to write deck: %w", err)
	}
	effectFile := filepath.Join(opts.OutDir, "riftbound-hidden-effects.txt")
	if err := effects.write(effectFile); err != nil {
		return Result{}, fmt.Errorf("failed to write deck: %w", err)
	}

	return Result{
		CostDeckFile:   costFile,
		EffectDeckFile: effectFile,
		MediaDir:       mediaDir,
		Notes:          len(costs.notes) + len(effects.notes),
	}, nil
}

// selectHidden picks the cards that actually carry the Hidden keyword, in
// printed order. Unless every printing is wanted, one representative printing
// stands in for each card: the same cost asked five times over teaches
// nothing extra.
func selectHidden(cs []cards.Card, allPrintings bool) []cards.Card {
	var out []cards.Card
	at := map[string]int{}
	for _, c := range cs {
		if !c.HasKeyword("Hidden") {
			continue
		}
		if allPrintings {
			out = append(out, c)
			continue
		}

		i, ok := at[c.BaseName()]
		if !ok {
			at[c.BaseName()] = len(out)
			out = append(out, c)
			continue
		}
		// Where a card was also handed out as a promo, the printing kept is
		// the one from the set it belongs to, since that is the set a reader
		// would go looking for it in.
		if out[i].IsPromo() && !c.IsPromo() {
			out[i] = c
		}
	}
	// Studying in name order beats studying in set order: the deck reads as a
	// list of cards rather than a walk through a release.
	slices.SortFunc(out, func(a, b cards.Card) int {
		return strings.Compare(a.Label(), b.Label())
	})
	return out
}

// rendered names the images written for one card, so the notes can reference
// them.
type rendered struct {
	full       string
	costMasked string
	textMasked string
}

// cardScan reads a card's scan at the size the decks show it at.
func cardScan(c cards.Card, opts Options) (*image.RGBA, error) {
	src := filepath.Join(opts.ImageDir, c.RiftboundID+".png")
	img, err := loadCardImage(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read the scan of %s: %w", c.Label(), err)
	}
	return scaleToWidth(img, opts.ImageWidth), nil
}

// renderFull writes just the intact scan of a card and reports what it was
// filed under, for the decks that show a card without asking anything of it.
// The name is the one renderCard uses, so the two share the one file in
// collection.media rather than each keeping a copy.
func renderFull(c cards.Card, mediaDir string, opts Options) (string, error) {
	img, err := cardScan(c, opts)
	if err != nil {
		return "", err
	}

	name := mediaName(c, "")
	if err := writeJPEG(filepath.Join(mediaDir, name), img); err != nil {
		return "", fmt.Errorf("failed to write the image of %s: %w", c.Label(), err)
	}
	return name, nil
}

// renderCard writes the intact and masked images for one card.
func renderCard(c cards.Card, mediaDir string, opts Options) (rendered, error) {
	img, err := cardScan(c, opts)
	if err != nil {
		return rendered{}, err
	}

	r := rendered{
		full:       mediaName(c, ""),
		costMasked: mediaName(c, "-cost-masked"),
		textMasked: mediaName(c, "-text-masked"),
	}
	if err := writeJPEG(filepath.Join(mediaDir, r.full), img); err != nil {
		return rendered{}, fmt.Errorf("failed to write the image of %s: %w", c.Label(), err)
	}
	if err := writeJPEG(filepath.Join(mediaDir, r.costMasked), maskTop(img, opts.MaskFraction)); err != nil {
		return rendered{}, fmt.Errorf("failed to write the cost-masked image of %s: %w", c.Label(), err)
	}
	if err := writeJPEG(filepath.Join(mediaDir, r.textMasked), maskBelowKeywords(img, opts.EffectMaskFraction)); err != nil {
		return rendered{}, fmt.Errorf("failed to write the text-masked image of %s: %w", c.Label(), err)
	}
	return r, nil
}

// mediaName builds a filename safe for collection.media. The '*' that marks a
// signature printing's riftbound_id is not legal on every filesystem.
func mediaName(c cards.Card, suffix string) string {
	return mediaPrefix + strings.ReplaceAll(c.RiftboundID, "*", "s") + suffix + ".jpg"
}

func tagsFor(c cards.Card, kind string) []string {
	return []string{
		"riftbound::hidden",
		kind,
		"riftbound::set::" + strings.ToUpper(c.Set.SetID),
		"riftbound::type::" + strings.ToLower(c.Classification.Type),
	}
}
