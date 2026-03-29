package main

import (
	"fmt"
	"io"
	"os"

	"dkredo/internal/ops"
	"dkredo/internal/stamp"
)

// Execute runs a pipeline of operations on a stamp label.
func Execute(label string, operations []Operation, globalFlags Flags, stdin io.Reader, stdout io.Writer) int {
	globalFlags.StampsParent = stamp.StampsParent(globalFlags.StampsDir)

	state, err := stamp.ReadStamp(globalFlags.StampsDir, label, globalFlags.Verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	exitCode := 0
	for _, op := range operations {
		opFlags := globalFlags // value copy — per-op flags don't leak
		op.Args = ExtractFlags(&opFlags, op.Args)
		opFlags.StampsParent = stamp.StampsParent(opFlags.StampsDir)
		code, err := runOp(op, state, stdin, opFlags.StampsParent, stdout, opFlags.Verbose)
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
		if err := stamp.WriteStamp(globalFlags.StampsDir, state, globalFlags.Verbose); err != nil {
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
	case "check-all":
		return ops.CheckAll(state, op.Args, stdin, stampsParent, verbose)
	case "names":
		return 0, ops.Names(state, op.Args, stampsParent, stdout, verbose)
	case "facts":
		return 0, ops.Facts(state, op.Args, stampsParent, stdout, verbose)
	default:
		return 2, fmt.Errorf("unknown operation: +%s", op.Name)
	}
}
