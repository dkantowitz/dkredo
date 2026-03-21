package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dkantowitz/dk-redo/internal/stamp"
)

// cmdAlways removes stamp files so that the corresponding targets are
// considered always out-of-date. With --all it removes every stamp file.
func cmdAlways(flags Flags, args []string) int {
	all := false
	var labels []string
	for _, a := range args {
		if a == "--all" {
			all = true
		} else {
			labels = append(labels, a)
		}
	}

	if all {
		entries, err := os.ReadDir(flags.StampsDir)
		if err != nil {
			// If the directory doesn't exist, there's nothing to remove.
			if errors.Is(err, os.ErrNotExist) {
				return 0
			}
			fmt.Fprintf(os.Stderr, "dk-always: cannot read stamps dir: %v\n", err)
			return 0
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			p := filepath.Join(flags.StampsDir, e.Name())
			if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
				// Ignore; always exit 0.
			}
			if flags.Verbose {
				fmt.Println(p)
			}
		}
		return 0
	}

	for _, label := range labels {
		p := filepath.Join(flags.StampsDir, stamp.EscapeLabel(label))
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			// Ignore; always exit 0.
		} else if err == nil && flags.Verbose {
			fmt.Println(p)
		}
	}
	return 0
}
