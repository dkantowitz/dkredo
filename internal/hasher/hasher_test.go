package hasher

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/zeebo/blake3"
)

// blake3Hex computes the expected BLAKE3 hex digest for the given data.
func blake3Hex(data []byte) string {
	h := blake3.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func TestHashFileWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	facts, err := HashFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if facts.Missing {
		t.Fatal("expected Missing to be false")
	}
	if facts.Size != 5 {
		t.Fatalf("expected Size=5, got %d", facts.Size)
	}

	want := blake3Hex([]byte("hello"))
	if facts.Blake3 != want {
		t.Fatalf("expected Blake3=%s, got %s", want, facts.Blake3)
	}
	if len(facts.Blake3) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(facts.Blake3))
	}
}

func TestHashFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	facts, err := HashFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if facts.Missing {
		t.Fatal("expected Missing to be false")
	}
	if facts.Size != 0 {
		t.Fatalf("expected Size=0, got %d", facts.Size)
	}

	want := blake3Hex([]byte{})
	if facts.Blake3 != want {
		t.Fatalf("expected Blake3=%s, got %s", want, facts.Blake3)
	}
}

func TestHashFileMissing(t *testing.T) {
	facts, err := HashFile("/nonexistent/path/to/file.txt")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if !facts.Missing {
		t.Fatal("expected Missing to be true")
	}
	if facts.Size != -1 {
		t.Fatalf("expected Size=-1, got %d", facts.Size)
	}
	if facts.Blake3 != "" {
		t.Fatalf("expected empty Blake3, got %s", facts.Blake3)
	}
}

func TestHashFilePermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not reliable on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denied as root")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(path, []byte("secret"), 0000); err != nil {
		t.Fatal(err)
	}

	_, err := HashFile(path)
	if err == nil {
		t.Fatal("expected error for permission denied file")
	}
}

func TestHashFileFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	factTarget, err := HashFile(target)
	if err != nil {
		t.Fatalf("unexpected error hashing target: %v", err)
	}
	factLink, err := HashFile(link)
	if err != nil {
		t.Fatalf("unexpected error hashing link: %v", err)
	}

	if factLink.Blake3 != factTarget.Blake3 {
		t.Fatalf("symlink hash %s != target hash %s", factLink.Blake3, factTarget.Blake3)
	}
	if factLink.Size != factTarget.Size {
		t.Fatalf("symlink size %d != target size %d", factLink.Size, factTarget.Size)
	}
	if factLink.Missing {
		t.Fatal("symlink should not be missing")
	}
}

func TestHashDirEmpty(t *testing.T) {
	dir := t.TempDir()

	results, err := HashDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty list, got %d items", len(results))
	}
}

func TestHashDirWithFiles(t *testing.T) {
	dir := t.TempDir()
	// Create files in non-alphabetical order
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bravo"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := HashDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Should be sorted by path
	if results[0].Path != "a.txt" {
		t.Fatalf("expected first path to be a.txt, got %s", results[0].Path)
	}
	if results[1].Path != "b.txt" {
		t.Fatalf("expected second path to be b.txt, got %s", results[1].Path)
	}

	// Verify hashes
	wantA := blake3Hex([]byte("alpha"))
	if results[0].Facts.Blake3 != wantA {
		t.Fatalf("expected hash %s for a.txt, got %s", wantA, results[0].Facts.Blake3)
	}
	wantB := blake3Hex([]byte("bravo"))
	if results[1].Facts.Blake3 != wantB {
		t.Fatalf("expected hash %s for b.txt, got %s", wantB, results[1].Facts.Blake3)
	}
}

func TestHashDirDeterminism(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "y.txt"), []byte("y"), 0644); err != nil {
		t.Fatal(err)
	}

	r1, err := HashDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r2, err := HashDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r1) != len(r2) {
		t.Fatalf("result lengths differ: %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Path != r2[i].Path {
			t.Fatalf("paths differ at %d: %s vs %s", i, r1[i].Path, r2[i].Path)
		}
		if r1[i].Facts.Blake3 != r2[i].Facts.Blake3 {
			t.Fatalf("hashes differ at %d", i)
		}
		if r1[i].Facts.Size != r2[i].Facts.Size {
			t.Fatalf("sizes differ at %d", i)
		}
	}
}

func TestHashDirFollowsSymlinks(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(subdir, "target.txt")
	if err := os.WriteFile(target, []byte("linked"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlink to the file in the main dir
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	results, err := HashDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have link.txt and sub/target.txt
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Both should have the same hash (same content)
	want := blake3Hex([]byte("linked"))
	for _, r := range results {
		if r.Facts.Blake3 != want {
			t.Fatalf("expected hash %s for %s, got %s", want, r.Path, r.Facts.Blake3)
		}
	}
}

func TestHashDirSymlinkLoop(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink from sub/loop -> dir (creates a cycle)
	loop := filepath.Join(sub, "loop")
	if err := os.Symlink(dir, loop); err != nil {
		t.Fatal(err)
	}

	_, err := HashDir(dir)
	if err == nil {
		t.Fatal("expected error for symlink loop, got nil")
	}
	t.Logf("got expected error: %v", err)
}

func TestFactsSizeAlongsideBlake3(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	content := []byte("some data for size test")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	facts, err := HashFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Size must always be present alongside blake3
	if facts.Blake3 == "" {
		t.Fatal("expected non-empty Blake3")
	}
	if facts.Size != int64(len(content)) {
		t.Fatalf("expected Size=%d, got %d", len(content), facts.Size)
	}

	// For missing files, both should reflect missing state
	missingFacts, err := HashFile("/nonexistent/file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if missingFacts.Blake3 != "" {
		t.Fatal("expected empty Blake3 for missing file")
	}
	if missingFacts.Size != -1 {
		t.Fatalf("expected Size=-1 for missing file, got %d", missingFacts.Size)
	}
}
