package ankigen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"rb/rules"
)

// RulesOptions configure a pass over the rulings.
type RulesOptions struct {
	RulingsPath string
	ReviewPath  string
	OutDir      string
	DeckName    string
	// Revisit reopens the rulings already decided on, each prefilled with the
	// note it was given, rather than only the ones never seen.
	Revisit bool
}

// RulesResult reports what a pass left behind.
type RulesResult struct {
	DeckFile string
	// Notes is how many approved notes the deck holds, over every pass so
	// far; Skipped is how many rulings have been turned down.
	Notes   int
	Skipped int
	// Decided counts the rulings answered in this pass, Left the ones still
	// never seen.
	Decided int
	Left    int
}

const (
	statusApproved = "approved"
	statusSkipped  = "skipped"
)

// ReviewRulings walks the rulings, drafts a short question and answer from
// each, and asks for it to be kept, reworded or skipped. Every decision is
// written through as it is made, so a pass can be stopped at any point and
// resumed later, and the approved notes are written out as a deck at the end.
//
// Drafting is mechanical — the ruling's own question, and the sentences that
// open its answer — because the wording that makes a note worth reviewing is
// the reviewer's, not a rule the code could apply.
func ReviewRulings(opts RulesOptions, in io.Reader, out io.Writer) (RulesResult, error) {
	rs, err := rules.Load(opts.RulingsPath)
	if err != nil {
		return RulesResult{}, fmt.Errorf("failed to load the rulings: %w", err)
	}
	rev, err := loadReview(opts.ReviewPath)
	if err != nil {
		return RulesResult{}, err
	}

	var queue []rules.Ruling
	for _, r := range rs {
		if _, decided := rev.byKey[r.Key()]; !decided || opts.Revisit {
			queue = append(queue, r)
		}
	}

	fmt.Fprintf(out, "%d %s · %d decided · %d to review\n",
		len(rs), plural(len(rs), "ruling"), len(rev.byKey), len(queue))
	if len(queue) > 0 {
		fmt.Fprintf(out, "enter keeps the drafted note · e rewords it · s skips it · q saves and stops\n")
	}

	sc := bufio.NewScanner(in)
	decided := 0
	for i, r := range queue {
		d, held := rev.byKey[r.Key()]
		if !held {
			d = draft{Key: r.Key()}
			d.Front, d.Back = condense(r)
		}

		stop, err := reviewOne(sc, out, r, &d, i+1, len(queue))
		if err != nil {
			return RulesResult{}, err
		}
		if stop {
			break
		}

		d.DecidedAt = time.Now()
		rev.put(d)
		if err := rev.save(opts.ReviewPath, rs); err != nil {
			return RulesResult{}, err
		}
		decided++
	}
	if err := sc.Err(); err != nil {
		return RulesResult{}, fmt.Errorf("failed to read an answer: %w", err)
	}

	res := RulesResult{Decided: decided}
	for _, r := range rs {
		switch d, held := rev.byKey[r.Key()]; {
		case !held:
			res.Left++
		case d.Status == statusSkipped:
			res.Skipped++
		}
	}

	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return RulesResult{}, fmt.Errorf("failed to create %s: %w", opts.OutDir, err)
	}
	d := rulesDeck(opts.DeckName, rs, rev)
	res.DeckFile, res.Notes = filepath.Join(opts.OutDir, "riftbound-rulings.txt"), len(d.notes)
	if err := d.write(res.DeckFile); err != nil {
		return RulesResult{}, fmt.Errorf("failed to write deck: %w", err)
	}
	return res, nil
}

// reviewOne settles one ruling, and reports whether the reviewer asked to
// stop. The ruling is shown in full: a note is only worth keeping if it says
// what the ruling says, and the reviewer needs the whole ruling to tell.
func reviewOne(sc *bufio.Scanner, out io.Writer, r rules.Ruling, d *draft, n, total int) (stop bool, err error) {
	fmt.Fprintf(out, "\n[%d/%d] %s · %s\n", n, total, r.Category, r.Topic)
	writeRuling(out, r)

	for {
		fmt.Fprintf(out, "\n%s%s", draftLine("q", d.Front), draftLine("a", d.Back))
		fmt.Fprintf(out, "  keep? [enter/e/s/q] ")
		if !sc.Scan() {
			fmt.Fprintln(out)
			return true, nil
		}
		switch answer := strings.ToLower(strings.TrimSpace(sc.Text())); answer {
		case "", "a", "y":
			d.Status = statusApproved
			return false, nil
		case "s", "n":
			d.Status = statusSkipped
			return false, nil
		case "q":
			return true, nil
		case "e":
			reword(sc, out, d)
		default:
			fmt.Fprintf(out, "  %q is not one of them\n", answer)
		}
	}
}

// reword takes a replacement for either half of the note, keeping whichever
// line is answered with an empty one.
func reword(sc *bufio.Scanner, out io.Writer, d *draft) {
	fmt.Fprintf(out, "  type a replacement, or enter to keep the line\n")
	if s := ask(sc, out, "  q: "); s != "" {
		d.Front = s
	}
	if s := ask(sc, out, "  a: "); s != "" {
		d.Back = s
	}
}

func ask(sc *bufio.Scanner, out io.Writer, prompt string) string {
	fmt.Fprint(out, prompt)
	if !sc.Scan() {
		return ""
	}
	return strings.TrimSpace(sc.Text())
}

const reviewWidth = 76

// writeRuling prints the ruling being asked about, dimmed so that the draft
// note under it stands out from the material it was drawn from.
func writeRuling(out io.Writer, r rules.Ruling) {
	for _, line := range wrap(r.Answer, reviewWidth-4) {
		if line == "" {
			fmt.Fprintln(out)
			continue
		}
		fmt.Fprintf(out, "    %s\n", dim(line))
	}
	if len(r.Rules) > 0 {
		fmt.Fprintf(out, "    %s\n", dim("core rules "+strings.Join(r.Rules, ", ")))
	}
	if r.Source != "" {
		fmt.Fprintf(out, "    %s\n", dim(r.Source))
	}
}

// draftLine lays out one half of the draft note under a two-letter label, with
// its continuation lines under the label rather than the text.
func draftLine(label, text string) string {
	lines := wrap(text, reviewWidth-5)
	if len(lines) == 0 {
		lines = []string{""}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  %s: %s\n", label, lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(&b, "     %s\n", line)
	}
	return b.String()
}

// wrap breaks text into lines no wider than width, keeping the blank lines
// that separate a ruling's paragraphs from its bullets.
func wrap(text string, width int) []string {
	var out []string
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, w := range words[1:] {
			if len(line)+1+len(w) > width {
				out, line = append(out, line), w
				continue
			}
			line += " " + w
		}
		out = append(out, line)
	}
	return out
}

// Styling is off when NO_COLOR is set (https://no-color.org).
func dim(s string) string {
	if s == "" || os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// rulesDeck collects the approved notes in the order the rulings file holds,
// so that the deck reads topic by topic.
func rulesDeck(name string, rs []rules.Ruling, rev *review) deck {
	d := deck{name: name, notetype: "Basic"}
	for _, r := range rs {
		n, held := rev.byKey[r.Key()]
		if !held || n.Status != statusApproved {
			continue
		}
		d.notes = append(d.notes, note{front: n.Front, back: noteBack(n.Back, r), tags: rulingTags(r)})
	}
	return d
}

// noteBack cites the rules the ruling rests on, which is worth having on the
// card without being part of the answer the reviewer worded.
func noteBack(back string, r rules.Ruling) string {
	if len(r.Rules) == 0 {
		return back
	}
	return back + "<br><br><small>Core Rules " + strings.Join(r.Rules, ", ") + "</small>"
}

func rulingTags(r rules.Ruling) []string {
	tags := []string{"riftbound::rules", "riftbound::rules::" + r.Category}
	if r.Slug != "" {
		tags = append(tags, "riftbound::topic::"+r.Slug)
	}
	return tags
}

// draft is the note a ruling has been given, and what was decided about it.
// Skipped rulings keep their draft too, so that a later pass can see what was
// turned down rather than offering it again from scratch.
type draft struct {
	Key       string    `json:"key"`
	Status    string    `json:"status"`
	Front     string    `json:"front"`
	Back      string    `json:"back"`
	DecidedAt time.Time `json:"decidedAt"`
}

// review is the running record of those decisions, kept beside the rulings so
// that reviewing 174 of them doesn't have to happen in one sitting.
type review struct {
	Notes []draft `json:"notes"`
	byKey map[string]draft
}

func loadReview(path string) (*review, error) {
	rev := &review{byKey: map[string]draft{}}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return rev, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, rev); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	for _, d := range rev.Notes {
		rev.byKey[d.Key] = d
	}
	return rev, nil
}

func (rev *review) put(d draft) { rev.byKey[d.Key] = d }

// save writes the decisions in rulings order, so that the file can be read
// and edited alongside the dataset it describes. A decision whose ruling has
// gone from the dataset is kept at the end rather than dropped, since the
// ruling may only have been edited out for now.
func (rev *review) save(path string, rs []rules.Ruling) error {
	rev.Notes = rev.Notes[:0]
	filed := map[string]bool{}
	for _, r := range rs {
		if d, held := rev.byKey[r.Key()]; held {
			rev.Notes = append(rev.Notes, d)
			filed[d.Key] = true
		}
	}
	var orphans []string
	for key := range rev.byKey {
		if !filed[key] {
			orphans = append(orphans, key)
		}
	}
	slices.Sort(orphans)
	for _, key := range orphans {
		rev.Notes = append(rev.Notes, rev.byKey[key])
	}

	// The rulings this file answers are full of quoted card text and of
	// comparisons like "damage >= Might", which the encoder escapes into
	// entities unless told not to; the file is meant to be read and edited by
	// hand alongside them.
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(rev); err != nil {
		return fmt.Errorf("failed to marshal the reviewed notes: %w", err)
	}
	return writeFile(path, b.Bytes())
}

// writeFile writes a file atomically, so an interrupted write can't leave a
// truncated one behind.
func writeFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", tmp, path, err)
	}
	return nil
}
