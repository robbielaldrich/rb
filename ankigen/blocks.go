package ankigen

import "fmt"

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
	{Name: "reaction-spells", Run: runReactionSpells},
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
	res, err := GenerateHiddenCosts(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.CostDeckFile, res.EffectDeckFile}, Notes: res.Notes}, nil
}

func runReactionSpells(opts Options) (BlockResult, error) {
	res, err := GenerateReactionSpells(opts)
	if err != nil {
		return BlockResult{}, err
	}
	return BlockResult{Files: []string{res.DeckFile}, Notes: res.Notes}, nil
}
