package ankigen

import (
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/cards"
)

// fixture writes a catalog and matching scans to a temp dir, so the tests
// don't depend on the downloaded card data.
func fixture(t *testing.T, cards []cards.Card) Options {
	t.Helper()
	dir := t.TempDir()
	imgDir := filepath.Join(dir, "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(cards)
	if err != nil {
		t.Fatal(err)
	}
	catPath := filepath.Join(dir, "cards.json")
	if err := os.WriteFile(catPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, c := range cards {
		writeScan(t, filepath.Join(imgDir, c.RiftboundID+".png"))
	}

	return Options{
		CatalogPath:  catPath,
		ImageDir:     imgDir,
		OutDir:       filepath.Join(dir, "out"),
		DeckName:     "Riftbound::Hidden Costs",
		MaskFraction: 1.0 / 3.0,
		ImageWidth:   60,
	}
}

// writeScan draws a stand-in for a card: solid white, with the transparent
// rounded corner the real scans have.
func writeScan(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 120, 168))
	for y := range 168 {
		for x := range 120 {
			img.Set(x, y, color.RGBA{0xff, 0xff, 0xff, 0xff})
		}
	}
	img.Set(0, 0, color.RGBA{0, 0, 0, 0})

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func hiddenCard(id, name string) cards.Card {
	return cards.Card{
		Name: name, RiftboundID: id, CollectorNumber: 3,
		Set:            cards.CardSet{SetID: "unl"},
		Classification: cards.Classification{Type: "Unit"},
		Text:           cards.Text{Plain: "[Hidden] (Hide now.)Deal 2 to an enemy unit."},
	}
}

func readDeck(t *testing.T, res Result) (headers []string, rows [][]string) {
	t.Helper()
	data, err := os.ReadFile(res.DeckFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.HasPrefix(line, "#") {
			headers = append(headers, line)
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	return headers, rows
}

func TestGenerateHiddenCosts(t *testing.T) {
	opts := fixture(t, []cards.Card{
		hiddenCard("unl-003-219", "Mischievous Marai"),
		{
			Name: "Arena Kingpin", RiftboundID: "unl-001-219", CollectorNumber: 1,
			Set:  cards.CardSet{SetID: "unl"},
			Text: cards.Text{Plain: "I enter ready."},
		},
	})

	res, err := GenerateHiddenCosts(opts)
	if err != nil {
		t.Fatalf("GenerateHiddenCosts: %v", err)
	}
	if res.Notes != 1 {
		t.Fatalf("Notes = %d, want 1 (only the Hidden card)", res.Notes)
	}

	headers, rows := readDeck(t, res)
	wantHeaders := []string{
		"#separator:Tab", "#html:true", "#notetype:Basic",
		"#deck:Riftbound::Hidden Costs", "#tags column:3",
	}
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d note rows, want 1: %v", len(rows), rows)
	}

	front, back, tags := rows[0][0], rows[0][1], rows[0][2]
	if len(rows[0]) != 3 {
		t.Fatalf("row has %d fields, want 3: %v", len(rows[0]), rows[0])
	}
	if !strings.Contains(front, "What is the cost?") {
		t.Errorf("front lacks the prompt: %q", front)
	}
	if !strings.Contains(front, "rb-unl-003-219-cost-masked.jpg") {
		t.Errorf("front does not show the masked image: %q", front)
	}
	if !strings.Contains(back, "rb-unl-003-219.jpg") || strings.Contains(back, "masked") {
		t.Errorf("back should show the intact card, got %q", back)
	}
	if !strings.Contains(tags, "riftbound::hidden") || !strings.Contains(tags, "riftbound::set::UNL") {
		t.Errorf("tags = %q", tags)
	}
	// Double quotes in the fields would need CSV escaping; single-quoted HTML
	// attributes keep them out entirely.
	if strings.Contains(front+back, `"`) {
		t.Errorf("fields contain a double quote and would need escaping: %q", front+back)
	}

	// Only the Hidden card's two images were written.
	entries, err := os.ReadDir(res.MediaDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("media dir holds %d files, want 2", len(entries))
	}
}

func TestMaskedImageHidesOnlyTheTop(t *testing.T) {
	opts := fixture(t, []cards.Card{hiddenCard("unl-003-219", "Mischievous Marai")})
	res, err := GenerateHiddenCosts(opts)
	if err != nil {
		t.Fatal(err)
	}

	img := readJPEG(t, filepath.Join(res.MediaDir, "rb-unl-003-219-cost-masked.jpg"))
	b := img.Bounds()
	if b.Dx() != 60 {
		t.Errorf("width = %d, want the image scaled to 60", b.Dx())
	}

	mid := b.Dx() / 2
	if !dark(img, mid, b.Dy()/6) {
		t.Error("the top band was not painted out")
	}
	if !dark(img, mid, b.Dy()/3-2) {
		t.Error("the band stops short of a third of the height")
	}
	if dark(img, mid, b.Dy()/3+4) {
		t.Error("the band reaches past a third of the height")
	}
	if dark(img, mid, b.Dy()-4) {
		t.Error("the bottom of the card was painted out")
	}

	// The answer image keeps the whole card.
	full := readJPEG(t, filepath.Join(res.MediaDir, "rb-unl-003-219.jpg"))
	if dark(full, mid, full.Bounds().Dy()/6) {
		t.Error("the answer image is masked, it should show the intact card")
	}
}

func TestAllPrintingsFlag(t *testing.T) {
	cards := []cards.Card{
		hiddenCard("unl-028-219", "Pyke - Dockside Butcher"),
		hiddenCard("unl-028a-219", "Pyke - Dockside Butcher (Alternate Art)"),
	}

	opts := fixture(t, cards)
	res, err := GenerateHiddenCosts(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 1 {
		t.Errorf("Notes = %d, want the reprints collapsed into 1", res.Notes)
	}

	opts = fixture(t, cards)
	opts.AllPrintings = true
	res, err = GenerateHiddenCosts(opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Notes != 2 {
		t.Errorf("Notes = %d with -all-printings, want 2", res.Notes)
	}
}

func TestNoHiddenCardsIsAnError(t *testing.T) {
	opts := fixture(t, []cards.Card{{
		Name: "Arena Kingpin", RiftboundID: "unl-001-219",
		Set:  cards.CardSet{SetID: "unl"},
		Text: cards.Text{Plain: "I enter ready."},
	}})
	if _, err := GenerateHiddenCosts(opts); err == nil {
		t.Fatal("want an error when the catalog holds no Hidden cards")
	}
}

func TestMediaNameAvoidsIllegalCharacters(t *testing.T) {
	c := cards.Card{RiftboundID: "sfd-230*-221"}
	if got := mediaName(c, "-cost-masked"); strings.ContainsAny(got, `*/\`) {
		t.Errorf("mediaName = %q, want no characters illegal in a filename", got)
	}
}

func readJPEG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode %s: %v", path, err)
	}
	return img
}

// dark reports whether a pixel is painted out, allowing for JPEG ringing.
func dark(img image.Image, x, y int) bool {
	r, g, b, _ := img.At(x, y).RGBA()
	return r>>8 < 40 && g>>8 < 40 && b>>8 < 40
}
