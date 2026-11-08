package ankigen

import (
	"strings"
	"testing"
)

var wantHeaders = []string{
	"#separator:Tab", "#html:true", "#tags column:3",
	"#deck column:4", "#notetype column:5", "#guid column:6",
}

func checkHeaders(t *testing.T, headers []string) {
	t.Helper()
	if strings.Join(headers, "\n") != strings.Join(wantHeaders, "\n") {
		t.Errorf("headers = %v, want %v", headers, wantHeaders)
	}
}

func TestSlug(t *testing.T) {
	for name, want := range map[string]string{
		"Hold 'Em":          "hold-em",
		"Guards!":           "guards",
		"Gentlemen's Duel":  "gentlemen-s-duel",
		"Kai'Sa - Survivor": "kai-sa-survivor",
	} {
		if got := slug(name); got != want {
			t.Errorf("slug(%q) = %q, want %q", name, got, want)
		}
	}
}

// The GUID is what lets a list grow without losing its notes' scheduling, so
// it must depend only on the tags naming the list and the card.
func TestGUIDIsStable(t *testing.T) {
	a := guidFor([]string{"riftbound::domain::calm", "riftbound::card::hush"})
	if a != guidFor([]string{"riftbound::domain::calm", "riftbound::card::hush"}) {
		t.Error("the same tags made two GUIDs")
	}
	if a == guidFor([]string{"riftbound::domain::mind", "riftbound::card::hush"}) {
		t.Error("two domains' notes for one card share a GUID")
	}
}
