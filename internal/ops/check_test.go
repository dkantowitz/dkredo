package ops

import (
	"os"
	"path/filepath"
	"testing"

	"dkredo/internal/hasher"
	"dkredo/internal/stamp"
)

func TestCheckEmptyStamp(t *testing.T) {
	state := stamp.NewStampState("test")
	code, err := Check(state, []string{}, nil, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("expected exit 1 (unchanged), got %d", code)
	}
}

func TestCheckAllFactsMatch(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := hasher.FileFacts(f)
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", facts)

	code, err := Check(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("expected exit 1 (unchanged), got %d", code)
	}
}

func TestCheckContentChanged(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := hasher.FileFacts(f)
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", facts)

	os.WriteFile(f, []byte("world"), 0644) // same size, different content

	code, err := Check(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}
}

func TestCheckSizeChanged(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hi"), 0644)

	facts, _ := hasher.FileFacts(f)
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", facts)

	os.WriteFile(f, []byte("hello world longer"), 0644)

	code, err := Check(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}
}

func TestCheckFileAppeared(t *testing.T) {
	dir := t.TempDir()
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "missing:true")

	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("appeared"), 0644)

	code, err := Check(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}
}

func TestCheckFileDisappeared(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := hasher.FileFacts(f)
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", facts)

	os.Remove(f)

	code, err := Check(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}
}

func TestCheckNoFacts(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")

	code, err := Check(state, []string{}, nil, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed, no facts), got %d", code)
	}
}

func TestCheckUnknownFactKey(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:5 future:xyz")

	code, err := Check(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed, unknown key), got %d", code)
	}
}

func TestCheckWithFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.h"), []byte("world"), 0644)

	factsH, _ := hasher.FileFacts(filepath.Join(dir, "b.h"))
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "") // no facts → would be "changed"
	state.AddEntry("b.h", factsH)

	// Check only .h → should be unchanged
	code, err := Check(state, []string{".h"}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 {
		t.Fatalf("expected exit 1 (unchanged for .h filter), got %d", code)
	}
}

func TestCheckAssertChanged(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")

	code, err := CheckAssert(state, []string{}, nil, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0 (changed), got %d", code)
	}
}

func TestCheckAssertUnchanged(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.c")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := hasher.FileFacts(f)
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", facts)

	code, err := CheckAssert(state, []string{}, nil, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if code != 2 {
		t.Fatalf("expected exit 2 (assert unchanged → error), got %d", code)
	}
}
