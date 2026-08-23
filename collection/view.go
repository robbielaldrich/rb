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
	prompt := "> "
	if e.setID != "" {
		prompt = e.setID + " > "
	}
	q := []rune(e.query)
	if n := width - 1 - len(prompt); len(q) > n {
		q = q[len(q)-n:]
	}
	caretRow = len(lines)
	caretCol = len(prompt) + len(q)
	lines = append(lines, prompt+string(q), "")

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
		"1-5 add · enter top match · #44 number · ctrl+z undo · esc quit", width-2)))
	return lines, caretRow, caretCol
}

func (e *editor) quantityFrame(width int) (lines []string, caretRow, caretCol int) {
	lines = append(lines, "  "+bold(truncate(e.card.Label(), width-2)))

	caretRow = len(lines)
	caretCol = 2 + utf8.RuneCountInString(e.qty)
	lines = append(lines, "× "+e.qty+dim("  copies owned"), "")

	lines = append(lines, "  "+dim(truncate(
		"digits, ↑↓ or +/- count · enter done · ctrl+z undo · or type a name", width-2)))
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

func (v *validator) frame(width int) (lines []string, caretRow, caretCol int) {
	e := v.entry()
	where := ""
	if v.setID != "" {
		where = " · " + v.setID
	}
	lines = append(lines, "  "+dim(truncate(fmt.Sprintf("card %d of %d%s%s",
		v.at+1, len(v.ids), where, v.status), width-2)))
	lines = append(lines, "  "+bold(truncate(
		fmt.Sprintf("%s %s %s", e.Name, e.Number, e.SetID), width-2)))

	caretRow = len(lines)
	caretCol = 2 + utf8.RuneCountInString(v.qty)
	lines = append(lines, "× "+v.qty+dim("  copies recorded"), "")

	lines = append(lines, "  "+dim(truncate(
		"enter ok · digits, ↑↓ or +/- fix · ctrl+z back · esc stop", width-2)))
	return lines, caretRow, caretCol
}
