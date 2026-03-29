package main

import (
	"fmt"
	"strings"
)

// Operation represents a parsed +operation with its args.
type Operation struct {
	Name string
	Args []string
}

// Flags holds all flags that can appear globally or per-operation.
// Global flags are set before the label. Operation-scoped flags appear
// after a +operation and override the global defaults for that operation only.
type Flags struct {
	Verbose      bool
	StampsDir    string
	StampsParent string // resolved by executor, read-only for operations
}

// ValidOps lists all valid operation names.
var ValidOps = map[string]bool{
	"add-names":    true,
	"remove-names": true,
	"stamp-facts":  true,
	"clear-facts":  true,
	"check":        true,
	"check-assert": true,
	"check-all":    true,
	"names":        true,
	"facts":        true,
}

// ExtractFlags removes recognized flags from args, applies them to flags,
// and returns the remaining args.
func ExtractFlags(flags *Flags, args []string) []string {
	var remaining []string
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-v":
			flags.Verbose = true
			i++
		case "--stamps-dir":
			i++
			if i < len(args) {
				flags.StampsDir = args[i]
				i++
			}
		default:
			remaining = append(remaining, args[i])
			i++
		}
	}
	return remaining
}

// Parse parses CLI args (after argv[0]) into flags, label, and operations.
// Handles --cmd alias expansion.
func Parse(args []string) (Flags, string, []Operation, error) {
	flags := Flags{}

	// Parse global flags from front
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-v":
			flags.Verbose = true
			i++
		case "--stamps-dir":
			i++
			if i >= len(args) {
				return flags, "", nil, fmt.Errorf("--stamps-dir requires an argument")
			}
			flags.StampsDir = args[i]
			i++
		default:
			goto labelParse
		}
	}

labelParse:
	if i >= len(args) {
		return flags, "", nil, fmt.Errorf("missing label argument")
	}

	label := args[i]
	if strings.HasPrefix(label, "+") {
		return flags, "", nil, fmt.Errorf(
			"missing label — first argument %q looks like an operation, not a label\n"+
				"usage: dkredo <label> [+operation [args...]]...", label)
	}
	i++

	// Check for --cmd
	if i < len(args) && args[i] == "--cmd" {
		i++
		if i >= len(args) {
			return flags, "", nil, fmt.Errorf("--cmd requires an alias name")
		}
		aliasName := args[i]
		i++

		// Check for multiple --cmd
		for _, a := range args[i:] {
			if a == "--cmd" {
				return flags, "", nil, fmt.Errorf("multiple --cmd not allowed")
			}
		}

		aliasArgs := args[i:]

		// Check that no +ops are mixed with --cmd
		for _, a := range aliasArgs {
			if strings.HasPrefix(a, "+") {
				return flags, "", nil, fmt.Errorf("--cmd and +operations cannot be mixed")
			}
		}

		expanded, err := ExpandAlias(aliasName, aliasArgs)
		if err != nil {
			return flags, "", nil, err
		}

		// Parse expanded ops
		ops, err := parseOps(expanded)
		if err != nil {
			return flags, "", nil, err
		}
		return flags, label, ops, nil
	}

	// Parse +operations
	ops, err := parseOps(args[i:])
	if err != nil {
		return flags, "", nil, err
	}

	return flags, label, ops, nil
}

func parseOps(args []string) ([]Operation, error) {
	var ops []Operation
	i := 0
	for i < len(args) {
		arg := args[i]
		if !strings.HasPrefix(arg, "+") {
			return nil, fmt.Errorf("expected +operation, got %q", arg)
		}

		opName := arg[1:]
		if !ValidOps[opName] {
			return nil, fmt.Errorf("unknown operation +%s", opName)
		}

		i++

		var opArgs []string
		for i < len(args) && !strings.HasPrefix(args[i], "+") {
			opArgs = append(opArgs, args[i])
			i++
		}

		ops = append(ops, Operation{Name: opName, Args: opArgs})
	}
	return ops, nil
}
