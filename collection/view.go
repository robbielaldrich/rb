package collection

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

// frame lays out the whole screen as plain lines plus the cursor position,
// leaving the escape-code bookkeeping to screen.render.
func (e *editor) frame(width int) (lines []string, caretRow, caretCol int) {
	if e.mode == modeQuantity {
		return e.quantityFrame(width)
	}
	return e.searchFrame(width)
}

func (e *editor) searchFrame(width int) (lines []string, caretRow, caretCol int) {
	lines = append(lines, "  "+dim(truncate(e.status, width-2)))

	// A query longer than the terminal scrolls, so the caret stays visible
	// rather than wrapping the prompt onto a second line.
	q := []rune(e.query)
	if n := width - 3; len(q) > n {
		q = q[len(q)-n:]
	}
	caretRow = len(lines)
	caretCol = 2 + len(q)
	lines = append(lines, "> "+string(q), "")

	switch {
	case len(e.results) > 0:
		for i, r := range e.results {
			lines = append(lines, resultLine(i, r, width))
		}
	case e.query != "":
		lines = append(lines, "  "+dim("no matches"))
	default:
		lines = append(lines, "  "+dim("start typing a card name"))
	}

	lines = append(lines, "", "  "+dim(truncate(
		"1-5 add · enter adds the top match · #44 collector number · esc quit", width-2)))
	return lines, caretRow, caretCol
}

func (e *editor) quantityFrame(width int) (lines []string, caretRow, caretCol int) {
	lines = append(lines, "  "+bold(truncate(e.card.Label(), width-2)))

	caretRow = len(lines)
	caretCol = 2 + utf8.RuneCountInString(e.qty)
	lines = append(lines, "× "+e.qty+dim("  copies owned"), "")

	lines = append(lines, "  "+dim(truncate(
		"digits, ↑↓ or +/- set the count · enter done · or type the next card name", width-2)))
	return lines, caretRow, caretCol
}

// resultLine renders one match as "  2  Astral Heron 44/166 VEN     have 3",
// with the owned count pushed to the right margin.
func resultLine(i int, r result, width int) string {
	owned := ""
	if r.owned > 0 {
		owned = "have " + strconv.Itoa(r.owned)
	}

	const gutter = 5 // "  1  "
	const ownedCol = 9
	label := truncate(r.card.Label(), width-gutter-ownedCol)
	if owned != "" {
		label = pad(label, width-gutter-ownedCol)
	}

	return fmt.Sprintf("  %s  %s%s", paint(strconv.Itoa(i+1), "36;1"), label, dim(owned))
}
