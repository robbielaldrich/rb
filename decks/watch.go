package decks

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// WatchOptions configures RunWatcher.
type WatchOptions struct {
	DecksPath, CatalogPath, SetsPath string
	// Interval is how often the clipboard is read. macOS has no way to be
	// told the clipboard changed short of linking against AppKit, so it is
	// polled, and pbpaste is cheap enough to run twice a second.
	Interval time.Duration
	// Notify raises a macOS notification for each deck saved, since the
	// copy happens in the browser while the terminal is out of sight.
	Notify bool
}

// RunWatcher saves every decklist copied to the clipboard into the register
// until ctx is cancelled, named after its legend. Decklist sites offer a copy
// button for a list; this files what that button copies without the paste.
//
// Whatever is on the clipboard when it starts is read too, so a list copied
// just before is not lost; a deck already registered is skipped either way.
func RunWatcher(ctx context.Context, opts WatchOptions) error {
	p, latest, err := loadCatalog(opts.CatalogPath, opts.SetsPath)
	if err != nil {
		return err
	}

	w := &watcher{path: opts.DecksPath, pool: p, latestSet: latest, read: pbpaste}
	if opts.Notify {
		w.notify = notify
	}

	fmt.Printf("watching the clipboard for decklists, saving into %s · ctrl+c to stop\n", opts.DecksPath)
	tick := time.NewTicker(opts.Interval)
	defer tick.Stop()
	for {
		if err := w.poll(os.Stdout); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			fmt.Printf("\nadded %d %s\n", w.added, plural(w.added, "deck"))
			return nil
		case <-tick.C:
		}
	}
}

type watcher struct {
	path      string
	pool      *pool
	latestSet string
	read      func() (string, error)
	notify    func(title, message string)
	// last is the clipboard as it was last read, so a list is filed once
	// however long it stays copied.
	last  string
	added int
}

func (w *watcher) poll(out io.Writer) error {
	text, err := w.read()
	if err != nil {
		return fmt.Errorf("failed to read the clipboard: %w", err)
	}
	if text == w.last {
		return nil
	}
	w.last = text
	return w.record(text, out)
}

// record files one clipboard's worth of text if it is a decklist the
// register doesn't hold. Most of what gets copied isn't a decklist at all and
// is passed over in silence; only text with the look of one is reported, so a
// list in a shape Parse doesn't read yet doesn't vanish without a word.
func (w *watcher) record(text string, out io.Writer) error {
	d, err := Parse(text)
	if err != nil {
		if looksLikeDeck(text) {
			fmt.Fprintf(out, "%s copied something like a decklist, but couldn't read it: %v\n", stamp(), err)
		}
		return nil
	}

	// The register is read afresh each time rather than held, so a deck
	// renamed by hand while the watcher runs isn't written back over.
	reg, err := loadRegistry(w.path)
	if err != nil {
		return fmt.Errorf("failed to load the deck register: %w", err)
	}
	if have, ok := reg.duplicate(d); ok {
		fmt.Fprintf(out, "%s copied %s, already registered as %s\n", stamp(), d.Name, have.ID)
		return nil
	}

	d = reg.register(d, w.pool, w.latestSet, time.Now())
	if err := reg.save(w.path); err != nil {
		return fmt.Errorf("failed to save the deck register: %w", err)
	}
	w.added++

	summary := fmt.Sprintf("%s · %d cards, %d in the sideboard · %s",
		d.Title(), d.Size(false), d.Size(true)-d.Size(false), strings.Join(d.Sets, " "))
	fmt.Fprintf(out, "%s saved %s as %s\n", stamp(), summary, d.ID)
	w.pool.warnUnknown(d, out)
	if w.notify != nil {
		w.notify("Saved deck "+d.ID, summary)
	}
	return nil
}

// looksLikeDeck reports whether text has a few "<count> <card>" lines in it,
// which a sentence or a URL that happens to be copied won't.
func looksLikeDeck(text string) bool {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		if entryRe.MatchString(strings.TrimSpace(line)) {
			n++
		}
	}
	return n >= 3
}

func stamp() string { return time.Now().Format(time.TimeOnly) }

func pbpaste() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run pbpaste: %w", err)
	}
	return string(out), nil
}

// notify is best effort: a deck that saved but couldn't be announced has
// still saved, and the terminal says so.
func notify(title, message string) {
	// The text goes in as arguments rather than spliced into the script, so
	// a card name with a quote in it can't break the AppleScript.
	exec.Command("osascript",
		"-e", "on run argv",
		"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
		"-e", "end run",
		title, message,
	).Run()
}
