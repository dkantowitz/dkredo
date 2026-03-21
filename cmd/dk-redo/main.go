package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// version is injected at build time via -ldflags.
var version = "dev"

// allCommands lists every command that dk-redo can dispatch to.
var allCommands = []string{
	"ifchange", "stamp", "always", "install", "ood", "affects", "sources", "dot",
}

// symlinkCommands lists commands that are valid via argv[0] symlink dispatch.
// "install" is intentionally excluded — it is only reachable via subcommand.
var symlinkCommands = []string{
	"ifchange", "stamp", "always", "ood", "affects", "sources", "dot",
}

// Flags holds the shared CLI flags.
type Flags struct {
	Verbose   bool
	Quiet     bool
	StampsDir string
}

func main() {
	cmd, args, err := resolveCommand(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(2)
	}

	flags, remaining, earlyExit := parseFlags(args)
	if earlyExit != "" {
		switch earlyExit {
		case "help":
			usage()
			os.Exit(0)
		case "version":
			fmt.Printf("dk-redo %s\n", version)
			os.Exit(0)
		}
	}

	os.Exit(dispatch(cmd, flags, remaining))
}

// resolveCommand determines the subcommand and remaining args from argv.
// It returns an error instead of calling os.Exit so it remains testable.
func resolveCommand(argv []string) (string, []string, error) {
	if len(argv) < 1 {
		return "", nil, fmt.Errorf("no arguments provided")
	}

	cmd := filepath.Base(argv[0])
	if strings.HasPrefix(cmd, "dk-") {
		cmd = strings.TrimPrefix(cmd, "dk-")
	}

	args := argv[1:]

	if cmd == "redo" {
		if len(args) < 1 {
			return "", nil, fmt.Errorf("dk-redo: subcommand required")
		}
		// Allow --help and --version before subcommand
		if args[0] == "--help" || args[0] == "--version" {
			return "redo", args, nil
		}
		cmd = args[0]
		args = args[1:]
	}

	// "install" is only allowed via the subcommand style (dk-redo install),
	// not via an argv[0] symlink (dk-install).
	if cmd == "install" && filepath.Base(argv[0]) != "dk-redo" {
		return "", nil, fmt.Errorf("install command is only available via 'dk-redo install'")
	}

	return cmd, args, nil
}

// parseFlags extracts shared flags from args and returns the Flags struct,
// the remaining positional arguments, and an optional early-exit directive
// ("help", "version", or "").
func parseFlags(args []string) (Flags, []string, string) {
	f := Flags{
		StampsDir: ".stamps/",
	}
	var remaining []string
	earlyExit := ""

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-v", "--verbose":
			f.Verbose = true
		case "-q", "--quiet":
			f.Quiet = true
		case "--stamps-dir":
			i++
			if i < len(args) {
				f.StampsDir = args[i]
			}
		case "--help":
			earlyExit = "help"
		case "--version":
			earlyExit = "version"
		default:
			remaining = append(remaining, args[i])
		}
		i++
	}

	return f, remaining, earlyExit
}

// dispatch calls the appropriate command function and returns an exit code.
func dispatch(cmd string, flags Flags, args []string) int {
	switch cmd {
	case "ifchange":
		return cmdStub("ifchange", flags, args)
	case "stamp":
		return cmdStub("stamp", flags, args)
	case "always":
		return cmdAlways(flags, args)
	case "install":
		return cmdInstall(flags, args)
	case "ood":
		return cmdStub("ood", flags, args)
	case "affects":
		return cmdStub("affects", flags, args)
	case "sources":
		return cmdStub("sources", flags, args)
	case "dot":
		return cmdStub("dot", flags, args)
	default:
		fmt.Fprintf(os.Stderr, "dk-redo: unknown command %q\n", cmd)
		usage()
		return 2
	}
}

// cmdStub is a placeholder for commands not yet implemented.
func cmdStub(name string, _ Flags, _ []string) int {
	fmt.Printf("%s: not yet implemented\n", name)
	return 0
}

// cmdInstall copies the running binary to destDir and creates symlinks for
// every symlink-dispatchable command.
func cmdInstall(flags Flags, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: dk-redo install <dest-dir>")
		return 2
	}
	destDir := args[0]

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo install: cannot determine executable path: %v\n", err)
		return 1
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo install: cannot resolve executable path: %v\n", err)
		return 1
	}

	destBin := filepath.Join(destDir, "dk-redo")

	// Copy binary to destination.
	data, err := os.ReadFile(self)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo install: cannot read binary: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo install: cannot create directory %s: %v\n", destDir, err)
		return 1
	}
	if err := os.WriteFile(destBin, data, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "dk-redo install: cannot write binary to %s: %v\n", destBin, err)
		return 1
	}

	if flags.Verbose {
		fmt.Printf("installed %s\n", destBin)
	}

	// Create symlinks for all symlink-dispatchable commands.
	for _, cmd := range symlinkCommands {
		link := filepath.Join(destDir, "dk-"+cmd)
		// Remove existing file/symlink if present.
		os.Remove(link)
		if err := os.Symlink("dk-redo", link); err != nil {
			fmt.Fprintf(os.Stderr, "dk-redo install: cannot create symlink %s: %v\n", link, err)
			return 1
		}
		if flags.Verbose {
			fmt.Printf("symlink %s -> dk-redo\n", link)
		}
	}

	fmt.Printf("installed dk-redo and %d symlinks to %s\n", len(symlinkCommands), destDir)
	return 0
}

// usage prints a help message to stderr.
func usage() {
	fmt.Fprintf(os.Stderr, `Usage: dk-redo <command> [flags] [args...]

Commands:
  ifchange    Mark targets as dependencies (rebuild if changed)
  stamp       Mark a target as stamp-based (content hash)
  always      Mark a target to always rebuild
  ood         List out-of-date targets
  affects     List targets affected by a source change
  sources     List source dependencies of targets
  dot         Output dependency graph in DOT format
  install     Install dk-redo and create symlinks (subcommand only)

Flags:
  -v, --verbose       Enable verbose output
  -q, --quiet         Suppress non-error output
  --stamps-dir PATH   Set stamps directory (default: .stamps/)
  --help              Show this help message
  --version           Show version

Symlink invocation:
  dk-ifchange file.c   (equivalent to: dk-redo ifchange file.c)

Version: %s
`, version)
}
