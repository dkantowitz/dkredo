package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dkantowitz/dk-redo/internal/resolve"
	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdAffects lists stamp labels that depend on any of the given files.
// Exit 0 if at least one label is found, 1 if none.
func cmdAffects(flags Flags, args []string) int {
	// 1. Resolve query files from args.
	queryFiles, err := resolve.Resolve(args, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "affects: %v\n", err)
		return 2
	}

	if len(queryFiles) == 0 {
		fmt.Fprintln(os.Stderr, "affects: no files specified")
		return 2
	}

	// 2. Build a set of queried paths for fast lookup.
	querySet := make(map[string]bool, len(queryFiles))
	for _, p := range queryFiles {
		querySet[p] = true
	}

	// 3. Read all stamp files from flags.StampsDir.
	entries, err := os.ReadDir(flags.StampsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No stamps directory means no labels.
			return 1
		}
		fmt.Fprintf(os.Stderr, "affects: cannot read stamps dir: %v\n", err)
		return 2
	}

	// 4. For each stamp, check if any queried file appears in its Files list.
	found := false
	// Collect results sorted by label for deterministic output.
	type match struct {
		label   string
		matched []string
	}
	var matches []match

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		label := stamp.UnescapeLabel(entry.Name())
		s, err := stamp.Read(flags.StampsDir, label)
		if err != nil {
			if !flags.Quiet {
				fmt.Fprintf(os.Stderr, "affects: warning: %v\n", err)
			}
			continue
		}
		if s == nil {
			continue
		}

		var matched []string
		for _, ff := range s.Files {
			if querySet[ff.Path] {
				matched = append(matched, ff.Path)
			}
		}

		if len(matched) > 0 {
			matches = append(matches, match{label: label, matched: matched})
		}
	}

	// Sort matches by label for deterministic output.
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].label < matches[j].label
	})

	// 5. Print results.
	for _, m := range matches {
		found = true
		if flags.Verbose {
			sort.Strings(m.matched)
			fmt.Printf("%s\t%s\n", m.label, strings.Join(m.matched, " "))
		} else {
			fmt.Println(m.label)
		}
	}

	// 6. Exit code based on whether any labels were found.
	if !found {
		return 1
	}
	return 0
}
