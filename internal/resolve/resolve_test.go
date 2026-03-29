package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dkredo/internal/stamp"
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

// --- ResolveFilters tests ---

func TestResolveFiltersSuffixPassthrough(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	result, err := ResolveFilters([]string{".c", ".h"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 filters, got %d: %v", len(result), result)
	}
	if result[0] != ".c" {
		t.Fatalf("expected .c, got %s", result[0])
	}
	if result[1] != ".h" {
		t.Fatalf("expected .h, got %s", result[1])
	}
}

func TestResolveFiltersMixed(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	result, err := ResolveFilters([]string{".c", "src/main.c"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 filters, got %d: %v", len(result), result)
	}
	// Suffix filter should be first (preserved order), path should be canonicalized
	if result[0] != ".c" {
		t.Fatalf("expected .c as first filter, got %s", result[0])
	}
	if result[1] != "src/main.c" {
		t.Fatalf("expected src/main.c as second filter, got %s", result[1])
	}
}

func TestResolveFiltersEmpty(t *testing.T) {
	dir := t.TempDir()
	result, err := ResolveFilters(nil, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 filters, got %d: %v", len(result), result)
	}
}

func TestResolveFiltersWithFileInput(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	listFile := filepath.Join(dir, "filter-list.txt")
	os.WriteFile(listFile, []byte("src/a.c\nsrc/b.h\n"), 0644)

	result, err := ResolveFilters([]string{".c", "-@", listFile}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	// .c suffix + 2 paths from file = 3
	if len(result) != 3 {
		t.Fatalf("expected 3 filters, got %d: %v", len(result), result)
	}
	if result[0] != ".c" {
		t.Fatalf("expected .c as first filter, got %s", result[0])
	}
}

// --- FilterEntries tests ---

func TestFilterEntriesEmptyFilter(t *testing.T) {
	entries := []stamp.Entry{
		{Path: "src/main.c"},
		{Path: "src/util.h"},
		{Path: "src/lib.c"},
	}
	result := FilterEntries(entries, nil)
	if len(result) != 3 {
		t.Fatalf("expected all 3 entries, got %d", len(result))
	}
}

func TestFilterEntriesBySuffix(t *testing.T) {
	entries := []stamp.Entry{
		{Path: "src/main.c"},
		{Path: "src/util.h"},
		{Path: "src/lib.c"},
	}
	result := FilterEntries(entries, []string{".c"})
	if len(result) != 2 {
		t.Fatalf("expected 2 .c entries, got %d: %v", len(result), result)
	}
	for _, e := range result {
		if filepath.Ext(e.Path) != ".c" {
			t.Fatalf("expected .c file, got %s", e.Path)
		}
	}
}

func TestFilterEntriesByExactPath(t *testing.T) {
	entries := []stamp.Entry{
		{Path: "src/main.c"},
		{Path: "src/util.h"},
		{Path: "src/lib.c"},
	}
	result := FilterEntries(entries, []string{"src/util.h"})
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(result), result)
	}
	if result[0].Path != "src/util.h" {
		t.Fatalf("expected src/util.h, got %s", result[0].Path)
	}
}

func TestFilterEntriesNoMatch(t *testing.T) {
	entries := []stamp.Entry{
		{Path: "src/main.c"},
		{Path: "src/util.h"},
	}
	result := FilterEntries(entries, []string{".py"})
	if len(result) != 0 {
		t.Fatalf("expected 0 entries, got %d: %v", len(result), result)
	}
}

func TestFilterEntriesMultipleFilters(t *testing.T) {
	entries := []stamp.Entry{
		{Path: "src/main.c"},
		{Path: "src/util.h"},
		{Path: "src/lib.go"},
		{Path: "src/extra.c"},
	}
	result := FilterEntries(entries, []string{".c", ".h"})
	if len(result) != 3 {
		t.Fatalf("expected 3 entries (.c and .h), got %d: %v", len(result), result)
	}
	for _, e := range result {
		ext := filepath.Ext(e.Path)
		if ext != ".c" && ext != ".h" {
			t.Fatalf("unexpected file %s", e.Path)
		}
	}
}
