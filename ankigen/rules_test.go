package ankigen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rb/rules"
)

func testRulings() []rules.Ruling {
	return []rules.Ruling{
		{Category: "cards", Topic: "Alpha Strike", Slug: "alpha-strike", Anchor: "targets",
			Question: "How are targets chosen when playing Alpha Strike?",
			Answer:   "Its controller chooses them while finalizing the spell.",
			Rules:    []string{"402.2"}},
		{Category: "mechanics", Topic: "Ambush", Slug: "ambush", Anchor: "timing",
			Question: "Ambush: when can the unit be played?",
			Answer:   "Only where you control the battlefield."},
		{Category: "general-rules", Topic: "Targeting", Slug: "targeting", Anchor: "illegal",
			Question: "What happens if a target becomes illegal?",
			Answer:   "The spell still resolves, but the illegal target is unaffected."},
	}
}

// setup writes a rulings dataset to a temporary directory and returns options
// pointing at it.
func setup(t *testing.T, rs []rules.Ruling) RulesOptions {
	t.Helper()
	dir := t.TempDir()
	data, err := json.Marshal(rs)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "rulings.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return RulesOptions{
		RulingsPath: path,
		ReviewPath:  filepath.Join(dir, "review.json"),
		OutDir:      dir,
		DeckName:    "Riftbound::Rulings",
	}
}

func pass(t *testing.T, opts RulesOptions, answers string) RulesResult {
	t.Helper()
	res, err := ReviewRulings(opts, strings.NewReader(answers), new(strings.Builder))
	if err != nil {
		t.Fatalf("ReviewRulings = %v", err)
	}
	return res
}

func deckLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

func TestReviewKeepsRewordsAndSkips(t *testing.T) {
	opts := setup(t, testRulings())

	res := pass(t, opts, "\ne\nWhen does Ambush let me play a unit?\nOnly where I control the battlefield.\n\ns\n")
	if res.Decided != 3 || res.Notes != 2 || res.Skipped != 1 || res.Left != 0 {
		t.Errorf("decided %d, notes %d, skipped %d, left %d; want 3, 2, 1, 0",
			res.Decided, res.Notes, res.Skipped, res.Left)
	}

	lines := deckLines(t, res.DeckFile)
	if len(lines) != 2 {
		t.Fatalf("deck holds %d notes:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	// The citation is added when the deck is written, not carried in the
	// wording the review settled on.
	want := "How are targets chosen when playing Alpha Strike?\tIts controller chooses them while finalizing the spell.<br><br><small>Core Rules 402.2</small>\triftbound::rules riftbound::rules::cards riftbound::topic::alpha-strike"
	if lines[0] != want {
		t.Errorf("first note:\n%s\nwant:\n%s", lines[0], want)
	}
	if front, back, _ := strings.Cut(lines[1], "\t"); front != "When does Ambush let me play a unit?" || !strings.HasPrefix(back, "Only where I control the battlefield.") {
		t.Errorf("the reworded note reads %q", lines[1])
	}
}

func TestReviewResumesWhereItStopped(t *testing.T) {
	opts := setup(t, testRulings())

	if res := pass(t, opts, "\nq\n"); res.Decided != 1 || res.Left != 2 {
		t.Fatalf("first pass decided %d with %d left, want 1 and 2", res.Decided, res.Left)
	}

	// The rulings already settled are not offered again, so answering twice
	// gets through the two that are left.
	res := pass(t, opts, "\n\n")
	if res.Decided != 2 || res.Notes != 3 || res.Left != 0 {
		t.Errorf("second pass decided %d, notes %d, left %d; want 2, 3, 0", res.Decided, res.Notes, res.Left)
	}
	if lines := deckLines(t, res.DeckFile); len(lines) != 3 {
		t.Errorf("deck holds %d notes, want 3", len(lines))
	}
}

func TestReviewRevisitsOnRequest(t *testing.T) {
	opts := setup(t, testRulings())
	pass(t, opts, "s\ns\ns\n")

	opts.Revisit = true
	res := pass(t, opts, "\nq\n")
	if res.Notes != 1 || res.Skipped != 2 {
		t.Errorf("notes %d, skipped %d; want 1 and 2", res.Notes, res.Skipped)
	}
}

// Running out of input is how ctrl+d ends a pass: what was decided up to then
// is kept, and nothing is decided by accident.
func TestReviewStopsAtEndOfInput(t *testing.T) {
	opts := setup(t, testRulings())

	res := pass(t, opts, "\n")
	if res.Decided != 1 || res.Left != 2 {
		t.Errorf("decided %d with %d left, want 1 and 2", res.Decided, res.Left)
	}
	if _, err := os.Stat(opts.ReviewPath); err != nil {
		t.Errorf("the decision was not written through: %v", err)
	}
}

func TestReviewKeepsDecisionsAboutRulingsThatHaveGone(t *testing.T) {
	all := testRulings()
	opts := setup(t, all)
	pass(t, opts, "\n\n\n")

	shortened := setup(t, all[:1])
	shortened.ReviewPath = opts.ReviewPath
	res := pass(t, shortened, "")

	rev, err := loadReview(opts.ReviewPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rev.Notes) != 3 {
		t.Errorf("the review file holds %d notes, want the 3 decided", len(rev.Notes))
	}
	// Only the ruling still in the dataset makes it into the deck.
	if res.Notes != 1 {
		t.Errorf("deck holds %d notes, want 1", res.Notes)
	}
}
