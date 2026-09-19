package ankigen

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
)

// jpegQuality trades a little sharpness for a deck that syncs: the source
// PNGs are ~800 KB each and every note carries two of them.
const jpegQuality = 85

// loadCardImage reads a card scan and flattens it onto black. The scans are
// RGBA with transparent rounded corners, and JPEG has no alpha channel; black
// is the card's own border colour, so the seam doesn't show.
func loadCardImage(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", path, err)
	}

	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst, nil
}

// scaleToWidth box-filters the image down to width, averaging each source
// rectangle into one destination pixel. Averaging is only correct because the
// image is already opaque, and only appropriate downscaling, so images that
// are already narrow enough are passed through untouched.
func scaleToWidth(src *image.RGBA, width int) *image.RGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if width <= 0 || width >= sw {
		return src
	}
	height := max(int(math.Round(float64(sh)*float64(width)/float64(sw))), 1)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		y0, y1 := y*sh/height, max((y+1)*sh/height, y*sh/height+1)
		for x := range width {
			x0, x1 := x*sw/width, max((x+1)*sw/width, x*sw/width+1)

			var r, g, b, n uint32
			for sy := y0; sy < y1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < x1; sx++ {
					p := row[sx*4:]
					r, g, b, n = r+uint32(p[0]), g+uint32(p[1]), b+uint32(p[2]), n+1
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o+0], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = uint8(r/n), uint8(g/n), uint8(b/n), 0xff
		}
	}
	return dst
}

// maskTop returns a copy with the top frac of the image painted out, hiding
// the cost and might printed along a card's upper edge while leaving the art,
// name and rules text below it readable.
func maskTop(src *image.RGBA, frac float64) *image.RGBA {
	h := src.Bounds().Dy()
	return maskBand(src, 0, min(bandHeight(h, frac), h))
}

// maskBottom returns a copy with the bottom frac of the image painted out,
// hiding the rules text printed across a card's lower half while leaving the
// name, cost and art above it readable.
func maskBottom(src *image.RGBA, frac float64) *image.RGBA {
	h := src.Bounds().Dy()
	return maskBand(src, max(h-bandHeight(h, frac), 0), h)
}

// maskBelowKeywords returns a copy painted out from just under the first
// keyword badge, leaving the keyword line itself readable. It is how the
// effect deck asks what a Hidden card does: the [Hidden] line says only that
// the card can be hidden, which is the premise of the question rather than
// its answer, while everything below it — the other keywords, and the rules
// text proper — is what has to be recalled.
//
// A fixed fraction can't do this. The rules box sits at a different height on
// a full-art printing than on an ordinary one, and cards carry different
// numbers of keyword lines, so any one fraction either swallows the [Hidden]
// line on some cards or leaves the effect text showing on others. frac is the
// fraction to fall back to on a card whose badge can't be found.
func maskBelowKeywords(src *image.RGBA, frac float64) *image.RGBA {
	y, ok := keywordBandBottom(src)
	if !ok {
		return maskBottom(src, frac)
	}
	return maskBand(src, y, src.Bounds().Dy())
}

// Keyword badges are a dark teal pill with the keyword set in white inside
// it. The two sets of scans differ a little in the red channel — Vendetta
// prints [0 115 97] where the earlier sets print [36 112 95] — so the test is
// loose there and tight on the green and blue that make the colour.
func isBadgeTeal(r, g, b uint8) bool {
	return r < 70 && g > 93 && g < 133 && b > 76 && b < 116
}

// keywordBandBottom finds where the first keyword badge ends, in pixels from
// the top, and reports whether one was found at all.
//
// The badge is looked for down the left edge of the rules box, where every
// keyword line starts, and only over the lower half of the card, so that the
// teal a piece of art happens to contain can't be mistaken for one. Art that
// does fall in the colour still leaves streaks a row or two tall, so a run
// only counts as a badge at something like the height one is printed at.
func keywordBandBottom(src *image.RGBA) (int, bool) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// A badge stands about 3% of the card tall; the bounds either side of that
	// separate one from both the streaks art leaves and any larger teal panel.
	minRun, maxRun := h*23/1000, h*45/1000
	// White letters interrupt the pill, so a row is counted on a handful of
	// teal pixels rather than a solid line of them.
	minPixels := max(w/80, 3)

	open := -1
	for y := h / 2; y < h; y++ {
		n := 0
		for x := w / 10; x < w*28/100; x++ {
			o := src.PixOffset(x, y)
			if isBadgeTeal(src.Pix[o], src.Pix[o+1], src.Pix[o+2]) {
				n++
			}
		}
		switch {
		case n >= minPixels && open < 0:
			open = y
		case n < minPixels && open >= 0:
			if y-open >= minRun && y-open <= maxRun {
				// Cleared by a hair so the pill's own edge doesn't survive.
				return min(y+h*7/1000, h), true
			}
			open = -1
		}
	}
	return 0, false
}

func bandHeight(height int, frac float64) int {
	return int(math.Round(float64(height) * frac))
}

func maskBand(src *image.RGBA, y0, y1 int) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)

	band := image.Rect(0, y0, src.Bounds().Dx(), y1)
	draw.Draw(dst, band, image.NewUniform(color.Black), image.Point{}, draw.Src)
	return dst
}

func writeJPEG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		f.Close()
		return fmt.Errorf("failed to encode %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", path, err)
	}
	return nil
}
