// Package resolve translates raw CLI arguments into a sorted, deduplicated
// list of canonical file paths.
package resolve

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/dkantowitz/dk-redo/internal/hasher"
)

// Resolve takes the raw arguments after <label> and returns a sorted,
// deduplicated list of canonical file paths.
// Arguments can be file paths, directory paths, "-" (read stdin newline-delimited),
// or "-0" (read stdin null-delimited).
// stdin is only read once, at the position of the first "-" or "-0" argument.
func Resolve(args []string, stdin io.Reader) ([]string, error) {
	var paths []string
	stdinConsumed := false

	for _, arg := range args {
		switch {
		case arg == "-" || arg == "-0":
			if stdinConsumed {
				continue
			}
			stdinConsumed = true
			nullTerminated := arg == "-0"
			stdinPaths, err := ReadStdin(stdin, nullTerminated)
			if err != nil {
				return nil, err
			}
			paths = append(paths, stdinPaths...)

		default:
			info, err := os.Stat(arg)
			if err == nil && info.IsDir() {
				// Directory: expand via HashDir
				fileFacts, err := hasher.HashDir(arg)
				if err != nil {
					return nil, err
				}
				for _, ff := range fileFacts {
					paths = append(paths, filepath.Join(arg, ff.Path))
				}
			} else {
				// Regular file (or non-existent — treat as file path)
				paths = append(paths, arg)
			}
		}
	}

	// Canonicalize
	for i, p := range paths {
		paths[i] = filepath.Clean(p)
	}

	// Sort and deduplicate
	sort.Strings(paths)
	paths = dedup(paths)

	return paths, nil
}

// ReadStdin reads file paths from an io.Reader.
// If nullTerminated is true, splits on \0; otherwise splits on \n.
// Empty entries are skipped.
func ReadStdin(r io.Reader, nullTerminated bool) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	var delim byte = '\n'
	if nullTerminated {
		delim = 0
	}

	parts := bytes.Split(data, []byte{delim})
	var result []string
	for _, p := range parts {
		s := string(p)
		if s != "" {
			result = append(result, s)
		}
	}

	return result, nil
}

// dedup removes consecutive duplicates from a sorted slice.
func dedup(sorted []string) []string {
	if len(sorted) == 0 {
		return nil
	}
	result := []string{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1] {
			result = append(result, sorted[i])
		}
	}
	return result
}
