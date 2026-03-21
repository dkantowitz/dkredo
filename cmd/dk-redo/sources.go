package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdSources lists all source file paths tracked across all stamps.
// With -v it also shows which labels track each file.
func cmdSources(flags Flags, args []string) int {
	entries, err := os.ReadDir(flags.StampsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "dk-sources: cannot read stamps dir: %v\n", err)
		return 2
	}

	// Collect all file paths; if verbose, also track which labels reference each path.
	pathSet := make(map[string]struct{})
	var pathLabels map[string][]string
	if flags.Verbose {
		pathLabels = make(map[string][]string)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		label := stamp.UnescapeLabel(e.Name())
		s, err := stamp.Read(flags.StampsDir, label)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dk-sources: %v\n", err)
			return 2
		}
		if s == nil {
			continue
		}
		for _, ff := range s.Files {
			pathSet[ff.Path] = struct{}{}
			if flags.Verbose {
				pathLabels[ff.Path] = append(pathLabels[ff.Path], label)
			}
		}
	}

	// Sort the unique paths.
	paths := make([]string, 0, len(pathSet))
	for p := range pathSet {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	// Print results.
	for _, p := range paths {
		if flags.Verbose {
			labels := pathLabels[p]
			sort.Strings(labels)
			fmt.Printf("%s (%s)\n", p, strings.Join(labels, ", "))
		} else {
			fmt.Println(p)
		}
	}

	return 0
}
