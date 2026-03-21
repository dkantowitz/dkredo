package main

import (
	"testing"
)

func TestResolveCommandSymlinkStyle(t *testing.T) {
	cmd, args, err := resolveCommand([]string{"dk-ifchange", "label", "file.c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "ifchange" {
		t.Errorf("cmd = %q, want %q", cmd, "ifchange")
	}
	if len(args) != 2 || args[0] != "label" || args[1] != "file.c" {
		t.Errorf("args = %v, want [label file.c]", args)
	}
}

func TestResolveCommandSubcommandStyle(t *testing.T) {
	cmd, args, err := resolveCommand([]string{"dk-redo", "ifchange", "label", "file.c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "ifchange" {
		t.Errorf("cmd = %q, want %q", cmd, "ifchange")
	}
	if len(args) != 2 || args[0] != "label" || args[1] != "file.c" {
		t.Errorf("args = %v, want [label file.c]", args)
	}
}

func TestResolveCommandNoSubcommand(t *testing.T) {
	_, _, err := resolveCommand([]string{"dk-redo"})
	if err == nil {
		t.Fatal("expected error for dk-redo with no subcommand, got nil")
	}
}

func TestResolveCommandInstallViaSubcommand(t *testing.T) {
	cmd, args, err := resolveCommand([]string{"dk-redo", "install", "/usr/local/bin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "install" {
		t.Errorf("cmd = %q, want %q", cmd, "install")
	}
	if len(args) != 1 || args[0] != "/usr/local/bin" {
		t.Errorf("args = %v, want [/usr/local/bin]", args)
	}
}

func TestResolveCommandInstallViaSymlink(t *testing.T) {
	_, _, err := resolveCommand([]string{"dk-install"})
	if err == nil {
		t.Fatal("expected error for dk-install symlink, got nil")
	}
}

func TestResolveCommandOtherSymlinks(t *testing.T) {
	tests := []struct {
		argv0 string
		want  string
	}{
		{"dk-stamp", "stamp"},
		{"dk-always", "always"},
		{"dk-ood", "ood"},
		{"dk-affects", "affects"},
		{"dk-sources", "sources"},
		{"dk-dot", "dot"},
	}
	for _, tt := range tests {
		t.Run(tt.argv0, func(t *testing.T) {
			cmd, _, err := resolveCommand([]string{tt.argv0, "arg1"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != tt.want {
				t.Errorf("cmd = %q, want %q", cmd, tt.want)
			}
		})
	}
}

func TestResolveCommandWithPath(t *testing.T) {
	// argv[0] may include a full path
	cmd, args, err := resolveCommand([]string{"/usr/local/bin/dk-ifchange", "target"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "ifchange" {
		t.Errorf("cmd = %q, want %q", cmd, "ifchange")
	}
	if len(args) != 1 || args[0] != "target" {
		t.Errorf("args = %v, want [target]", args)
	}
}

func TestParseFlagsVerbose(t *testing.T) {
	flags, remaining, exit := parseFlags([]string{"-v", "file1", "file2"})
	if !flags.Verbose {
		t.Error("expected Verbose=true")
	}
	if flags.Quiet {
		t.Error("expected Quiet=false")
	}
	if exit != "" {
		t.Errorf("expected no early exit, got %q", exit)
	}
	if len(remaining) != 2 {
		t.Errorf("remaining = %v, want [file1 file2]", remaining)
	}
}

func TestParseFlagsQuiet(t *testing.T) {
	flags, _, _ := parseFlags([]string{"-q"})
	if !flags.Quiet {
		t.Error("expected Quiet=true")
	}
}

func TestParseFlagsStampsDir(t *testing.T) {
	flags, _, _ := parseFlags([]string{"--stamps-dir", "/tmp/stamps"})
	if flags.StampsDir != "/tmp/stamps" {
		t.Errorf("StampsDir = %q, want %q", flags.StampsDir, "/tmp/stamps")
	}
}

func TestParseFlagsHelp(t *testing.T) {
	_, _, exit := parseFlags([]string{"--help"})
	if exit != "help" {
		t.Errorf("expected early exit 'help', got %q", exit)
	}
}

func TestParseFlagsVersion(t *testing.T) {
	_, _, exit := parseFlags([]string{"--version"})
	if exit != "version" {
		t.Errorf("expected early exit 'version', got %q", exit)
	}
}

func TestParseFlagsDefault(t *testing.T) {
	flags, _, _ := parseFlags([]string{})
	if flags.StampsDir != ".stamps/" {
		t.Errorf("StampsDir = %q, want %q", flags.StampsDir, ".stamps/")
	}
}

func TestDispatchUnknownCommand(t *testing.T) {
	code := dispatch("nonexistent", Flags{}, nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestDispatchKnownCommands(t *testing.T) {
	for _, cmd := range []string{"ifchange", "always", "affects", "sources"} {
		t.Run(cmd, func(t *testing.T) {
			code := dispatch(cmd, Flags{}, nil)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 for stub command %q", code, cmd)
			}
		})
	}
}
