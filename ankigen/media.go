package ankigen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DefaultAnkiDir is where Anki keeps its profiles on this machine.
func DefaultAnkiDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(home, "Library")); err == nil {
		return filepath.Join(home, "Library", "Application Support", "Anki2"), nil
	}
	return filepath.Join(home, ".local", "share", "Anki2"), nil
}

// AddMedia copies the generated images into a profile's collection.media, which
// a text import doesn't do. With no profile named it uses the only one there is,
// and otherwise says which to choose from.
func AddMedia(mediaDir, ankiDir, profile string, w io.Writer) error {
	profiles, err := ankiProfiles(ankiDir)
	if err != nil {
		return err
	}
	if profile == "" {
		switch len(profiles) {
		case 0:
			return fmt.Errorf("no Anki profiles found in %s", ankiDir)
		case 1:
			profile = profiles[0]
		default:
			return fmt.Errorf("several Anki profiles found, name one with -profile: %s", strings.Join(profiles, ", "))
		}
	} else if !slices.Contains(profiles, profile) {
		return fmt.Errorf("no Anki profile %q in %s, have: %s", profile, ankiDir, strings.Join(profiles, ", "))
	}

	files, err := os.ReadDir(mediaDir)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", mediaDir, err)
	}
	dest := filepath.Join(ankiDir, profile, "collection.media")
	n := 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(mediaDir, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", f.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dest, f.Name()), data, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", f.Name(), err)
		}
		n++
	}
	fmt.Fprintf(w, "copied %d files into %s\n", n, dest)
	return nil
}

// ankiProfiles lists the profile folders that hold a collection.media.
func ankiProfiles(ankiDir string) ([]string, error) {
	entries, err := os.ReadDir(ankiDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", ankiDir, err)
	}
	var out []string
	for _, e := range entries {
		if st, err := os.Stat(filepath.Join(ankiDir, e.Name(), "collection.media")); err == nil && st.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}
