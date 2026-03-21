// Package hasher computes BLAKE3 hashes of files and directories.
package hasher

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/zeebo/blake3"
)

// Facts holds the hash and metadata for a single file.
type Facts struct {
	Blake3  string // hex digest, empty if missing
	Size    int64  // byte count, -1 if missing
	Missing bool   // true if file did not exist
}

// FileFacts pairs a relative path with its Facts.
type FileFacts struct {
	Path  string
	Facts Facts
}

// HashFile returns per-file facts for a single file path.
// Symlinks are followed — the hash reflects the target content.
// Returns Facts{Missing: true, Size: -1} for absent files (not an error).
// Returns an error for permission denied or other I/O failures.
func HashFile(path string) (Facts, error) {
	info, err := os.Stat(path) // follows symlinks
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Facts{Missing: true, Size: -1}, nil
		}
		return Facts{}, fmt.Errorf("hasher: %w", err)
	}

	size := info.Size()

	f, err := os.Open(path)
	if err != nil {
		return Facts{}, fmt.Errorf("hasher: %w", err)
	}
	defer f.Close()

	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return Facts{}, fmt.Errorf("hasher: %w", err)
	}

	digest := h.Sum(nil)
	return Facts{
		Blake3: hex.EncodeToString(digest),
		Size:   size,
	}, nil
}

// HashDir walks a directory recursively, hashing all regular files.
// Symlinks are followed. Circular symlink loops are detected and reported as errors.
// Returns a sorted list of (path, Facts) pairs with paths relative to dirPath.
func HashDir(dirPath string) ([]FileFacts, error) {
	realRoot, err := filepath.EvalSymlinks(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("hasher: directory does not exist: %s", dirPath)
		}
		return nil, fmt.Errorf("hasher: %w", err)
	}

	visited := map[string]bool{
		realRoot: true,
	}

	var results []FileFacts

	err = walkDir(realRoot, "", visited, &results)
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}

// walkDir recursively walks a directory, following symlinks and detecting loops.
// relDir is the path relative to the root dirPath.
func walkDir(realDir, relDir string, visited map[string]bool, results *[]FileFacts) error {
	entries, err := os.ReadDir(realDir)
	if err != nil {
		return fmt.Errorf("hasher: %w", err)
	}

	for _, entry := range entries {
		relPath := filepath.Join(relDir, entry.Name())
		realPath := filepath.Join(realDir, entry.Name())

		// Resolve symlinks
		resolvedPath, err := filepath.EvalSymlinks(realPath)
		if err != nil {
			return fmt.Errorf("hasher: %w", err)
		}

		info, err := os.Stat(resolvedPath)
		if err != nil {
			return fmt.Errorf("hasher: %w", err)
		}

		if info.IsDir() {
			if visited[resolvedPath] {
				return fmt.Errorf("hasher: symlink loop detected at %s", relPath)
			}
			visited[resolvedPath] = true
			if err := walkDir(resolvedPath, relPath, visited, results); err != nil {
				return err
			}
		} else {
			facts, err := HashFile(resolvedPath)
			if err != nil {
				return err
			}
			*results = append(*results, FileFacts{
				Path:  relPath,
				Facts: facts,
			})
		}
	}

	return nil
}
