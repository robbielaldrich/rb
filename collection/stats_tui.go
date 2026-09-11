package collection

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"rb/cards"
)

// RunStats loads the catalog and collection, leaves the summary data file
// for the collection page where asked, and shows the results in a small
// terminal viewer: one row per set plus a totals row, with a card selected to
// drill into what it is missing along either axis — set completion (any copy
// at all) or playset completion (the three a deck can run).
func RunStats(collectionPath, catalogPath, dataPath string) error {
	coll, err := load(collectionPath)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	cs, err := cards.Load(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	sets := summarise(cs, coll)
	if dataPath != "" {
		if err := writeStatsData(dataPath, sets); err != nil {
			return fmt.Errorf("failed to write the summary data: %w", err)
		}
	}

	var buf bytes.Buffer
	writeStats(&buf, sets)
	table := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	v := &statsViewer{coll: coll, cards: cs, sets: sets, table: table, axis: AxisPlayset}
	if err := v.run(os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("failed to show the collection stats: %w", err)
	}
	return nil
}

type statsMode int

const (
	statsList statsMode = iota
	statsMissing
)

// statsViewer browses the set-by-set table and, on request, the missing-card
// report for whichever row is selected.
type statsViewer struct {
	coll  *collection
	cards []cards.Card
	sets  []setStats

	// table is the rendered stats grid — header, one line per set, then the
	// totals row — built once since the numbers don't change while it's open.
	table []string

	mode statsMode
	// cursor selects a row: an index into sets, or len(sets) for the totals
	// row, which stands for the whole catalog.
	cursor int
	axis   Axis

	// missing is the rendered report for the row and axis last opened, and
	// title what it is scoped to, for the header above it.
	missing []string
	title   string
}

func (v *statsViewer) run(in, out *os.File) error {
	return runLoop(in, out, v.frame, v.handle)
}

func (v *statsViewer) handle(k key) (quit bool, err error) {
	switch k.name {
	case "ctrl+c", "ctrl+d":
		return true, nil
	case "esc", "backspace":
		if v.mode == statsMissing {
			v.mode = statsList
			return false, nil
		}
		return true, nil
	case "up":
		if v.mode == statsList {
			v.cursor = max(v.cursor-1, 0)
		}
		return false, nil
	case "down":
		if v.mode == statsList {
			v.cursor = min(v.cursor+1, len(v.sets))
		}
		return false, nil
	case "enter":
		v.openMissing(v.axis)
		return false, nil
	case "":
		switch k.r {
		case 'q':
			return true, nil
		case 's':
			v.openMissing(AxisSet)
		case 'p':
			v.openMissing(AxisPlayset)
		}
	}
	return false, nil
}

// openMissing builds the missing-card report for the selected row along the
// given axis and switches to showing it.
func (v *statsViewer) openMissing(axis Axis) {
	v.axis = axis
	cs, title := v.cards, "everything"
	if v.cursor < len(v.sets) {
		s := v.sets[v.cursor]
		cs = keep(cs, func(c cards.Card) bool { return strings.EqualFold(c.Set.SetID, s.setID) })
		title = strings.TrimSpace(s.setID + " " + s.label)
	}

	var buf bytes.Buffer
	writeMissing(&buf, wants(cs, v.coll, axis), title, axis)
	v.missing = strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	v.title, v.mode = title, statsMissing
}

func (v *statsViewer) frame(width int) (lines []string, caretRow, caretCol int) {
	if v.mode == statsMissing {
		return v.missingFrame(width)
	}
	return v.listFrame(width)
}

func (v *statsViewer) listFrame(width int) (lines []string, caretRow, caretCol int) {
	lines = append(lines, "  "+bold("collection stats"), "")
	for i, row := range v.table {
		if i == v.cursor+1 {
			row = reverse(row)
		}
		lines = append(lines, row)
	}
	lines = append(lines, "", "  "+dim(truncate(
		"↑↓ select a set · p playset missing · s set missing · esc/q quit", width-2)))

	caretRow = len(lines) - 1
	return lines, caretRow, 0
}

func (v *statsViewer) missingFrame(width int) (lines []string, caretRow, caretCol int) {
	lines = append(lines, "  "+bold(truncate(fmt.Sprintf("%s — %s", v.title, v.axis), width-2)), "")
	lines = append(lines, v.missing...)
	lines = append(lines, "", "  "+dim(truncate("esc back · q quit", width-2)))

	caretRow = len(lines) - 1
	return lines, caretRow, 0
}
