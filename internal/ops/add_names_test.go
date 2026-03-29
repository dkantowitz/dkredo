package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dkredo/internal/stamp"
)

func TestAddNamesToEmptyStamp(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	state := stamp.NewStampState("test")
	err := AddNames(state, []string{"src/a.c", "src/b.c"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(state.Entries))
	}
	if state.Entries[0].Facts != "" || state.Entries[1].Facts != "" {
		t.Fatal("new entries should have empty facts")
	}
	if !state.Modified {
		t.Fatal("should be modified")
	}
}

func TestAddNamesDuplicateIgnored(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	state := stamp.NewStampState("test")
	state.AddEntry("src/a.c", "blake3:abc size:100")
	state.Modified = false
	err := AddNames(state, []string{"src/a.c"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(state.Entries))
	}
	if state.Entries[0].Facts != "blake3:abc size:100" {
		t.Fatal("existing facts should be preserved")
	}
}

func TestAddNamesPreservesExistingFacts(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	state := stamp.NewStampState("test")
	state.AddEntry("src/a.c", "blake3:abc size:100")
	state.Modified = false
	err := AddNames(state, []string{"src/b.c"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(state.Entries))
	}
	e := state.FindEntry("src/a.c")
	if e.Facts != "blake3:abc size:100" {
		t.Fatal("a.c facts should be preserved")
	}
}

func TestAddNamesFromStdin(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	state := stamp.NewStampState("test")
	stdin := strings.NewReader("x.c\ny.c\n")
	err := AddNames(state, []string{"-"}, stdin, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(state.Entries))
	}
}

func TestAddNamesFromFileInput(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	listFile := filepath.Join(dir, "list.txt")
	os.WriteFile(listFile, []byte("a.c\nb.c\n"), 0644)

	state := stamp.NewStampState("test")
	err := AddNames(state, []string{"-@", listFile}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(state.Entries))
	}
}

func TestAddNamesFromDepfile(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	depFile := filepath.Join(dir, "out.d")
	os.WriteFile(depFile, []byte("out.o: src/main.c include/config.h\n"), 0644)

	state := stamp.NewStampState("test")
	err := AddNames(state, []string{"-M", depFile}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(state.Entries), state.Entries)
	}
}

func TestAddNamesEmptyArgs(t *testing.T) {
	state := stamp.NewStampState("test")
	err := AddNames(state, []string{}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 0 {
		t.Fatal("expected 0 entries")
	}
	if state.Modified {
		t.Fatal("should not be modified")
	}
}

func TestAddNamesEntriesSorted(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	state := stamp.NewStampState("test")
	AddNames(state, []string{"z.c", "a.c", "m.c"}, nil, dir, false)
	if state.Entries[0].Path != "a.c" || state.Entries[1].Path != "m.c" || state.Entries[2].Path != "z.c" {
		t.Fatalf("entries not sorted: %v", state.Entries)
	}
}
