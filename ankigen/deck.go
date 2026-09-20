package ankigen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The output is Anki's plain-text import format, documented at
// https://docs.ankiweb.net/importing/text-files.html: a few "#key:value"
// option lines, then one tab-separated note per line. Anki treats the first
// field as a note's identity, so re-importing a regenerated file updates the
// existing notes instead of duplicating them.

// note is one row of the import file.
type note struct {
	front string
	back  string
	tags  []string
}

type deck struct {
	name     string
	notetype string
	notes    []note
}

func (d *deck) write(path string) error {
	// Read before the file is truncated: what it used to ask is the only
	// record of notes this run has stopped generating.
	before, err := previousFronts(path)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	w := bufio.NewWriter(f)

	fmt.Fprintf(w, "#separator:Tab\n")
	fmt.Fprintf(w, "#html:true\n")
	fmt.Fprintf(w, "#notetype:%s\n", d.notetype)
	fmt.Fprintf(w, "#deck:%s\n", d.name)
	fmt.Fprintf(w, "#tags column:3\n")
	for _, n := range d.notes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", field(n.front), field(n.back), field(strings.Join(n.tags, " ")))
	}

	if err := w.Flush(); err != nil {
		f.Close()
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", path, err)
	}

	return recordStale(filepath.Join(filepath.Dir(path), staleFile), d.name, before, d.fronts())
}

// staleFile is where a run records the notes it has stopped generating. It
// sits beside the deck files because it is read at the same moment they are:
// after an import, to tidy up what the import could not.
//
// Re-importing a regenerated deck updates the notes it still asks and adds any
// that are new, but a note that has gone out of the file is not mentioned by
// it, so Anki leaves the old card in place — scheduled, and quietly wrong.
// Nothing outside Anki can delete it, so the least a run can do is say which
// ones they are, and keep saying so until they are gone.
const staleFile = "to-delete.txt"

const staleHeader = `# Notes these decks used to generate and no longer do.
#
# Re-importing can't remove them: Anki only updates the notes a file mentions,
# so these are still in the collection, still scheduled, and no longer correct.
# Search Anki for the question, delete the note, then delete the line.
#
# Written by ` + "`rb gen-anki`" + `. A question that comes back — a new set can
# make a dropped one worth asking again — is taken off this list by the run
# that starts asking it.
#
# date	deck	question
`

// stale is one note that was generated once and isn't any more.
type stale struct {
	date  string
	deck  string
	front string
}

// fronts lists the questions this deck asks. Anki files a note under its first
// field, so the question is the identity a note is kept or lost by.
func (d *deck) fronts() map[string]bool {
	out := make(map[string]bool, len(d.notes))
	for _, n := range d.notes {
		out[n.front] = true
	}
	return out
}

// previousFronts reads the questions a deck file already on disk asks. A file
// that isn't there yet has asked nothing, which is not an error: it is the
// first run.
func previousFronts(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	out := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		front, _, _ := strings.Cut(line, "\t")
		out[unfield(front)] = true
	}
	return out, nil
}

// recordStale keeps the list of this deck's dropped questions up to date:
// anything asked before and not now goes on it, and anything on it that is
// asked again comes off. Other decks' lines are left alone, since every deck
// written into a directory shares the one list.
func recordStale(path, deckName string, before, now map[string]bool) error {
	kept, err := readStale(path)
	if err != nil {
		return err
	}

	var out []stale
	listed := map[string]bool{}
	for _, s := range kept {
		if s.deck == deckName {
			if now[s.front] {
				continue
			}
			listed[s.front] = true
		}
		out = append(out, s)
	}

	today := time.Now().Format(time.DateOnly)
	var dropped []string
	for front := range before {
		if !now[front] && !listed[front] {
			dropped = append(dropped, front)
		}
	}
	slices.Sort(dropped)
	for _, front := range dropped {
		out = append(out, stale{date: today, deck: deckName, front: front})
	}

	if len(out) == 0 && !fileExists(path) {
		return nil
	}
	return writeStale(path, out)
}

func readStale(path string) ([]stale, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var out []stale
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		out = append(out, stale{date: parts[0], deck: parts[1], front: parts[2]})
	}
	return out, nil
}

func writeStale(path string, ss []stale) error {
	var b strings.Builder
	b.WriteString(staleHeader)
	for _, s := range ss {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", s.date, s.deck, s.front)
	}

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// unfield undoes field's quoting, so a question read back matches the one that
// was written.
func unfield(s string) string {
	if !strings.HasPrefix(s, `"`) || !strings.HasSuffix(s, `"`) || len(s) < 2 {
		return s
	}
	return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
}

// field quotes a value only when the format demands it. Anki reads a quote
// that doesn't open a field as literal text, and the fields written here hold
// no tabs or newlines, so in practice nothing is quoted and the file stays
// readable.
func field(s string) string {
	if !strings.ContainsAny(s, "\t\n\r") && !strings.HasPrefix(s, `"`) {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// img renders a media reference. Single quotes keep double quotes out of the
// fields entirely, so no row ever needs escaping.
func img(name string) string {
	return fmt.Sprintf("<img src='%s'>", name)
}

// thumb renders the same reference shown small, for a note carrying a row of
// cards rather than one. The width is asked for in the tag rather than by
// scaling the file, so the note shares the full-size image every other deck
// uses instead of adding a second copy to collection.media.
func thumb(name string, width int) string {
	return fmt.Sprintf("<img src='%s' width='%d'>", name, width)
}
