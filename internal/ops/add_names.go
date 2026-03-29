package ops

import (
	"fmt"
	"io"
	"os"

	"dkredo/internal/resolve"
	"dkredo/internal/stamp"
)

// AddNames implements the +add-names operation.
func AddNames(state *stamp.StampState, args []string, stdin io.Reader, stampsParent string, verbose bool) error {
	paths, err := resolve.ResolveFiles(args, stdin, stampsParent)
	if err != nil {
		return err
	}

	added := 0
	for _, p := range paths {
		if state.AddEntry(p, "") {
			added++
		}
	}

	if verbose && added > 0 {
		fmt.Fprintf(os.Stderr, "+add-names: added %d new entries (%d total)\n", added, len(state.Entries))
	}
	return nil
}
