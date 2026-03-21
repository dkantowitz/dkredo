package main

import (
	"fmt"
	"os"

	"github.com/dkantowitz/dk-redo/internal/hasher"
	"github.com/dkantowitz/dk-redo/internal/resolve"
	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdIfchange implements the ifchange command.
// It checks whether the input files have changed since the last stamp.
//
// Exit codes:
//
//	0 = changed (or first run, or -n flag) — recipe continues
//	1 = unchanged — ? sigil stops recipe
//	2 = error
func cmdIfchange(flags Flags, args []string) int {
	// Parse -n flag (ifchange-specific: force changed).
	forceChanged := false
	var positional []string
	for _, arg := range args {
		if arg == "-n" {
			forceChanged = true
		} else {
			positional = append(positional, arg)
		}
	}

	// Extract label and inputs.
	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "dk-ifchange: label argument required")
		return 2
	}
	label := positional[0]
	inputs := positional[1:]

	// If -n (force changed): exit 0 immediately.
	if forceChanged {
		return 0
	}

	// Resolve inputs (expand stdin markers, etc.).
	resolved, err := resolve.Resolve(inputs, os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-ifchange %s: resolve: %v\n", label, err)
		return 2
	}

	// Hash all resolved files.
	currentFacts := make([]hasher.FileFacts, 0, len(resolved))
	for _, path := range resolved {
		facts, err := hasher.HashFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dk-ifchange %s: hash %s: %v\n", label, path, err)
			return 2
		}
		currentFacts = append(currentFacts, hasher.FileFacts{
			Path:  path,
			Facts: facts,
		})
	}

	// Read existing stamp.
	existingStamp, err := stamp.Read(flags.StampsDir, label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-ifchange %s: read stamp: %v\n", label, err)
		return 2
	}

	// No stamp exists: first run — treat as changed.
	if existingStamp == nil {
		return 0
	}

	// Compare stamp against current facts.
	result := stamp.Compare(existingStamp, currentFacts)

	// Print warnings to stderr.
	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "dk-ifchange %s: warning: %s\n", label, w)
	}

	// Verbose: print changed files and reasons.
	if flags.Verbose && result.Changed {
		for _, cf := range result.ChangedFiles {
			fmt.Fprintf(os.Stderr, "dk-ifchange %s: changed: %s (%s)\n", label, cf.Path, cf.Reason)
		}
	}

	if result.Changed {
		return 0
	}

	// Unchanged.
	if !flags.Quiet {
		fmt.Fprintf(os.Stderr, "dk-ifchange %s: up to date\n", label)
	}
	return 1
}
