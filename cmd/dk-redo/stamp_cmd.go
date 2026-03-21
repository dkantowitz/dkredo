package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dkantowitz/dk-redo/internal/hasher"
	"github.com/dkantowitz/dk-redo/internal/resolve"
	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdStamp implements the "stamp" subcommand.
// It hashes input files and writes (or appends to) a stamp file.
func cmdStamp(flags Flags, args []string) int {
	// Parse --append flag and extract positional args.
	appendMode := false
	var positional []string
	for _, a := range args {
		if a == "--append" {
			appendMode = true
		} else {
			positional = append(positional, a)
		}
	}

	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "dk-redo stamp: label required")
		return 2
	}

	label := positional[0]
	inputs := positional[1:]

	// Resolve inputs to file paths.
	resolved, err := resolve.Resolve(inputs, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo stamp: resolve: %v\n", err)
		return 2
	}

	// Hash all resolved files.
	fileFacts := make([]hasher.FileFacts, 0, len(resolved))
	for _, path := range resolved {
		facts, err := hasher.HashFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dk-redo stamp: hash %s: %v\n", path, err)
			return 2
		}
		fileFacts = append(fileFacts, hasher.FileFacts{
			Path:  path,
			Facts: facts,
		})
	}

	var s *stamp.Stamp

	if appendMode {
		// Read existing stamp and merge.
		existing, err := stamp.Read(flags.StampsDir, label)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dk-redo stamp: read: %v\n", err)
			return 2
		}
		if existing == nil {
			// No existing stamp — start fresh.
			existing = &stamp.Stamp{Label: label}
		}
		s = stamp.Append(existing, fileFacts)
	} else {
		// Build new stamp.
		files := make([]stamp.FileFact, len(fileFacts))
		for i, ff := range fileFacts {
			files[i] = stamp.FileFact{
				Path:  ff.Path,
				Facts: stamp.FormatFacts(ff.Facts),
			}
		}
		s = &stamp.Stamp{
			Label: label,
			Files: files,
		}
	}

	// Write stamp to disk.
	if err := stamp.Write(flags.StampsDir, s); err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo stamp: write: %v\n", err)
		return 2
	}

	// Verbose output.
	if flags.Verbose {
		stampPath := filepath.Join(flags.StampsDir, stamp.EscapeLabel(label))
		fmt.Printf("stamp: %s\n", stampPath)
		for _, ff := range s.Files {
			fmt.Printf("  %s\t%s\n", ff.Path, ff.Facts)
		}
	}

	return 0
}
