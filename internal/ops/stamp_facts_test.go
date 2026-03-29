package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dkredo/internal/stamp"
)

func TestStampFactsAll(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.c"), []byte("world"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")
	state.AddEntry("b.c", "")
	state.Modified = false

	err := StampFacts(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(state.Entries[0].Facts, "blake3:") {
		t.Fatalf("a.c facts not computed: %q", state.Entries[0].Facts)
	}
	if !strings.HasPrefix(state.Entries[1].Facts, "blake3:") {
		t.Fatalf("b.c facts not computed: %q", state.Entries[1].Facts)
	}
	if !state.Modified {
		t.Fatal("should be modified")
	}
}

func TestStampFactsByFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.h"), []byte("world"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")
	state.AddEntry("b.h", "")

	err := StampFacts(state, []string{".c"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(state.FindEntry("a.c").Facts, "blake3:") {
		t.Fatal("a.c should have facts")
	}
	if state.FindEntry("b.h").Facts != "" {
		t.Fatal("b.h should not have facts")
	}
}

func TestStampFactsMissingFile(t *testing.T) {
	dir := t.TempDir()
	state := stamp.NewStampState("test")
	state.AddEntry("gone.c", "")

	err := StampFacts(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if state.Entries[0].Facts != "missing:true" {
		t.Fatalf("expected missing:true, got %q", state.Entries[0].Facts)
	}
}

func TestStampFactsDoesNotAddNames(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.c"), []byte("world"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")

	StampFacts(state, []string{}, nil, dir, false)
	if len(state.Entries) != 1 {
		t.Fatal("should not add new names")
	}
}

func TestStampFactsDeterministic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("deterministic"), 0644)

	state1 := stamp.NewStampState("test")
	state1.AddEntry("a.c", "")
	StampFacts(state1, []string{}, nil, dir, false)

	state2 := stamp.NewStampState("test")
	state2.AddEntry("a.c", "")
	StampFacts(state2, []string{}, nil, dir, false)

	if state1.Entries[0].Facts != state2.Entries[0].Facts {
		t.Fatal("facts should be deterministic")
	}
}
