package main

import (
	"testing"
)

func TestExpandIfchangeWithFiles(t *testing.T) {
	ops, err := ExpandAlias("ifchange", []string{"a.c", "b.c"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+add-names", "a.c", "b.c", "+check"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandIfchangeNoFiles(t *testing.T) {
	ops, err := ExpandAlias("ifchange", nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+check"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandStamp(t *testing.T) {
	ops, err := ExpandAlias("stamp", []string{"a.c", "b.c"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+remove-names", "+add-names", "a.c", "b.c", "+stamp-facts"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandStampAppend(t *testing.T) {
	ops, err := ExpandAlias("stamp", []string{"--append", "a.c", "b.c"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+add-names", "a.c", "b.c", "+stamp-facts"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandStampWithDepfile(t *testing.T) {
	ops, err := ExpandAlias("stamp", []string{"-M", "out.d"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+remove-names", "+add-names", "-M", "out.d", "+stamp-facts"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandStampAppendWithDepfile(t *testing.T) {
	ops, err := ExpandAlias("stamp", []string{"--append", "-M", "out.d"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+add-names", "-M", "out.d", "+stamp-facts"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandAlways(t *testing.T) {
	ops, err := ExpandAlias("always", nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+clear-facts"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandFnames(t *testing.T) {
	ops, err := ExpandAlias("fnames", []string{".c"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+names", "-e", ".c"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandFnamesNoFilter(t *testing.T) {
	ops, err := ExpandAlias("fnames", nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"+names", "-e"}
	if !sliceEqual(ops, expected) {
		t.Fatalf("got %v, want %v", ops, expected)
	}
}

func TestExpandUnknownAlias(t *testing.T) {
	_, err := ExpandAlias("bogus", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdFlag(t *testing.T) {
	_, label, ops, err := Parse([]string{"label", "--cmd", "ifchange", "a.c"})
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
		t.Fatalf("ops = %v, %v", ops[0].Name, ops[1].Name)
	}
}

func TestCmdInvalidAlias(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "--cmd", "bogus"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdNoAliasName(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "--cmd"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCmdMixedWithOps(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "--cmd", "ifchange", "+names"})
	if err == nil {
		t.Fatal("expected error for --cmd mixed with +ops")
	}
}

func TestMultipleCmd(t *testing.T) {
	_, _, _, err := Parse([]string{"label", "--cmd", "ifchange", "--cmd", "stamp"})
	if err == nil {
		t.Fatal("expected error for multiple --cmd")
	}
}

func TestCmdStampAppend(t *testing.T) {
	_, _, ops, err := Parse([]string{"label", "--cmd", "stamp", "--append", "a.c"})
	if err != nil {
		t.Fatal(err)
	}
	// Should expand to: +add-names a.c +stamp-facts
	if len(ops) != 2 {
		t.Fatalf("ops = %d", len(ops))
	}
	if ops[0].Name != "add-names" {
		t.Fatalf("expected add-names, got %s", ops[0].Name)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
