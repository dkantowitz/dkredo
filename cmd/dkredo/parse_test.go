package main

import (
	"strings"
	"testing"
)

func TestParseLabel(t *testing.T) {
	_, label, ops, err := Parse([]string{"my-label", "+add-names", "a.c", "b.c"})
	if err != nil {
		t.Fatal(err)
	}
	if label != "my-label" {
		t.Fatalf("label = %q", label)
	}
	if len(ops) != 1 {
		t.Fatalf("ops = %d", len(ops))
	}
	if ops[0].Name != "add-names" {
		t.Fatalf("op name = %q", ops[0].Name)
	}
	if len(ops[0].Args) != 2 || ops[0].Args[0] != "a.c" || ops[0].Args[1] != "b.c" {
		t.Fatalf("op args = %v", ops[0].Args)
	}
}

func TestParseMultipleOps(t *testing.T) {
	_, label, ops, err := Parse([]string{"label", "+add-names", "a.c", "+check"})
	if err != nil {
		t.Fatal(err)
	}
	if label != "label" {
		t.Fatalf("label = %q", label)
	}
	if len(ops) != 2 {
		t.Fatalf("ops = %d", len(ops))
	}
	if ops[0].Name != "add-names" || ops[1].Name != "check" {
		t.Fatalf("ops = %v %v", ops[0].Name, ops[1].Name)
	}
	if len(ops[1].Args) != 0 {
		t.Fatalf("check should have no args, got %v", ops[1].Args)
	}
}

func TestParseNoOps(t *testing.T) {
	_, _, ops, err := Parse([]string{"label"})
	if err != nil {
		t.Fatal("should not error — empty ops is valid at parse level")
	}
	if len(ops) != 0 {
		t.Fatal("expected 0 ops")
	}
}

func TestParseUnknownOp(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "+bogus"})
	if err == nil {
		t.Fatal("expected error for unknown op")
	}
}

func TestParseGlobalFlags(t *testing.T) {
	flags, label, ops, err := Parse([]string{"-v", "label", "+check"})
	if err != nil {
		t.Fatal(err)
	}
	if !flags.Verbose {
		t.Fatal("verbose should be true")
	}
	if label != "label" {
		t.Fatalf("label = %q", label)
	}
	if len(ops) != 1 {
		t.Fatalf("ops = %d", len(ops))
	}
}

func TestParseStampsDir(t *testing.T) {
	flags, label, _, err := Parse([]string{"--stamps-dir", "/tmp/s", "label", "+check"})
	if err != nil {
		t.Fatal(err)
	}
	if flags.StampsDir != "/tmp/s" {
		t.Fatalf("stamps dir = %q", flags.StampsDir)
	}
	if label != "label" {
		t.Fatalf("label = %q", label)
	}
}

func TestParseMissingLabelWithOperation(t *testing.T) {
	_, _, _, err := Parse([]string{"+add-names", "a.c"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing label") {
		t.Fatalf("expected 'missing label' error, got: %v", err)
	}
}

func TestParseMissingLabel(t *testing.T) {
	_, _, _, err := Parse([]string{})
	if err == nil {
		t.Fatal("expected error for missing label")
	}
}

func TestParseNonOpArgAfterLabel(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "notanop"})
	if err == nil {
		t.Fatal("expected error for non-op arg after label")
	}
}

// --- ExtractFlags tests ---

func TestExtractFlagsVerbose(t *testing.T) {
	f := Flags{}
	remaining := ExtractFlags(&f, []string{"-v", "a.c", "b.c"})
	if !f.Verbose {
		t.Fatal("expected verbose")
	}
	if len(remaining) != 2 || remaining[0] != "a.c" || remaining[1] != "b.c" {
		t.Fatalf("remaining = %v", remaining)
	}
}

func TestExtractFlagsStampsDir(t *testing.T) {
	f := Flags{}
	remaining := ExtractFlags(&f, []string{"--stamps-dir", "/tmp/s", "a.c"})
	if f.StampsDir != "/tmp/s" {
		t.Fatalf("stamps dir = %q", f.StampsDir)
	}
	if len(remaining) != 1 || remaining[0] != "a.c" {
		t.Fatalf("remaining = %v", remaining)
	}
}

func TestExtractFlagsNoFlags(t *testing.T) {
	f := Flags{Verbose: true}
	remaining := ExtractFlags(&f, []string{"a.c", ".c"})
	if !f.Verbose {
		t.Fatal("verbose should be preserved")
	}
	if len(remaining) != 2 {
		t.Fatalf("remaining = %v", remaining)
	}
}

func TestExtractFlagsEmpty(t *testing.T) {
	f := Flags{}
	remaining := ExtractFlags(&f, []string{})
	if remaining != nil {
		t.Fatalf("remaining = %v", remaining)
	}
}

func TestOperationLocalVerbose(t *testing.T) {
	// Parse: label +check -v
	_, _, ops, err := Parse([]string{"label", "+check", "-v"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 1 {
		t.Fatalf("ops = %d", len(ops))
	}
	// After parse, -v is still in the args (extraction happens at execute time)
	found := false
	for _, a := range ops[0].Args {
		if a == "-v" {
			found = true
		}
	}
	if !found {
		t.Fatal("-v should be in check args")
	}
}

func TestOperationLocalDoesNotLeakToNext(t *testing.T) {
	// Parse: label +check -v +stamp-facts
	// -v should be in check's args, not stamp-facts'
	_, _, ops, err := Parse([]string{"label", "+check", "-v", "+stamp-facts"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 {
		t.Fatalf("ops = %d", len(ops))
	}
	// -v in check args
	found := false
	for _, a := range ops[0].Args {
		if a == "-v" {
			found = true
		}
	}
	if !found {
		t.Fatal("-v should be in check args")
	}
	// no -v in stamp-facts args
	for _, a := range ops[1].Args {
		if a == "-v" {
			t.Fatal("-v should not be in stamp-facts args")
		}
	}
}
