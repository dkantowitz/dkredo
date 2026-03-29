package main

import (
	"fmt"
	"io"
	"os"

	"dkredo/internal/ops"
	"dkredo/internal/stamp"
)

// Execute runs a pipeline of operations on a stamp label.
func Execute(label string, operations []Operation, stampsDir string, verbose bool, stdin io.Reader, stdout io.Writer) int {
	stampsParent := stamp.StampsParent(stampsDir)

	state, err := stamp.ReadStamp(stampsDir, label, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	exitCode := 0
	for _, op := range operations {
		code, err := runOp(op, state, stdin, stampsParent, stdout, verbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 2
		}
		exitCode = code
		if exitCode != 0 {
			break
		}
	}

	if state.Modified {
		if err := stamp.WriteStamp(stampsDir, state, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 2
		}
	}

	return exitCode
}

func runOp(op Operation, state *stamp.StampState, stdin io.Reader, stampsParent string, stdout io.Writer, verbose bool) (int, error) {
	switch op.Name {
	case "add-names":
		return 0, ops.AddNames(state, op.Args, stdin, stampsParent, verbose)
	case "remove-names":
		return 0, ops.RemoveNames(state, op.Args, stdin, stampsParent, verbose)
	case "stamp-facts":
		return 0, ops.StampFacts(state, op.Args, stdin, stampsParent, verbose)
	case "clear-facts":
		return 0, ops.ClearFacts(state, op.Args, stdin, stampsParent, verbose)
	case "check":
		return ops.Check(state, op.Args, stdin, stampsParent, verbose)
	case "check-assert":
		return ops.CheckAssert(state, op.Args, stdin, stampsParent, verbose)
	case "names":
		return 0, ops.Names(state, op.Args, stampsParent, stdout, verbose)
	case "facts":
		return 0, ops.Facts(state, op.Args, stampsParent, stdout, verbose)
	default:
		return 2, fmt.Errorf("unknown operation: +%s", op.Name)
	}
}
