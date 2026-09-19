package ankigen

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"rb/cards"
)

// badgeCard draws a card-shaped image with keyword badges at the given rows,
// so the search can be tested without a scan on disk.
func badgeCard(t *testing.T, w, h int, bands ...[2]int) *image.RGBA {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	teal := color.RGBA{36, 112, 95, 0xff}
	for _, b := range bands {
		for y := b[0]; y < b[1]; y++ {
			for x := w / 10; x < w*26/100; x++ {
				img.Set(x, y, teal)
			}
		}
	}
	return img
}

// The mask starts under the first badge, so the keyword line reads but the
// lines under it don't.
func TestKeywordBandBottomFindsTheFirstBadge(t *testing.T) {
	img := badgeCard(t, 500, 700, [2]int{480, 502}, [2]int{510, 532})

	got, ok := keywordBandBottom(img)
	if !ok {
		t.Fatal("no badge found")
	}
	if got < 502 || got > 510 {
		t.Errorf("band bottom = %d, want it between the two badges (502..510)", got)
	}
}

// Art that happens to be the badge's colour leaves streaks a row or two tall,
// which are too short to be a keyword line.
func TestKeywordBandBottomIgnoresShortStreaks(t *testing.T) {
	img := badgeCard(t, 500, 700, [2]int{400, 404}, [2]int{430, 437}, [2]int{480, 502})

	got, ok := keywordBandBottom(img)
	if !ok {
		t.Fatal("no badge found")
	}
	if got < 502 || got > 515 {
		t.Errorf("band bottom = %d, want the real badge at 480..502, not a streak above it", got)
	}
}

// A card with no badge at all falls back to the fraction rather than masking
// from row zero.
func TestMaskBelowKeywordsFallsBack(t *testing.T) {
	img := badgeCard(t, 500, 700)
	if _, ok := keywordBandBottom(img); ok {
		t.Fatal("found a badge on a blank card")
	}

	masked := maskBelowKeywords(img, 0.4)
	// The top stays as it was, and the bottom 40% is painted out.
	if r, _, _, _ := masked.At(250, 100).RGBA(); r == 0 {
		t.Error("the top of the card was painted out")
	}
	if r, _, _, _ := masked.At(250, 650).RGBA(); r != 0 {
		t.Error("the bottom 40% was not painted out")
	}
}

// The heuristic is guarded against the real scans, where the printings that
// break it actually live: a full-art card whose art carries the badge colour,
// and Vendetta's slightly different teal.
func TestKeywordBandFoundOnEveryHiddenCard(t *testing.T) {
	cs, err := cards.Load("../cards/cards.json")
	if err != nil {
		t.Skip(err)
	}

	for _, c := range selectHidden(cs, false) {
		img, err := loadCardImage(filepath.Join("../cards/images", c.RiftboundID+".png"))
		if err != nil {
			t.Skip(err)
		}
		img = scaleToWidth(img, 500)

		y, ok := keywordBandBottom(img)
		if !ok {
			t.Errorf("%s: no keyword badge found", c.Label())
			continue
		}
		// Every printing puts the rules box in the lower half, and none puts
		// the first keyword line as far down as the card's foot.
		if h := img.Bounds().Dy(); y < h/2 || y > h*85/100 {
			t.Errorf("%s: band bottom %d is outside the rules box (height %d)", c.Label(), y, h)
		}
	}
}
