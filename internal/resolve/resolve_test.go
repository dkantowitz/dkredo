package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFilesPositional(t *testing.T) {
	dir := t.TempDir()
	// Create files so Abs works from the dir context
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	paths, err := ResolveFiles([]string{"src/a.c", "src/b.c"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesStdinNewline(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	stdin := strings.NewReader("x.c\ny.c\n")
	paths, err := ResolveFiles([]string{"-"}, stdin, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesStdinNull(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	stdin := strings.NewReader("x.c\x00y.c\x00")
	paths, err := ResolveFiles([]string{"-0"}, stdin, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesFileInput(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	listFile := filepath.Join(dir, "list.txt")
	os.WriteFile(listFile, []byte("a.c\nb.c\nc.c\n"), 0644)

	paths, err := ResolveFiles([]string{"-@", listFile}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesFileInputNull(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	listFile := filepath.Join(dir, "list.txt")
	os.WriteFile(listFile, []byte("a.c\x00b.c\x00"), 0644)

	paths, err := ResolveFiles([]string{"-@0", listFile}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesDepfile(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	depFile := filepath.Join(dir, "out.d")
	os.WriteFile(depFile, []byte("out.o: src/main.c src/util.h\n"), 0644)

	paths, err := ResolveFiles([]string{"-M", depFile}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesDedup(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	stdin := strings.NewReader("a.c\n")
	paths, err := ResolveFiles([]string{"a.c", "-"}, stdin, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 path (deduped), got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesCanonical(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	paths, err := ResolveFiles([]string{"./src/main.c", "src/main.c"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 path (canonical dedup), got %d: %v", len(paths), paths)
	}
}

func TestResolveFilesEmpty(t *testing.T) {
	dir := t.TempDir()
	paths, err := ResolveFiles(nil, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected 0 paths, got %d", len(paths))
	}
}

func TestMatchesFilterExactPath(t *testing.T) {
	if !MatchesFilter("src/main.c", "src/main.c") {
		t.Fatal("exact match should match")
	}
	if MatchesFilter("src/main.c", "src/other.c") {
		t.Fatal("different path should not match")
	}
}

func TestMatchesFilterSuffix(t *testing.T) {
	if !MatchesFilter("src/main.c", ".c") {
		t.Fatal(".c should match src/main.c")
	}
	if MatchesFilter("src/main.c", ".h") {
		t.Fatal(".h should not match src/main.c")
	}
}

func TestIsSuffixFilter(t *testing.T) {
	if !isSuffixFilter(".c") {
		t.Fatal(".c should be suffix filter")
	}
	if !isSuffixFilter(".h") {
		t.Fatal(".h should be suffix filter")
	}
	if isSuffixFilter("src/main.c") {
		t.Fatal("path should not be suffix filter")
	}
	if isSuffixFilter(".") {
		t.Fatal(". alone should not be suffix filter")
	}
}
