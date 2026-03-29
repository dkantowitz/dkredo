package ops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"dkredo/internal/resolve"
	"dkredo/internal/stamp"
)

// Names implements the +names operation.
func Names(state *stamp.StampState, args []string, stampsParent string, stdout io.Writer, verbose bool) error {
	existsOnly := false
	filterArgs := args
	if len(args) > 0 && args[0] == "-e" {
		existsOnly = true
		filterArgs = args[1:]
	}

	entries := resolve.FilterEntries(state.Entries, filterArgs)

	for _, e := range entries {
		if existsOnly {
			fullPath := filepath.Join(stampsParent, e.Path)
			if _, err := os.Stat(fullPath); err != nil {
				continue
			}
		}
		fmt.Fprintf(stdout, "%s\n", e.Path)
	}
	return nil
}
