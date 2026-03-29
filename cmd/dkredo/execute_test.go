package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dkredo/internal/stamp"
)

func TestExecutePipeline(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	ops := []Operation{
		{Name: "add-names", Args: []string{"a.c"}},
		{Name: "stamp-facts", Args: nil},
	}
	code := Execute("test", ops, stampsDir, false, nil, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	// Verify stamp was written
	s, err := stamp.ReadStamp(stampsDir, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(s.Entries))
	}
	if !strings.HasPrefix(s.Entries[0].Facts, "blake3:") {
		t.Fatalf("facts not computed: %q", s.Entries[0].Facts)
	}
}

func TestExecuteCheckUnchanged(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// First: stamp
	ops := []Operation{
		{Name: "add-names", Args: []string{"a.c"}},
		{Name: "stamp-facts", Args: nil},
	}
	Execute("test", ops, stampsDir, false, nil, &bytes.Buffer{})

	// Second: check should be unchanged
	ops2 := []Operation{{Name: "check", Args: nil}}
	code := Execute("test", ops2, stampsDir, false, nil, &bytes.Buffer{})
	if code != 1 {
		t.Fatalf("expected exit 1 (unchanged), got %d", code)
	}
}

func TestExecuteWritesOnExit1(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// +add-names a.c +check → check exit 0 (no stamp = changed, entry has no facts)
	// But a.c entry should persist
	ops := []Operation{
		{Name: "add-names", Args: []string{"a.c"}},
		{Name: "check", Args: nil},
	}
	code := Execute("test", ops, stampsDir, false, nil, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}

	// Verify stamp was written with a.c
	s, err := stamp.ReadStamp(stampsDir, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Entries) != 1 || s.Entries[0].Path != "a.c" {
		t.Fatalf("expected a.c entry, got %v", s.Entries)
	}
}

func TestExecuteStampThenCheckChanged(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hello"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	// Stamp
	ops := []Operation{
		{Name: "add-names", Args: []string{"a.c"}},
		{Name: "stamp-facts", Args: nil},
	}
	Execute("test", ops, stampsDir, false, nil, &bytes.Buffer{})

	// Modify file
	os.WriteFile(f, []byte("modified"), 0644)

	// Check → should be changed
	code := Execute("test", []Operation{{Name: "check", Args: nil}}, stampsDir, false, nil, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}
}

func TestExecuteNamesOutput(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	ops := []Operation{
		{Name: "add-names", Args: []string{"a.c"}},
	}
	Execute("test", ops, stampsDir, false, nil, &bytes.Buffer{})

	var buf bytes.Buffer
	code := Execute("test", []Operation{{Name: "names", Args: nil}}, stampsDir, false, nil, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if buf.String() != "a.c\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
