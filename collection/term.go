package collection

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// key is one decoded keystroke. Printable keys carry their rune and an empty
// name; everything else is named and carries no rune.
type key struct {
	r    rune
	name string
}

// decodeKeys splits one read from the terminal into keystrokes. Escape
// sequences arrive whole in a single read, so a lone 0x1b in the chunk is a
// real Escape press rather than the head of an arrow key.
func decodeKeys(b []byte) []key {
	var out []key
	for len(b) > 0 {
		k, n := decodeKey(b)
		out = append(out, k)
		b = b[n:]
	}
	return out
}

func decodeKey(b []byte) (key, int) {
	switch c := b[0]; {
	case c == 0x1b && len(b) >= 3 && (b[1] == '[' || b[1] == 'O'):
		switch b[2] {
		case 'A':
			return key{name: "up"}, 3
		case 'B':
			return key{name: "down"}, 3
		case 'C':
			return key{name: "right"}, 3
		case 'D':
			return key{name: "left"}, 3
		}
		// Some other control sequence. Swallow it up to and including its
		// final byte, so its tail doesn't arrive as typed text.
		n := 2
		for n < len(b) && (b[n] < 0x40 || b[n] > 0x7e) {
			n++
		}
		if n < len(b) {
			n++
		}
		return key{name: "unknown"}, n
	case c == 0x1b:
		return key{name: "esc"}, 1
	case c == '\r' || c == '\n':
		return key{name: "enter"}, 1
	case c == 0x7f || c == 0x08:
		return key{name: "backspace"}, 1
	case c == 0x03:
		return key{name: "ctrl+c"}, 1
	case c == 0x04:
		return key{name: "ctrl+d"}, 1
	case c == 0x15:
		return key{name: "ctrl+u"}, 1
	case c == 0x17:
		return key{name: "ctrl+w"}, 1
	// Raw mode has already turned off the terminal's own handling of ^Z, so
	// it arrives as an ordinary byte rather than suspending rb.
	case c == 0x1a:
		return key{name: "ctrl+z"}, 1
	case c < 0x20:
		return key{name: "unknown"}, 1
	}
	r, n := utf8.DecodeRune(b)
	return key{r: r}, n
}

// runLoop drives a full-screen prompt: it puts the terminal into raw mode,
// paints what frame lays out, and feeds every keystroke to handle until it
// asks to stop or fails.
func runLoop(in, out *os.File, frame func(width int) ([]string, int, int), handle func(key) (bool, error)) error {
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("stdin is not a terminal")
	}
	restore, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to put the terminal into raw mode: %w", err)
	}
	defer term.Restore(fd, restore)

	s := &screen{w: out}
	defer s.finish()

	buf := make([]byte, 128)
	for {
		width, _, err := term.GetSize(fd)
		if err != nil || width < 20 {
			width = 80
		}
		lines, caretRow, caretCol := frame(width)
		s.render(lines, caretRow, caretCol)

		n, err := in.Read(buf)
		if err != nil {
			return fmt.Errorf("failed to read from the terminal: %w", err)
		}
		for _, k := range decodeKeys(buf[:n]) {
			stop, err := handle(k)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}
	}
}

// screen paints a block of lines wherever the cursor happens to be and
// repaints it in place, so the frame updates without filling the scrollback.
// Between frames the cursor is parked on the caret, which doubles as the
// anchor the next frame climbs back to.
type screen struct {
	w        io.Writer
	lines    int
	caretRow int
	painted  bool
}

func (s *screen) render(lines []string, caretRow, caretCol int) {
	if len(lines) == 0 {
		return
	}
	var b strings.Builder
	if s.painted && s.caretRow > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", s.caretRow)
	}
	// Clearing to the end of the display wipes the tail of a frame that was
	// taller than this one.
	b.WriteString("\r\x1b[J")
	b.WriteString(strings.Join(lines, "\r\n"))

	if up := len(lines) - 1 - caretRow; up > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", up)
	}
	b.WriteString("\r")
	if caretCol > 0 {
		fmt.Fprintf(&b, "\x1b[%dC", caretCol)
	}

	s.lines, s.caretRow, s.painted = len(lines), caretRow, true
	io.WriteString(s.w, b.String())
}

// finish leaves the cursor on its own line below the last frame, so whatever
// the shell prints next doesn't land on top of it.
func (s *screen) finish() {
	if !s.painted {
		return
	}
	if down := s.lines - 1 - s.caretRow; down > 0 {
		fmt.Fprintf(s.w, "\x1b[%dB", down)
	}
	io.WriteString(s.w, "\r\n")
}

// Styling is off when NO_COLOR is set (https://no-color.org).
var useColor = os.Getenv("NO_COLOR") == ""

func paint(s, code string) string {
	if !useColor || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func dim(s string) string  { return paint(s, "2") }
func bold(s string) string { return paint(s, "1") }

// truncate cuts a string to at most n columns, marking the cut with an
// ellipsis. It counts runes rather than display cells, which is exact for the
// card names and set codes it is used on.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func pad(s string, n int) string {
	if d := n - utf8.RuneCountInString(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}
