package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDepfileSimple(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.d")
	os.WriteFile(f, []byte("out.o: src/main.c src/util.h\n"), 0644)

	paths, err := ParseDepfile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != "src/main.c" || paths[1] != "src/util.h" {
		t.Fatalf("unexpected paths: %v", paths)
	}
}

func TestParseDepfileMultiline(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.d")
	os.WriteFile(f, []byte("out.o: a.c \\\n  b.c c.c\n"), 0644)

	paths, err := ParseDepfile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
	}
}

func TestParseDepfileMultipleTargets(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.d")
	os.WriteFile(f, []byte("out.o out.d: a.c b.c\n"), 0644)

	paths, err := ParseDepfile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestParseDepfileEscapedSpaces(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.d")
	os.WriteFile(f, []byte("out.o: my\\ file.c other.c\n"), 0644)

	paths, err := ParseDepfile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != "my file.c" {
		t.Fatalf("expected 'my file.c', got %q", paths[0])
	}
}

func TestParseDepfileEmpty(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.d")
	os.WriteFile(f, []byte(""), 0644)

	paths, err := ParseDepfile(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected 0 paths, got %d", len(paths))
	}
}

func TestParseDepfileMissing(t *testing.T) {
	_, err := ParseDepfile("/nonexistent/out.d")
	if err == nil {
		t.Fatal("expected error for missing depfile")
	}
}

func TestParseDepfileMalformed(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "out.d")
	os.WriteFile(f, []byte("garbage without colon\n"), 0644)

	_, err := ParseDepfile(f)
	if err == nil {
		t.Fatal("expected error for malformed depfile")
	}
}
