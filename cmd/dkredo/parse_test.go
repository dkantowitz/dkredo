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
	cfg, label, ops, err := Parse([]string{"-v", "label", "+check"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Verbose {
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
	cfg, label, _, err := Parse([]string{"--stamps-dir", "/tmp/s", "label", "+check"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StampsDir != "/tmp/s" {
		t.Fatalf("stamps dir = %q", cfg.StampsDir)
	}
	if label != "label" {
		t.Fatalf("label = %q", label)
	}
}

func TestParseMissingLabel(t *testing.T) {
	_, _, _, err := Parse([]string{})
	if err == nil {
		t.Fatal("expected error for missing label")
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

func TestParseNonOpArgAfterLabel(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "notanop"})
	if err == nil {
		t.Fatal("expected error for non-op arg after label")
	}
}
