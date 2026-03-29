package ops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"dkredo/internal/facts"
	"dkredo/internal/resolve"
	"dkredo/internal/stamp"
)

// StampFacts implements the +stamp-facts operation.
func StampFacts(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error {
	filters, err := resolveEntryFilters(args, stdin, stampsParent)
	if err != nil {
		return err
	}

	matching := resolve.FilterEntries(state.Entries, filters)

	count := 0
	for _, m := range matching {
		e := state.FindEntry(m.Path)
		if e == nil {
			continue
		}
		fullPath := filepath.Join(stampsParent, e.Path)
		factsStr, err := facts.FileFacts(fullPath)
		if err != nil {
			return fmt.Errorf("+stamp-facts: %w", err)
		}
		e.Facts = factsStr
		state.Modified = true
		count++

		if verbose {
			fmt.Fprintf(os.Stderr, "  %s %s\n", e.Path, factsStr)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "+stamp-facts: computed facts for %d files\n", count)
	}
	return nil
}
