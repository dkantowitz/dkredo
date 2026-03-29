package ops

import (
	"os"
	"path/filepath"
	"testing"

	"dkredo/internal/stamp"
)

func TestRemoveNamesByExactPath(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.c", "blake3:def size:200")
	state.Modified = false
	err := RemoveNames(state, []string{"a.c"}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(state.Entries))
	}
	if state.Entries[0].Path != "b.c" {
		t.Fatal("wrong entry remaining")
	}
	if !state.Modified {
		t.Fatal("should be modified")
	}
}

func TestRemoveNamesBySuffixFilter(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.h", "blake3:def size:200")
	err := RemoveNames(state, []string{".h"}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(state.Entries))
	}
	if state.Entries[0].Path != "a.c" {
		t.Fatal("wrong entry remaining")
	}
}

func TestRemoveNamesAll(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "facts")
	state.AddEntry("b.c", "facts")
	state.AddEntry("c.h", "facts")
	state.Modified = false
	err := RemoveNames(state, []string{}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(state.Entries))
	}
	if !state.Modified {
		t.Fatal("should be modified")
	}
}

func TestRemoveNamesNE_FileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hello"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	err := RemoveNames(state, []string{"-ne"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 {
		t.Fatal("file exists, should NOT be removed")
	}
}

func TestRemoveNamesNE_FileMissingFactMissing(t *testing.T) {
	dir := t.TempDir()
	state := stamp.NewStampState("test")
	state.AddEntry("gone.c", "missing:true")
	err := RemoveNames(state, []string{"-ne"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 {
		t.Fatal("missing:true entry should NOT be removed")
	}
}

func TestRemoveNamesNE_FileMissingFactStale(t *testing.T) {
	dir := t.TempDir()
	state := stamp.NewStampState("test")
	state.AddEntry("gone.c", "blake3:abc size:100")
	err := RemoveNames(state, []string{"-ne"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 0 {
		t.Fatal("stale entry for missing file should be removed")
	}
}

func TestRemoveNamesNonexistentName(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "facts")
	err := RemoveNames(state, []string{"x.c"}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 1 {
		t.Fatal("should not remove anything")
	}
}
