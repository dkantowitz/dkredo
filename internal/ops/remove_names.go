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

// RemoveNames implements the +remove-names operation.
func RemoveNames(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error {
	neMode := false
	filterArgs := args
	if len(args) > 0 && args[0] == "-ne" {
		neMode = true
		filterArgs = args[1:]
	}

	filters, err := resolveEntryFilters(filterArgs, stdin, stampsParent)
	if err != nil {
		return err
	}

	matching := resolve.FilterEntries(state.Entries, filters)

	removed := 0
	for _, m := range matching {
		if neMode {
			fullPath := filepath.Join(stampsParent, m.Path)
			_, statErr := os.Stat(fullPath)
			if statErr == nil {
				continue // file exists, don't remove
			}
			parsed := facts.ParseFacts(m.Facts)
			if parsed["missing"] == "true" {
				continue // intentionally tracked as absent
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "+remove-names -ne: removed %s (file missing, not expected absent)\n", m.Path)
			}
		}
		state.RemoveEntry(m.Path)
		removed++
	}

	if verbose && removed > 0 {
		if neMode {
			fmt.Fprintf(os.Stderr, "+remove-names -ne: removed %d stale entries (%d remaining)\n", removed, len(state.Entries))
		} else {
			fmt.Fprintf(os.Stderr, "+remove-names: removed %d entries (%d remaining)\n", removed, len(state.Entries))
		}
	}
	return nil
}

// resolveEntryFilters resolves filter args for operations that filter existing entries.
// Simple suffix/path args are passed through directly; special modes (-@, -M, etc.)
// are resolved through ResolveFilters.
func resolveEntryFilters(args []string, stdin io.Reader, stampsParent string) ([]string, error) {
	hasSpecial := false
	for _, a := range args {
		if a == "-" || a == "-0" || a == "-@" || a == "-@0" || a == "-M" {
			hasSpecial = true
			break
		}
	}
	if hasSpecial {
		return resolve.ResolveFilters(args, stdin, stampsParent)
	}
	// Simple args — use directly as filters (suffix or exact path match)
	return args, nil
}
