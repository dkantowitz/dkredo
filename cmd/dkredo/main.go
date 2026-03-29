package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dkredo/internal/stamp"
)

var version = "dev"

func main() {
	// Detect argv[0] alias dispatch
	argv0 := filepath.Base(os.Args[0])
	isAlias := argv0 != "dkredo" && strings.HasPrefix(argv0, "dkr-")

	args := os.Args[1:]

	// Prepend DKREDO_ARGS
	if envArgs := os.Getenv("DKREDO_ARGS"); envArgs != "" {
		extra := ShellSplit(envArgs)
		args = append(extra, args...)
	}

	if len(args) == 0 && !isAlias {
		printShortHelp()
		os.Exit(2)
	}

	// Check early-exit flags (before any parsing)
	for i, arg := range args {
		switch arg {
		case "--version":
			fmt.Printf("dkredo %s\n", version)
			os.Exit(0)
		case "--help":
			printFullHelp()
			os.Exit(0)
		case "-h":
			printShortHelp()
			os.Exit(0)
		case "--install":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: --install requires a directory argument\n")
				os.Exit(2)
			}
			if err := Install(args[i+1]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(2)
			}
			os.Exit(0)
		}
	}

	if isAlias {
		aliasName := strings.TrimPrefix(argv0, "dkr-")

		// Parse global flags from front of args
		flags := Flags{}
		i := 0
		for i < len(args) {
			switch args[i] {
			case "-v":
				flags.Verbose = true
				i++
			case "--stamps-dir":
				i++
				if i >= len(args) {
					fmt.Fprintf(os.Stderr, "error: --stamps-dir requires an argument\n")
					os.Exit(2)
				}
				flags.StampsDir = args[i]
				i++
			default:
				goto aliasLabel
			}
		}
	aliasLabel:
		if i >= len(args) {
			fmt.Fprintf(os.Stderr, "error: missing label argument\n")
			os.Exit(2)
		}

		label := args[i]
		i++
		aliasArgs := args[i:]

		expanded, err := ExpandAlias(aliasName, aliasArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		ops, err := parseOps(expanded)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}

		flags.StampsDir = resolveStampsDir(flags.StampsDir)
		exitCode := Execute(label, ops, flags, os.Stdin, os.Stdout)
		os.Exit(exitCode)
	}

	flags, label, operations, err := Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if len(operations) == 0 {
		fmt.Fprintf(os.Stderr, "error: no operations specified\n")
		printShortHelp()
		os.Exit(2)
	}

	flags.StampsDir = resolveStampsDir(flags.StampsDir)
	exitCode := Execute(label, operations, flags, os.Stdin, os.Stdout)
	os.Exit(exitCode)
}

func resolveStampsDir(override string) string {
	if override != "" {
		return override
	}
	found := stamp.FindStampsDir()
	if found != "" {
		return found
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	return filepath.Join(cwd, ".stamps")
}

func printShortHelp() {
	fmt.Fprintf(os.Stderr, `Usage: dkredo [flags] <label> [+operation [args...]]...

Flags:
  -v               Verbose output
  --stamps-dir D   Override .stamps/ directory
  --version        Print version
  --help           Full help
  -h               Short help
  --cmd ALIAS      Expand built-in alias

Operations: +add-names, +remove-names, +stamp-facts, +clear-facts,
            +check, +check-assert, +names, +facts
`)
}

func printFullHelp() {
	fmt.Printf(`dkredo — composable file change detection

Usage: dkredo [flags] <label> [+operation [args...]]...

Operations execute left to right on a single label's stamp file.

FLAGS
  -v                 Verbose output to stderr
  --stamps-dir DIR   Override .stamps/ directory location
  --version          Print version and exit
  --help             Show this help
  -h                 Show short help
  --cmd ALIAS        Expand a built-in alias (ifchange, stamp, always, fnames)
  --install DIR      Install binary and symlinks to DIR

STAMP MANIPULATION
  +add-names [files...]          Add files to stamp's name list
  +remove-names [filters...]     Remove matching entries (empty = all)
  +remove-names -ne [filters...] Remove only if file missing and not expected absent
  +stamp-facts [filters...]      Compute blake3+size facts for entries
  +clear-facts [filters...]      Remove facts, keep names

QUERYING
  +names [filters...]            Print file names from stamp
  +names -e [filters...]         Print only names that exist on disk
  +facts [filters...]            Print path + facts (diagnostic)

VERIFYING
  +check [filters...]            Exit 0=changed, 1=unchanged, 2=error
  +check-assert [filters...]     Like +check but exit 2 when unchanged

INPUT MODES (for file arguments)
  file/path.c                    Literal file path
  -                              Read paths from stdin (newline-terminated)
  -0                             Read paths from stdin (null-terminated)
  -@ file                        Read paths from file
  -@0 file                       Read null-terminated paths from file
  -M file.d                      Parse makefile dep format

FILTER MODES (superset of input modes)
  .suffix                        Match by file extension

EXIT CODES
  0  Changed / action taken
  1  Unchanged (only from +check)
  2  Error

ALIASES (via --cmd or symlinks)
  ifchange [files...]   → +add-names [files...] +check
  stamp [files...]      → +remove-names +add-names [files...] +stamp-facts
  stamp --append [f...] → +add-names [files...] +stamp-facts
  always                → +clear-facts
  fnames [filter...]    → +names -e [filter...]

ENVIRONMENT
  DKREDO_ARGS           Shell-split and prepended to args
`)
}
