package ankigen

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
	return nil
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
