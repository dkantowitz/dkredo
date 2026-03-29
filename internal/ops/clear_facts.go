package ops

import (
	"fmt"
	"io"
	"os"

	"dkredo/internal/resolve"
	"dkredo/internal/stamp"
)

// ClearFacts implements the +clear-facts operation.
func ClearFacts(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error {
	filters, err := resolveEntryFilters(args, stdin, stampsParent)
	if err != nil {
		return err
	}

	matching := resolve.FilterEntries(state.Entries, filters)

	count := 0
	for _, m := range matching {
		e := state.FindEntry(m.Path)
		if e != nil && e.Facts != "" {
			e.Facts = ""
			state.Modified = true
			count++
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "+clear-facts: cleared facts for %d entries\n", count)
	}
	return nil
}
