package main

import (
	"fmt"
	"os"

	"github.com/dkantowitz/dk-redo/internal/hasher"
	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdOod lists out-of-date labels. Exit codes:
//
//	0 = at least one label is out of date
//	1 = all labels are up to date
//	2 = error (no stamps, I/O error)
func cmdOod(flags Flags, args []string) int {
	labels, err := resolveLabels(flags, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-ood: %v\n", err)
		return 2
	}
	if len(labels) == 0 {
		fmt.Fprintln(os.Stderr, "dk-ood: no stamps found")
		return 2
	}

	anyOutOfDate := false

	for _, label := range labels {
		s, err := stamp.Read(flags.StampsDir, label)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dk-ood: %v\n", err)
			return 2
		}
		if s == nil {
			fmt.Fprintf(os.Stderr, "dk-ood: stamp not found for label %q\n", label)
			return 2
		}

		currentFacts, err := rehashFiles(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dk-ood: %v\n", err)
			return 2
		}

		result := stamp.Compare(s, currentFacts)

		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "dk-ood: warning: %s\n", w)
		}

		if result.Changed {
			anyOutOfDate = true
			if !flags.Quiet {
				fmt.Println(label)
			}
			if flags.Verbose {
				for _, cf := range result.ChangedFiles {
					fmt.Printf("  %s: %s\n", cf.Path, cf.Reason)
				}
			}
		}
	}

	if anyOutOfDate {
		return 0
	}
	return 1
}

// resolveLabels determines which labels to check. If args are provided, those
// are the labels. Otherwise, scan the stamps directory for all labels.
func resolveLabels(flags Flags, args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	entries, err := os.ReadDir(flags.StampsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading stamps directory: %w", err)
	}

	var labels []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden/temp files
		if len(name) > 0 && name[0] == '.' {
			continue
		}
		labels = append(labels, stamp.UnescapeLabel(name))
	}
	return labels, nil
}

// rehashFiles re-hashes each file referenced in the stamp and returns the
// current facts.
func rehashFiles(s *stamp.Stamp) ([]hasher.FileFacts, error) {
	facts := make([]hasher.FileFacts, 0, len(s.Files))
	for _, ff := range s.Files {
		f, err := hasher.HashFile(ff.Path)
		if err != nil {
			return nil, fmt.Errorf("hashing %s: %w", ff.Path, err)
		}
		facts = append(facts, hasher.FileFacts{
			Path:  ff.Path,
			Facts: f,
		})
	}
	return facts, nil
}
