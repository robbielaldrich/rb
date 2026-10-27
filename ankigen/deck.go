package ankigen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The output is Anki's plain-text import format, documented at
// https://docs.ankiweb.net/importing/text-files.html: a few "#key:value"
// option lines, then one tab-separated note per line. Re-importing a
// regenerated file updates the existing notes instead of duplicating them,
// matching each by its GUID where the row gives one and by its first field
// where it doesn't.
//
// Every deck shares one file, each row naming its deck and notetype, so a
// single import brings in all of them.

// DeckFile is the import file every deck writes its notes into.
const DeckFile = "riftbound.txt"

// note is one row of the import file.
type note struct {
	front string
	back  string
	tags  []string
	// guid is the note's identity in Anki, or empty to let Anki match it by
	// its first field. A Basic note's question is fixed, so it can be; a cloze
	// note's first field is the whole list, which changes whenever it grows.
	guid string
}

type deck struct {
	name     string
	notetype string
	notes    []note
}

// row is one note line of an import file already on disk, kept as written so
// the decks this run doesn't regenerate pass through untouched.
type row struct {
	deck  string
	label string
	line  string
}

// write puts the deck's notes into the shared import file at path, replacing
// the rows it wrote last time and keeping every other deck's. A deck keeps its
// place in the file, and one written for the first time goes at the end.
func (d *deck) write(path string) error {
	// Read before the file is rewritten: what it used to ask is the only
	// record of notes this run has stopped generating.
	rows, err := readRows(path)
	if err != nil {
		return err
	}

	before := map[string]bool{}
	for _, r := range rows {
		if r.deck == d.name {
			before[r.label] = true
		}
	}

	var lines []string
	placed := false
	for _, r := range rows {
		if r.deck != d.name {
			lines = append(lines, r.line)
		} else if !placed {
			lines = append(lines, d.lines()...)
			placed = true
		}
	}
	if !placed {
		lines = append(lines, d.lines()...)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "#separator:Tab\n")
	fmt.Fprintf(&b, "#html:true\n")
	fmt.Fprintf(&b, "#tags column:3\n")
	fmt.Fprintf(&b, "#deck column:4\n")
	fmt.Fprintf(&b, "#notetype column:5\n")
	fmt.Fprintf(&b, "#guid column:6\n")
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return recordStale(filepath.Join(filepath.Dir(path), staleFile), d.name, before, d.labels())
}

// lines renders the deck's notes as rows of the import file.
func (d *deck) lines() []string {
	out := make([]string, 0, len(d.notes))
	for _, n := range d.notes {
		out = append(out, strings.Join([]string{
			field(n.front), field(n.back), field(strings.Join(n.tags, " ")),
			field(d.name), field(d.notetype), field(n.guid),
		}, "\t"))
	}
	return out
}

// readRows reads the note rows of an import file already on disk. A file that
// isn't there yet holds nothing, which is not an error: it is the first run.
//
// A row of four fields is from before the file named each row's notetype and
// GUID, when every note was Basic and matched by its question, and is brought
// up to six so it can sit under the headers written now.
func readRows(path string) ([]row, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var rows []row
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		switch len(fields) {
		case 4:
			fields = append(fields, "Basic", "")
			line = strings.Join(fields, "\t")
		case 6:
		default:
			return nil, fmt.Errorf("%s has a row with %d fields, want front, back, tags, deck, notetype and guid: %q", path, len(fields), line)
		}
		deckName := unfield(fields[3])
		rows = append(rows, row{
			deck:  deckName,
			label: label(deckName, unfield(fields[0]), strings.Fields(unfield(fields[2])), unfield(fields[5])),
			line:  line,
		})
	}
	return rows, nil
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
# Search Anki for the note — its question, or the search given in its place —
# delete it, then delete the line.
#
# Written by ` + "`rb gen-anki`" + `. A question that comes back — a new set can
# make a dropped one worth asking again — is taken off this list by the run
# that starts asking it.
#
# date	deck	note
`

// stale is one note that was generated once and isn't any more.
type stale struct {
	date  string
	deck  string
	label string
}

// labels lists what this deck's notes go by in to-delete.txt.
func (d *deck) labels() map[string]bool {
	out := make(map[string]bool, len(d.notes))
	for _, n := range d.notes {
		out[label(d.name, n.front, n.tags, n.guid)] = true
	}
	return out
}

// label is how to-delete.txt names a note, which is how it is found in Anki to
// be deleted. A note matched by its first field is its question. A note
// matched by GUID has no question to search for, so it is named by an Anki
// search for its deck and tags, which pick out that one note just as the GUID
// they make does.
func label(deckName, front string, tags []string, guid string) string {
	if guid == "" {
		return front
	}
	terms := []string{fmt.Sprintf("deck:%q", deckName)}
	for _, t := range tags {
		terms = append(terms, "tag:"+t)
	}
	return strings.Join(terms, " ")
}

// recordStale keeps the list of this deck's dropped notes up to date: anything
// generated before and not now goes on it, and anything on it that is
// generated again comes off. Other decks' lines are left alone, since every deck
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
			if now[s.label] {
				continue
			}
			listed[s.label] = true
		}
		out = append(out, s)
	}

	today := time.Now().Format(time.DateOnly)
	var dropped []string
	for l := range before {
		if !now[l] && !listed[l] {
			dropped = append(dropped, l)
		}
	}
	slices.Sort(dropped)
	for _, l := range dropped {
		out = append(out, stale{date: today, deck: deckName, label: l})
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
		out = append(out, stale{date: parts[0], deck: parts[1], label: parts[2]})
	}
	return out, nil
}

func writeStale(path string, ss []stale) error {
	var b strings.Builder
	b.WriteString(staleHeader)
	for _, s := range ss {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", s.date, s.deck, s.label)
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
