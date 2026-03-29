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

// Config holds global CLI configuration.
type Config struct {
	Verbose   bool
	StampsDir string
}

// ValidOps lists all valid operation names.
var ValidOps = map[string]bool{
	"add-names":    true,
	"remove-names": true,
	"stamp-facts":  true,
	"clear-facts":  true,
	"check":        true,
	"check-assert": true,
	"names":        true,
	"facts":        true,
}

// Parse parses CLI args (after argv[0]) into config, label, and operations.
// Handles --cmd alias expansion.
func Parse(args []string) (Config, string, []Operation, error) {
	cfg := Config{}

	// Parse global flags from front
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-v":
			cfg.Verbose = true
			i++
		case "--stamps-dir":
			i++
			if i >= len(args) {
				return cfg, "", nil, fmt.Errorf("--stamps-dir requires an argument")
			}
			cfg.StampsDir = args[i]
			i++
		default:
			goto labelParse
		}
	}

labelParse:
	if i >= len(args) {
		return cfg, "", nil, fmt.Errorf("missing label argument")
	}

	label := args[i]
	i++

	// Check for --cmd
	if i < len(args) && args[i] == "--cmd" {
		i++
		if i >= len(args) {
			return cfg, "", nil, fmt.Errorf("--cmd requires an alias name")
		}
		aliasName := args[i]
		i++

		// Check for multiple --cmd
		for _, a := range args[i:] {
			if a == "--cmd" {
				return cfg, "", nil, fmt.Errorf("multiple --cmd not allowed")
			}
		}

		aliasArgs := args[i:]

		// Check that no +ops are mixed with --cmd
		for _, a := range aliasArgs {
			if strings.HasPrefix(a, "+") {
				return cfg, "", nil, fmt.Errorf("--cmd and +operations cannot be mixed")
			}
		}

		expanded, err := ExpandAlias(aliasName, aliasArgs)
		if err != nil {
			return cfg, "", nil, err
		}

		// Parse expanded ops
		ops, err := parseOps(expanded)
		if err != nil {
			return cfg, "", nil, err
		}
		return cfg, label, ops, nil
	}

	// Parse +operations
	ops, err := parseOps(args[i:])
	if err != nil {
		return cfg, "", nil, err
	}

	return cfg, label, ops, nil
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
