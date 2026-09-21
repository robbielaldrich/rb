package ankigen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func mediaFixture(t *testing.T, profiles ...string) (mediaDir, ankiDir string) {
	t.Helper()
	dir := t.TempDir()
	mediaDir = filepath.Join(dir, "media")
	ankiDir = filepath.Join(dir, "Anki2")
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "rb-a.jpg"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles {
		if err := os.MkdirAll(filepath.Join(ankiDir, p, "collection.media"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Not a profile: Anki keeps other folders beside them.
	if err := os.MkdirAll(filepath.Join(ankiDir, "addons21"), 0o755); err != nil {
		t.Fatal(err)
	}
	return mediaDir, ankiDir
}

func TestAddMediaUsesTheOnlyProfile(t *testing.T) {
	mediaDir, ankiDir := mediaFixture(t, "User 1")
	if err := AddMedia(mediaDir, ankiDir, "", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ankiDir, "User 1", "collection.media", "rb-a.jpg")); err != nil {
		t.Errorf("image was not copied: %v", err)
	}
}

func TestAddMediaAsksWhichOfSeveralProfiles(t *testing.T) {
	mediaDir, ankiDir := mediaFixture(t, "User 1", "Rob")
	if err := AddMedia(mediaDir, ankiDir, "", &bytes.Buffer{}); err == nil {
		t.Fatal("want an error when the profile is ambiguous")
	}
	if err := AddMedia(mediaDir, ankiDir, "Rob", &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if err := AddMedia(mediaDir, ankiDir, "Nobody", &bytes.Buffer{}); err == nil {
		t.Fatal("want an error for an unknown profile")
	}
}
