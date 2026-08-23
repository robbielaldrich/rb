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
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)

	h := int(math.Round(float64(src.Bounds().Dy()) * frac))
	band := image.Rect(0, 0, src.Bounds().Dx(), min(h, src.Bounds().Dy()))
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
