package facts

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/zeebo/blake3"
)

// FileFacts computes facts for a single file path.
// Returns "blake3:<hex> size:<n>" for existing files, "missing:true" for absent files.
func FileFacts(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "missing:true", nil
		}
		return "", fmt.Errorf("facts: stat %s: %w", path, err)
	}

	size := info.Size()

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("facts: open %s: %w", path, err)
	}
	defer f.Close()

	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("facts: read %s: %w", path, err)
	}

	sum := h.Sum(nil)
	hexStr := hex.EncodeToString(sum[:])

	return fmt.Sprintf("blake3:%s size:%d", hexStr, size), nil
}

// ParseFacts parses a fact string into key:value pairs.
func ParseFacts(raw string) map[string]string {
	result := make(map[string]string)
	if raw == "" {
		return result
	}
	for _, part := range strings.Fields(raw) {
		idx := strings.Index(part, ":")
		if idx < 0 {
			continue
		}
		result[part[:idx]] = part[idx+1:]
	}
	return result
}

// KnownFactKeys are the fact keys this version understands.
var KnownFactKeys = map[string]bool{
	"blake3":  true,
	"size":    true,
	"missing": true,
}

// CheckFact verifies a single file's recorded facts against the current filesystem.
// Returns (changed bool, reason string, err error).
func CheckFact(path string, recordedFacts string) (bool, string, error) {
	if recordedFacts == "" {
		return true, "no facts recorded", nil
	}

	facts := ParseFacts(recordedFacts)

	// Check for unknown keys
	for key := range facts {
		if !KnownFactKeys[key] {
			return true, fmt.Sprintf("unknown fact key %q", key), nil
		}
	}

	// Handle missing:true
	if facts["missing"] == "true" {
		_, err := os.Stat(path)
		if err == nil {
			return true, "file appeared", nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return false, "", nil // still missing, unchanged
		}
		return false, "", fmt.Errorf("facts: stat %s: %w", path, err)
	}

	// File should exist
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, "file disappeared", nil
		}
		return false, "", fmt.Errorf("facts: stat %s: %w", path, err)
	}

	// Size fast path
	if sizeStr, ok := facts["size"]; ok {
		recordedSize, err := strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			return true, "unreadable size fact", nil
		}
		if info.Size() != recordedSize {
			return true, "size differs", nil
		}
	}

	// Hash comparison
	if hashStr, ok := facts["blake3"]; ok {
		f, err := os.Open(path)
		if err != nil {
			return false, "", fmt.Errorf("facts: open %s: %w", path, err)
		}
		defer f.Close()

		h := blake3.New()
		if _, err := io.Copy(h, f); err != nil {
			return false, "", fmt.Errorf("facts: read %s: %w", path, err)
		}
		sum := h.Sum(nil)
		currentHash := hex.EncodeToString(sum[:])
		if currentHash != hashStr {
			return true, "hash differs", nil
		}
	}

	return false, "", nil
}
