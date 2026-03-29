package stamp

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StampPath returns the full path to a stamp file.
func StampPath(stampsDir, label string) string {
	return filepath.Join(stampsDir, EscapeLabel(label))
}

// ReadStamp reads a stamp file. Returns an empty state if the file doesn't exist.
func ReadStamp(stampsDir, label string, verbose bool) (*StampState, error) {
	path := StampPath(stampsDir, label)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		s := NewStampState(label)
		if verbose {
			fmt.Fprintf(os.Stderr, "stamp: no file for %s\n", label)
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("stamp: read %s: %w", path, err)
	}

	s := NewStampState(label)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		idx := strings.Index(line, "\t")
		var path, facts string
		if idx < 0 {
			path = DecodePath(line)
			facts = ""
		} else {
			path = DecodePath(line[:idx])
			facts = line[idx+1:]
		}
		s.Entries = append(s.Entries, Entry{Path: path, Facts: facts})
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "stamp: loaded %s (%d entries)\n", StampPath(stampsDir, label), len(s.Entries))
	}
	return s, nil
}

// WriteStamp atomically writes a stamp file.
func WriteStamp(stampsDir string, state *StampState, verbose bool) error {
	if err := os.MkdirAll(stampsDir, 0755); err != nil {
		return fmt.Errorf("stamp: create dir %s: %w", stampsDir, err)
	}

	path := StampPath(stampsDir, state.Label)
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())

	var b strings.Builder
	for _, e := range state.Entries {
		b.WriteString(EncodePath(e.Path))
		b.WriteByte('\t')
		b.WriteString(e.Facts)
		b.WriteByte('\n')
	}

	if err := os.WriteFile(tmp, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("stamp: write temp %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("stamp: rename %s -> %s: %w", tmp, path, err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "stamp: wrote %s (%d entries)\n", path, len(state.Entries))
	}
	return nil
}

// FindStampsDir searches upward from cwd for a .stamps/ directory.
// Returns empty string if not found.
func FindStampsDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".stamps")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// StampsDir finds or creates a .stamps/ directory.
// If an existing one is found by walking upward, use it.
// Otherwise, create in cwd.
func StampsDir() (string, error) {
	found := FindStampsDir()
	if found != "" {
		return found, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("stamp: getwd: %w", err)
	}
	path := filepath.Join(cwd, ".stamps")
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", fmt.Errorf("stamp: create .stamps: %w", err)
	}
	return path, nil
}

// StampsParent returns the parent directory of the stamps dir.
func StampsParent(stampsDir string) string {
	return filepath.Dir(stampsDir)
}
