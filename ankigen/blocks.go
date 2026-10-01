package ankigen

import (
	"fmt"
	"os"
)

// A Block is one independently importable slice of the deck output: it
// writes its own import file (and, for image decks, its own media) under
// opts.OutDir and reports what it wrote. Splitting generation into blocks
// lets a run regenerate one deck — say, after a new set's rulings land —
// without redoing the image rendering the others depend on, and lets the
// files be imported into Anki one at a time rather than all at once.
type Block struct {
	Name string
	Run  func(Options) (BlockResult, error)
}

// BlockResult reports the files one block wrote and how many notes went into
// them.
type BlockResult struct {
	Files []string
	Notes int
}

// Blocks lists every block gen-anki can produce, in the order they run.
var Blocks = []Block{
	{Name: "hidden", Run: runHidden},
	{Name: "hidden-domains", Run: runHiddenDomains},
	{Name: "reaction-cards", Run: runReactionCards},
	{Name: "action-cards", Run: runActionCards},
	{Name: "signature-cards", Run: runSignatureCards},
	// Rulings run last, since reviewing them asks questions at the terminal
	// and the blocks before it don't need anyone there.
	{Name: "rulings", Run: runRulings},
}

// BlockByName finds a block by name, for a caller that wants to run less than
// everything in Blocks.
func BlockByName(name string) (Block, error) {
	for _, b := range Blocks {
		if b.Name == name {
			return b, nil
		}
	}
	return Block{}, fmt.Errorf("unknown block %q, want one of %s", name, BlockNames())
}

// BlockNames lists the available block names, comma-separated, for help text
// and error messages.
func BlockNames() string {
	var s string
	for i, b := range Blocks {
		if i > 0 {
			s += ", "
		}
		s += b.Name
	}
	return s
}

func runHidden(opts Options) (BlockResult, error) {
	res, err := GenerateHiddenEffects(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.EffectDeckFile}, Notes: res.Notes}, nil
}

func runHiddenDomains(opts Options) (BlockResult, error) {
	res, err := GenerateHiddenDomains(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.DeckFile}, Notes: res.Notes}, nil
}

func runReactionCards(opts Options) (BlockResult, error) {
	res, err := GenerateReactionCards(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.DeckFile}, Notes: res.Notes}, nil
}

func runActionCards(opts Options) (BlockResult, error) {
	res, err := GenerateActionCards(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.DeckFile}, Notes: res.Notes}, nil
}

func runSignatureCards(opts Options) (BlockResult, error) {
	res, err := GenerateSignatureCards(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.DeckFile}, Notes: res.Notes}, nil
}

// runRulings reviews the rulings not yet decided on before writing their deck.
// With none left, or no terminal to answer at, it writes the deck straight
// away, so a run of every block only stops to ask when there is something to
// ask about.
func runRulings(opts Options) (BlockResult, error) {
	ro := opts.Rules
	ro.OutDir = opts.OutDir
	res, err := ReviewRulings(ro, os.Stdin, os.Stdout)
	if err != nil {
		return BlockResult{}, err
	}
	fmt.Printf("\n%d %s reviewed this pass · %d approved · %d skipped · %d left\n\n",
		res.Decided, plural(res.Decided, "ruling"), res.Notes, res.Skipped, res.Left)
	return BlockResult{Files: []string{res.DeckFile}, Notes: res.Notes}, nil
}
