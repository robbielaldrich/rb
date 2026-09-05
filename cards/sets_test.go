package cards

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func set(name, id, published string, count int) Set {
	return Set{Name: name, SetID: id, CardCount: count, PublishedOn: published + "T00:00:00"}
}

// The real set list, in the order sets.json happens to hold it.
var releases = []Set{
	set("Unleashed", "UNL", "2026-05-08", 280),
	set("Vendetta", "VEN", "2026-07-31", 358),
	set("Origins", "OGN", "2025-10-31", 352),
	set("Origins: Proving Grounds", "OGS", "2025-10-31", 24),
	set("Riftbound Organized Play Promotional Cards", "OPP", "2026-08-14", 133),
	set("Spiritforged", "SFD", "2026-02-13", 288),
}

func TestLatest(t *testing.T) {
	for _, tc := range []struct {
		name string
		asOf string
		want string
	}{
		// A promo set printed after Vendetta doesn't move the format on.
		{"newest set in print", "2026-09-05", "VEN"},
		// A set announced but not out yet can't be built from.
		{"unreleased set ignored", "2026-07-30", "UNL"},
		// Proving Grounds ships with Origins but isn't what the format is called.
		{"released the same day as a bigger set", "2025-12-01", "OGN"},
	} {
		got, err := Latest(releases, day(tc.asOf))
		if err != nil {
			t.Errorf("%s: Latest: %v", tc.name, err)
			continue
		}
		if got.SetID != tc.want {
			t.Errorf("%s: Latest as of %s = %s, want %s", tc.name, tc.asOf, got.SetID, tc.want)
		}
	}
}

func TestLatestBeforeAnySetShipped(t *testing.T) {
	if _, err := Latest(releases, day("2025-01-01")); err == nil {
		t.Error("Latest reported a set from before the game was published")
	}
}
