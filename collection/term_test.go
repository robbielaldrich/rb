package collection

import (
	"strings"
	"testing"
)

// typeKeys drives a prompt the way a terminal would, feeding decoded
// keystrokes one at a time to whichever handler is being tested. Named keys
// are written as "<enter>", everything else stands for itself.
//
// The editor, the validator and the stats viewer are three prompts over the
// one runLoop, so they are typed at through the one helper.
func typeKeys(t *testing.T, handle func(key) (bool, error), s string) {
	t.Helper()
	for len(s) > 0 {
		var k key
		if strings.HasPrefix(s, "<") {
			end := strings.Index(s, ">")
			k, s = key{name: s[1:end]}, s[end+1:]
		} else {
			r := []rune(s)[0]
			k, s = key{r: r}, s[len(string(r)):]
		}
		if _, err := handle(k); err != nil {
			t.Fatalf("handle(%v): %v", k, err)
		}
	}
}
