package ops

import (
	"fmt"
	"io"

	"dkredo/internal/resolve"
	"dkredo/internal/stamp"
)

// Facts implements the +facts operation.
func Facts(state *stamp.StampState, args []string, stampsParent string, stdout io.Writer, verbose bool) error {
	entries := resolve.FilterEntries(state.Entries, args)

	for _, e := range entries {
		fmt.Fprintf(stdout, "%s\t%s\n", e.Path, e.Facts)
	}
	return nil
}
