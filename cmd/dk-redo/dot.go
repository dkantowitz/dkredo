package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdDot outputs the dependency graph in DOT format.
func cmdDot(flags Flags, args []string) int {
	// Parse --lr flag and collect label arguments.
	lr := false
	var labels []string
	for _, a := range args {
		if a == "--lr" {
			lr = true
		} else {
			labels = append(labels, a)
		}
	}

	var stamps []*stamp.Stamp

	if len(labels) > 0 {
		// Read specified stamps.
		for _, label := range labels {
			s, err := stamp.Read(flags.StampsDir, label)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dk-dot: %v\n", err)
				return 2
			}
			if s == nil {
				fmt.Fprintf(os.Stderr, "dk-dot: stamp %q not found\n", label)
				return 2
			}
			stamps = append(stamps, s)
		}
	} else {
		// Read all stamps from the stamps directory.
		entries, err := os.ReadDir(flags.StampsDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(os.Stderr, "dk-dot: no stamps found")
				return 2
			}
			fmt.Fprintf(os.Stderr, "dk-dot: cannot read stamps dir: %v\n", err)
			return 2
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			label := stamp.UnescapeLabel(e.Name())
			s, err := stamp.Read(flags.StampsDir, label)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dk-dot: %v\n", err)
				return 2
			}
			if s != nil {
				stamps = append(stamps, s)
			}
		}
	}

	if len(stamps) == 0 {
		fmt.Fprintln(os.Stderr, "dk-dot: no stamps found")
		return 2
	}

	// Emit DOT graph.
	rankdir := "TB"
	if lr {
		rankdir = "LR"
	}

	fmt.Println("digraph deps {")
	fmt.Printf("    rankdir=%s;\n", rankdir)
	for _, s := range stamps {
		for _, f := range s.Files {
			fmt.Printf("    %q -> %q;\n", s.Label, f.Path)
		}
	}
	fmt.Println("}")

	return 0
}
