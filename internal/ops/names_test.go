package ops

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"dkredo/internal/stamp"
)

func TestNamesAll(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.h", "blake3:def size:200")
	var buf bytes.Buffer
	err := Names(state, []string{}, "/project", &buf, false)
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "a.c\nb.h\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestNamesFilterBySuffix(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "facts")
	state.AddEntry("b.h", "facts")
	var buf bytes.Buffer
	Names(state, []string{".c"}, "/project", &buf, false)
	if buf.String() != "a.c\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestNamesExistsOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "facts")
	state.AddEntry("gone.c", "missing:true")
	var buf bytes.Buffer
	Names(state, []string{"-e"}, dir, &buf, false)
	if buf.String() != "a.c\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestNamesExistsWithFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.c"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.h"), []byte("world"), 0644)

	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "facts")
	state.AddEntry("b.h", "facts")
	state.AddEntry("gone.c", "facts")
	var buf bytes.Buffer
	Names(state, []string{"-e", ".c"}, dir, &buf, false)
	if buf.String() != "a.c\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestNamesEmptyStamp(t *testing.T) {
	state := stamp.NewStampState("test")
	var buf bytes.Buffer
	Names(state, []string{}, "/project", &buf, false)
	if buf.String() != "" {
		t.Fatalf("expected empty output, got %q", buf.String())
	}
}

func TestFactsAll(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.h", "blake3:def size:200")
	var buf bytes.Buffer
	Facts(state, []string{}, "/project", &buf, false)
	expected := "a.c\tblake3:abc size:100\nb.h\tblake3:def size:200\n"
	if buf.String() != expected {
		t.Fatalf("unexpected output: %q, want %q", buf.String(), expected)
	}
}

func TestFactsWithFilter(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.h", "blake3:def size:200")
	var buf bytes.Buffer
	Facts(state, []string{".c"}, "/project", &buf, false)
	if buf.String() != "a.c\tblake3:abc size:100\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestFactsEmptyFacts(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")
	var buf bytes.Buffer
	Facts(state, []string{}, "/project", &buf, false)
	if buf.String() != "a.c\t\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
