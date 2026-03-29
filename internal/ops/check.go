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

// Check implements the +check operation.
// Returns exit code: 0=changed, 1=unchanged, 2=error.
func Check(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) (int, error) {
	filters, err := resolveEntryFilters(args, stdin, stampsParent)
	if err != nil {
		return 2, err
	}

	matching := resolve.FilterEntries(state.Entries, filters)

	if len(matching) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "+check: unchanged (no entries to check)\n")
		}
		return 1, nil
	}

	for _, m := range matching {
		fullPath := filepath.Join(stampsParent, m.Path)
		changed, reason, err := facts.CheckFact(fullPath, m.Facts)
		if err != nil {
			return 2, fmt.Errorf("+check: %w", err)
		}
		if changed {
			if verbose {
				fmt.Fprintf(os.Stderr, "+check: changed (%s: %s)\n", m.Path, reason)
			}
			return 0, nil
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "+check: unchanged (%d files, all facts match)\n", len(matching))
	}
	return 1, nil
}

// CheckAssert implements the +check-assert operation.
// Same as Check but exit 2 instead of 1 when unchanged.
func CheckAssert(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) (int, error) {
	code, err := Check(state, args, stdin, stampsParent, verbose)
	if code == 1 {
		return 2, nil
	}
	return code, err
}
