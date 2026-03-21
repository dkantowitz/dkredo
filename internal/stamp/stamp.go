package stamp

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dkantowitz/dk-redo/internal/hasher"
)

// Stamp represents the stored facts about a set of files under a given label.
type Stamp struct {
	Label string
	Files []FileFact // sorted by path
}

// FileFact pairs a file path with its raw facts string.
type FileFact struct {
	Path  string
	Facts string // "blake3:<hex> size:<n>" or "missing:true"
}

// CompareResult describes how the current file state differs from a stamp.
type CompareResult struct {
	Changed      bool
	ChangedFiles []ChangedFile
	Warnings     []string
}

// ChangedFile records why a single file is considered changed.
type ChangedFile struct {
	Path   string
	Reason string // "modified", "added", "removed", "appeared", "disappeared", "unknown_facts"
}

// FormatFacts converts hasher.Facts to the string representation stored in stamps.
func FormatFacts(f hasher.Facts) string {
	if f.Missing {
		return "missing:true"
	}
	return fmt.Sprintf("blake3:%s size:%d", f.Blake3, f.Size)
}

// maxLineLength is the maximum allowed line length when reading stamp files.
// Lines longer than this are treated as corrupt input.
const maxLineLength = 1024 * 1024 // 1 MiB

// Read loads a stamp from disk. Returns (nil, nil) if the stamp file doesn't exist.
func Read(stampsDir, label string) (*Stamp, error) {
	path := filepath.Join(stampsDir, EscapeLabel(label))

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stamp: read %s: %w", label, err)
	}
	defer f.Close()

	s := &Stamp{Label: label}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineLength)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for binary/non-text data: NUL bytes
		if strings.ContainsRune(line, '\x00') {
			return nil, fmt.Errorf("stamp: corrupt stamp %q: binary data at line %d", label, lineNum)
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("stamp: corrupt stamp %q: missing tab at line %d", label, lineNum)
		}

		decodedPath := DecodePath(parts[0])
		facts := parts[1]

		s.Files = append(s.Files, FileFact{
			Path:  decodedPath,
			Facts: facts,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stamp: corrupt stamp %q: %w", label, err)
	}

	return s, nil
}

// Write atomically writes a stamp to disk. Creates stampsDir if needed.
// Files are sorted by path before writing.
func Write(stampsDir string, s *Stamp) error {
	if err := os.MkdirAll(stampsDir, 0o755); err != nil {
		return fmt.Errorf("stamp: create dir: %w", err)
	}

	// Sort files by path
	sorted := make([]FileFact, len(s.Files))
	copy(sorted, s.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	// Write to temp file, then rename for atomicity
	stampPath := filepath.Join(stampsDir, EscapeLabel(s.Label))

	tmp, err := os.CreateTemp(stampsDir, ".stamp-tmp-*")
	if err != nil {
		return fmt.Errorf("stamp: write: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	w := bufio.NewWriter(tmp)
	for _, ff := range sorted {
		encoded := EncodePath(ff.Path)
		if _, err := fmt.Fprintf(w, "%s\t%s\n", encoded, ff.Facts); err != nil {
			return fmt.Errorf("stamp: write: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("stamp: write: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("stamp: write: %w", err)
	}

	if err := os.Rename(tmpName, stampPath); err != nil {
		return fmt.Errorf("stamp: write: %w", err)
	}

	success = true
	return nil
}

// parseFacts parses a facts string into key:value pairs.
// Returns the map and a list of any unknown keys encountered.
func parseFacts(facts string) (map[string]string, []string) {
	m := make(map[string]string)
	var unknowns []string
	known := map[string]bool{"blake3": true, "size": true, "missing": true}

	parts := strings.Fields(facts)
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			unknowns = append(unknowns, part)
			continue
		}
		key, val := kv[0], kv[1]
		m[key] = val
		if !known[key] {
			unknowns = append(unknowns, key)
		}
	}

	return m, unknowns
}

// Compare checks if current file facts match the stamp.
func Compare(s *Stamp, currentFacts []hasher.FileFacts) CompareResult {
	result := CompareResult{}

	// Build maps for lookup
	stampMap := make(map[string]string, len(s.Files))
	for _, ff := range s.Files {
		stampMap[ff.Path] = ff.Facts
	}

	currentMap := make(map[string]hasher.Facts, len(currentFacts))
	for _, cf := range currentFacts {
		currentMap[cf.Path] = cf.Facts
	}

	// Check for files in stamp but not in current (removed)
	for _, ff := range s.Files {
		if _, ok := currentMap[ff.Path]; !ok {
			result.Changed = true
			result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
				Path:   ff.Path,
				Reason: "removed",
			})
		}
	}

	// Check for files in current but not in stamp (added)
	for _, cf := range currentFacts {
		if _, ok := stampMap[cf.Path]; !ok {
			result.Changed = true
			result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
				Path:   cf.Path,
				Reason: "added",
			})
		}
	}

	// Compare files present in both
	for _, ff := range s.Files {
		currentF, ok := currentMap[ff.Path]
		if !ok {
			continue // already handled as "removed"
		}

		storedFacts, unknowns := parseFacts(ff.Facts)

		if len(unknowns) > 0 {
			result.Changed = true
			result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
				Path:   ff.Path,
				Reason: "unknown_facts",
			})
			for _, u := range unknowns {
				result.Warnings = append(result.Warnings, fmt.Sprintf("unknown fact key %q for path %q", u, ff.Path))
			}
			continue
		}

		// Check missing state transitions
		if storedFacts["missing"] == "true" {
			if !currentF.Missing {
				result.Changed = true
				result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
					Path:   ff.Path,
					Reason: "appeared",
				})
			}
			continue
		}

		// Stamp says file existed, check if now missing
		if currentF.Missing {
			result.Changed = true
			result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
				Path:   ff.Path,
				Reason: "disappeared",
			})
			continue
		}

		// Size fast path
		if sizeStr, ok := storedFacts["size"]; ok {
			currentSize := fmt.Sprintf("%d", currentF.Size)
			if sizeStr != currentSize {
				result.Changed = true
				result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
					Path:   ff.Path,
					Reason: "modified",
				})
				continue
			}
		}

		// Blake3 comparison
		if blake3Str, ok := storedFacts["blake3"]; ok {
			if blake3Str != currentF.Blake3 {
				result.Changed = true
				result.ChangedFiles = append(result.ChangedFiles, ChangedFile{
					Path:   ff.Path,
					Reason: "modified",
				})
				continue
			}
		}
	}

	return result
}

// Append merges new facts into an existing stamp.
// Files in newFacts update or add to the stamp. Files not in newFacts are preserved.
func Append(existing *Stamp, newFacts []hasher.FileFacts) *Stamp {
	// Build map from existing
	fileMap := make(map[string]FileFact, len(existing.Files))
	for _, ff := range existing.Files {
		fileMap[ff.Path] = ff
	}

	// Update/add from newFacts
	for _, nf := range newFacts {
		fileMap[nf.Path] = FileFact{
			Path:  nf.Path,
			Facts: FormatFacts(nf.Facts),
		}
	}

	// Collect and sort
	files := make([]FileFact, 0, len(fileMap))
	for _, ff := range fileMap {
		files = append(files, ff)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return &Stamp{
		Label: existing.Label,
		Files: files,
	}
}
