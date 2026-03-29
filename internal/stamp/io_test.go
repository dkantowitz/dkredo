package stamp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := NewStampState("test-label")
	s.AddEntry("src/main.c", "blake3:abc123 size:100")
	s.AddEntry("src/util.c", "blake3:def456 size:200")
	s.AddEntry("include/config.h", "")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	got, err := ReadStamp(stampsDir, "test-label", false)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}

	if got.Label != "test-label" {
		t.Errorf("label = %q, want %q", got.Label, "test-label")
	}
	if len(got.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(got.Entries))
	}
	// Entries should be sorted
	if got.Entries[0].Path != "include/config.h" || got.Entries[0].Facts != "" {
		t.Errorf("entry[0] = %+v", got.Entries[0])
	}
	if got.Entries[1].Path != "src/main.c" || got.Entries[1].Facts != "blake3:abc123 size:100" {
		t.Errorf("entry[1] = %+v", got.Entries[1])
	}
	if got.Entries[2].Path != "src/util.c" || got.Entries[2].Facts != "blake3:def456 size:200" {
		t.Errorf("entry[2] = %+v", got.Entries[2])
	}
}

func TestWriteCreatesStampsDir(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, "nested", ".stamps")

	s := NewStampState("label")
	s.AddEntry("a.c", "")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	if _, err := os.Stat(stampsDir); err != nil {
		t.Fatalf(".stamps dir not created: %v", err)
	}
}

func TestReadMissingStamp(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0755)

	s, err := ReadStamp(stampsDir, "nonexistent", false)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}
	if len(s.Entries) != 0 {
		t.Fatal("expected empty state")
	}
	if s.Label != "nonexistent" {
		t.Errorf("label = %q", s.Label)
	}
}

func TestAtomicWriteNoTmpRemains(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := NewStampState("label")
	s.AddEntry("a.c", "")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	entries, _ := os.ReadDir(stampsDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || len(e.Name()) > 20 {
			// Check for tmp files
			if e.Name() != "label" {
				// Only the stamp file should exist
			}
		}
	}
	// Verify only stamp file exists
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected 1 file in stamps dir, got %v", names)
	}
}

func TestLabelWithSlash(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := NewStampState("output/config.json")
	s.AddEntry("a.c", "")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	expected := filepath.Join(stampsDir, "output%2Fconfig.json")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected file at %s: %v", expected, err)
	}

	got, err := ReadStamp(stampsDir, "output/config.json", false)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatal("roundtrip failed")
	}
}

func TestPathWithTab(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := NewStampState("test")
	s.AddEntry("dir\tname/file", "blake3:abc size:1")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	got, err := ReadStamp(stampsDir, "test", false)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}
	if got.Entries[0].Path != "dir\tname/file" {
		t.Errorf("path roundtrip failed: got %q", got.Entries[0].Path)
	}
	if got.Entries[0].Facts != "blake3:abc size:1" {
		t.Errorf("facts roundtrip failed: got %q", got.Entries[0].Facts)
	}
}

func TestPathWithPercent(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := NewStampState("test")
	s.AddEntry("100%/file", "")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	got, err := ReadStamp(stampsDir, "test", false)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}
	if got.Entries[0].Path != "100%/file" {
		t.Errorf("roundtrip failed: got %q", got.Entries[0].Path)
	}
}

func TestPathWithSpaces(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := NewStampState("test")
	s.AddEntry("my file.c", "blake3:xyz size:10")

	if err := WriteStamp(stampsDir, s, false); err != nil {
		t.Fatalf("WriteStamp: %v", err)
	}

	got, err := ReadStamp(stampsDir, "test", false)
	if err != nil {
		t.Fatalf("ReadStamp: %v", err)
	}
	if got.Entries[0].Path != "my file.c" {
		t.Errorf("roundtrip failed: got %q", got.Entries[0].Path)
	}
}

func TestFindStampsDirInCwd(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	found := FindStampsDir()
	if found != stampsDir {
		t.Errorf("found %q, want %q", found, stampsDir)
	}
}

func TestFindStampsDirInParent(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0755)
	child := filepath.Join(dir, "subdir")
	os.MkdirAll(child, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(child)
	defer os.Chdir(oldWd)

	found := FindStampsDir()
	if found != stampsDir {
		t.Errorf("found %q, want %q", found, stampsDir)
	}
}

func TestFindStampsDirInGrandparent(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0755)
	grandchild := filepath.Join(dir, "a", "b")
	os.MkdirAll(grandchild, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(grandchild)
	defer os.Chdir(oldWd)

	found := FindStampsDir()
	if found != stampsDir {
		t.Errorf("found %q, want %q", found, stampsDir)
	}
}

func TestFindStampsDirNotFound(t *testing.T) {
	dir := t.TempDir()

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldWd)

	found := FindStampsDir()
	if found != "" {
		t.Errorf("expected empty, got %q", found)
	}
}

func TestNestedProject(t *testing.T) {
	dir := t.TempDir()
	parentStamps := filepath.Join(dir, ".stamps")
	os.MkdirAll(parentStamps, 0755)
	childDir := filepath.Join(dir, "child")
	childStamps := filepath.Join(childDir, ".stamps")
	os.MkdirAll(childStamps, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(childDir)
	defer os.Chdir(oldWd)

	found := FindStampsDir()
	if found != childStamps {
		t.Errorf("found %q, want %q (child should win)", found, childStamps)
	}
}
