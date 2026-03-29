package resolve

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ResolveFiles resolves raw args into canonical, deduplicated file paths.
// stampsParent is the directory that paths should be relative to.
func ResolveFiles(args []string, stdin io.Reader, stampsParent string) ([]string, error) {
	var items []string

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-":
			paths, err := readLines(stdin)
			if err != nil {
				return nil, fmt.Errorf("resolve: reading stdin: %w", err)
			}
			items = append(items, paths...)
		case "-0":
			paths, err := readNullTerminated(stdin)
			if err != nil {
				return nil, fmt.Errorf("resolve: reading stdin (null): %w", err)
			}
			items = append(items, paths...)
		case "-@":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("resolve: -@ requires a file argument")
			}
			paths, err := readLinesFromFile(args[i])
			if err != nil {
				return nil, err
			}
			items = append(items, paths...)
		case "-@0":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("resolve: -@0 requires a file argument")
			}
			paths, err := readNullTerminatedFromFile(args[i])
			if err != nil {
				return nil, err
			}
			items = append(items, paths...)
		case "-M":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("resolve: -M requires a file argument")
			}
			paths, err := ParseDepfile(args[i])
			if err != nil {
				return nil, err
			}
			items = append(items, paths...)
		default:
			items = append(items, arg)
		}
		i++
	}

	return canonicalize(items, stampsParent)
}

// ResolveFilters resolves filter args. Filters can include all input modes
// plus suffix patterns (.c, .h). Suffix patterns are returned as-is.
func ResolveFilters(args []string, stdin io.Reader, stampsParent string) ([]string, error) {
	var items []string

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-":
			paths, err := readLines(stdin)
			if err != nil {
				return nil, fmt.Errorf("resolve: reading stdin: %w", err)
			}
			items = append(items, paths...)
		case "-0":
			paths, err := readNullTerminated(stdin)
			if err != nil {
				return nil, fmt.Errorf("resolve: reading stdin (null): %w", err)
			}
			items = append(items, paths...)
		case "-@":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("resolve: -@ requires a file argument")
			}
			paths, err := readLinesFromFile(args[i])
			if err != nil {
				return nil, err
			}
			items = append(items, paths...)
		case "-@0":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("resolve: -@0 requires a file argument")
			}
			paths, err := readNullTerminatedFromFile(args[i])
			if err != nil {
				return nil, err
			}
			items = append(items, paths...)
		case "-M":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("resolve: -M requires a file argument")
			}
			paths, err := ParseDepfile(args[i])
			if err != nil {
				return nil, err
			}
			items = append(items, paths...)
		default:
			// Suffix filters start with '.' and don't contain path separators
			if isSuffixFilter(arg) {
				items = append(items, arg)
			} else {
				items = append(items, arg)
			}
		}
		i++
	}

	// Canonicalize non-suffix items
	var result []string
	var nonSuffix []string
	suffixIndices := make(map[int]bool)

	for idx, item := range items {
		if isSuffixFilter(item) {
			suffixIndices[idx] = true
		} else {
			nonSuffix = append(nonSuffix, item)
		}
	}

	canonPaths, err := canonicalize(nonSuffix, stampsParent)
	if err != nil {
		return nil, err
	}

	// Rebuild in order: suffixes stay as-is, paths are canonicalized
	canonIdx := 0
	for idx, item := range items {
		if suffixIndices[idx] {
			result = append(result, item)
		} else {
			if canonIdx < len(canonPaths) {
				result = append(result, canonPaths[canonIdx])
				canonIdx++
			}
		}
	}

	return dedup(result), nil
}

// MatchesFilter returns true if path matches the filter.
// Filter can be an exact path or a suffix pattern (.c, .h).
func MatchesFilter(path string, filter string) bool {
	if isSuffixFilter(filter) {
		return filepath.Ext(path) == filter
	}
	return path == filter
}

// isSuffixFilter returns true if the arg is a suffix filter (starts with dot, no path sep).
func isSuffixFilter(arg string) bool {
	if len(arg) < 2 || arg[0] != '.' {
		return false
	}
	for _, c := range arg[1:] {
		if c == '/' || c == filepath.Separator {
			return false
		}
	}
	return true
}

func canonicalize(paths []string, stampsParent string) ([]string, error) {
	var result []string
	for _, p := range paths {
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("resolve: abs %s: %w", p, err)
		}
		rel, err := filepath.Rel(stampsParent, abs)
		if err != nil {
			return nil, fmt.Errorf("resolve: rel %s: %w", p, err)
		}
		// Normalize to forward slashes
		rel = filepath.ToSlash(rel)
		result = append(result, rel)
	}
	sort.Strings(result)
	return dedup(result), nil
}

func dedup(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func readLines(r io.Reader) ([]string, error) {
	var paths []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths, scanner.Err()
}

func readNullTerminated(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, part := range bytes.Split(data, []byte{0}) {
		s := string(part)
		if s != "" {
			paths = append(paths, s)
		}
	}
	return paths, nil
}

func readLinesFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("resolve: open -@ %s: %w", path, err)
	}
	defer f.Close()
	return readLines(f)
}

func readNullTerminatedFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("resolve: open -@0 %s: %w", path, err)
	}
	defer f.Close()
	return readNullTerminated(f)
}
