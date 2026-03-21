//go:build integration

package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dkantowitz/dk-redo/internal/stamp"
	"github.com/dkantowitz/dk-redo/internal/testutil"
)

// binaryPath holds the path to the compiled dk-redo binary.
// It is set once by TestMain before any tests run.
var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once into a temp directory.
	tmpDir, err := os.MkdirTemp("", "dk-redo-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "dk-redo")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/dk-redo")
	cmd.Dir = findModuleRoot()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build dk-redo: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// findModuleRoot walks up from the current directory to find the go.mod file.
func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find go.mod")
		}
		dir = parent
	}
}

// runDkRedo is a convenience wrapper around testutil.RunBinary that uses the
// pre-built binary path. The stampsDir is passed as empty so no env var is set;
// instead, --stamps-dir is passed as a CLI argument.
func runDkRedo(t *testing.T, stampsDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	return testutil.RunBinary(t, binaryPath, stampsDir, args...)
}

// TestFirstRun verifies that when no stamp exists, dk-ifchange exits 0 (changed).
func TestFirstRun(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	testutil.WriteTempFile(t, dir, "input.c", "hello")

	_, _, code := runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "myLabel", filepath.Join(dir, "input.c"))
	if code != 0 {
		t.Errorf("expected exit 0 (changed/first run), got %d", code)
	}
}

// TestUnchanged stamps files, then checks ifchange returns 1 (unchanged).
func TestUnchanged(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")
	f2 := testutil.WriteTempFile(t, dir, "b.txt", "bbb")

	// Stamp the files.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "lbl", f1, f2)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// ifchange should report unchanged.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "lbl", f1, f2)
	if code != 1 {
		t.Errorf("expected exit 1 (unchanged), got %d", code)
	}
}

// TestFileModified stamps files, modifies one, then ifchange should exit 0.
func TestFileModified(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	// Stamp.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "mod", f1)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Modify the file.
	time.Sleep(10 * time.Millisecond) // ensure different mtime
	if err := os.WriteFile(f1, []byte("changed content"), 0644); err != nil {
		t.Fatal(err)
	}

	// ifchange should detect the change.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "mod", f1)
	if code != 0 {
		t.Errorf("expected exit 0 (changed), got %d", code)
	}
}

// TestFileAdded stamps with one file, then adds another file to the args.
func TestFileAdded(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")
	f2 := testutil.WriteTempFile(t, dir, "b.txt", "bbb")

	// Stamp with only f1.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "add", f1)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// ifchange with f1 AND f2 — f2 is "added".
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "add", f1, f2)
	if code != 0 {
		t.Errorf("expected exit 0 (changed due to added file), got %d", code)
	}
}

// TestFileRemoved stamps with two files, then only passes one to ifchange.
func TestFileRemoved(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")
	f2 := testutil.WriteTempFile(t, dir, "b.txt", "bbb")

	// Stamp with both files.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "rem", f1, f2)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// ifchange with only f1 — f2 is "removed" from args.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "rem", f1)
	if code != 0 {
		t.Errorf("expected exit 0 (changed due to removed file), got %d", code)
	}
}

// TestDirFileAdded stamps a directory, then adds a new file to it.
func TestDirFileAdded(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	subDir := testutil.WriteTempDir(t, dir, "mydir", map[string]string{
		"one.txt": "one",
	})

	// Stamp the directory.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "diradd", subDir)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Add a new file to the directory.
	testutil.WriteTempFile(t, subDir, "two.txt", "two")

	// ifchange should detect the new file.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "diradd", subDir)
	if code != 0 {
		t.Errorf("expected exit 0 (changed due to added file in dir), got %d", code)
	}
}

// TestDirFileRemoved stamps a directory, then removes a file from it.
func TestDirFileRemoved(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	subDir := testutil.WriteTempDir(t, dir, "mydir", map[string]string{
		"one.txt": "one",
		"two.txt": "two",
	})

	// Stamp the directory.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "dirrem", subDir)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Remove a file from the directory.
	if err := os.Remove(filepath.Join(subDir, "two.txt")); err != nil {
		t.Fatal(err)
	}

	// ifchange should detect the removal.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "dirrem", subDir)
	if code != 0 {
		t.Errorf("expected exit 0 (changed due to removed file in dir), got %d", code)
	}
}

// TestMissingFileSentinel stamps a nonexistent file (missing:true sentinel),
// then creates the file — ifchange should exit 0 (changed).
func TestMissingFileSentinel(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	ghostFile := filepath.Join(dir, "ghost.txt")

	// Stamp the nonexistent file — hasher returns missing:true.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "miss", ghostFile)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Now create the file.
	if err := os.WriteFile(ghostFile, []byte("now exists"), 0644); err != nil {
		t.Fatal(err)
	}

	// ifchange should detect the file appeared.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "miss", ghostFile)
	if code != 0 {
		t.Errorf("expected exit 0 (changed: file appeared), got %d", code)
	}
}

// TestStampReplace verifies that running stamp twice replaces the previous stamp.
func TestStampReplace(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")
	f2 := testutil.WriteTempFile(t, dir, "b.txt", "bbb")

	// Stamp with f1 and f2.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "repl", f1, f2)
	if code != 0 {
		t.Fatalf("first stamp failed with exit %d", code)
	}

	// Stamp again with only f1 — this should replace, not append.
	_, _, code = runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "repl", f1)
	if code != 0 {
		t.Fatalf("second stamp failed with exit %d", code)
	}

	// Read the stamp and verify only f1 is present.
	s, err := stamp.Read(stampsDir, "repl")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Files) != 1 {
		t.Errorf("expected 1 file in stamp, got %d", len(s.Files))
	}
	if len(s.Files) > 0 && s.Files[0].Path != f1 {
		t.Errorf("expected stamped path %s, got %s", f1, s.Files[0].Path)
	}
}

// TestStampAppend verifies that --append merges new files into existing stamp.
func TestStampAppend(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")
	f2 := testutil.WriteTempFile(t, dir, "b.txt", "bbb")

	// Stamp with f1.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "app", f1)
	if code != 0 {
		t.Fatalf("first stamp failed with exit %d", code)
	}

	// Stamp --append with f2.
	_, _, code = runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "--append", "app", f2)
	if code != 0 {
		t.Fatalf("append stamp failed with exit %d", code)
	}

	// Read the stamp and verify both files are present.
	s, err := stamp.Read(stampsDir, "app")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Files) != 2 {
		t.Errorf("expected 2 files in stamp after append, got %d", len(s.Files))
	}
}

// TestAlways verifies that dk-always removes the stamp, making ifchange exit 0.
func TestAlways(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	// Stamp the file.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "alw", f1)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Verify it's unchanged first.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "alw", f1)
	if code != 1 {
		t.Fatalf("expected exit 1 (unchanged) before always, got %d", code)
	}

	// Run always to remove the stamp.
	_, _, code = runDkRedo(t, "", "always", "--stamps-dir", stampsDir, "alw")
	if code != 0 {
		t.Fatalf("always failed with exit %d", code)
	}

	// ifchange should now report changed (no stamp).
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "alw", f1)
	if code != 0 {
		t.Errorf("expected exit 0 (changed after always), got %d", code)
	}
}

// TestAlwaysAll verifies that dk-always --all removes all stamp files.
func TestAlwaysAll(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")
	f2 := testutil.WriteTempFile(t, dir, "b.txt", "bbb")

	// Stamp two labels.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "lbl1", f1)
	if code != 0 {
		t.Fatalf("stamp lbl1 failed with exit %d", code)
	}
	_, _, code = runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "lbl2", f2)
	if code != 0 {
		t.Fatalf("stamp lbl2 failed with exit %d", code)
	}

	// Run always --all.
	_, _, code = runDkRedo(t, "", "always", "--stamps-dir", stampsDir, "--all")
	if code != 0 {
		t.Fatalf("always --all failed with exit %d", code)
	}

	// Both labels should now be changed (stamps removed).
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "lbl1", f1)
	if code != 0 {
		t.Errorf("expected exit 0 for lbl1 after always --all, got %d", code)
	}
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "lbl2", f2)
	if code != 0 {
		t.Errorf("expected exit 0 for lbl2 after always --all, got %d", code)
	}
}

// TestCorruptStamp writes garbage to the stamp file and verifies ifchange exits 0.
func TestCorruptStamp(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	// Stamp normally first so the directory is created.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "corrupt", f1)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Overwrite stamp file with garbage (no NUL bytes, but missing tab separator
	// which makes Read return a corrupt-stamp error, leading to exit 2).
	// Also test with NUL bytes which triggers the binary-data check.
	stampFile := filepath.Join(stampsDir, stamp.EscapeLabel("corrupt"))

	// Case 1: garbage with no tab — corrupt stamp error → exit 2 (error).
	if err := os.WriteFile(stampFile, []byte("this is not a valid stamp line"), 0644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "corrupt", f1)
	if code != 2 {
		t.Errorf("case 1 (no-tab garbage): expected exit 2 (error), got %d; stderr: %s", code, stderr)
	}

	// Case 2: NUL byte in stamp — binary data error → exit 2 (error).
	if err := os.WriteFile(stampFile, []byte("path\x00garbage"), 0644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "corrupt", f1)
	if code != 2 {
		t.Errorf("case 2 (NUL byte garbage): expected exit 2 (error), got %d; stderr: %s", code, stderr)
	}

	// Case 3: truncated/empty stamp file — Read returns empty stamp with no files.
	// Compare will detect all current files as "added" → changed → exit 0.
	if err := os.WriteFile(stampFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "corrupt", f1)
	if code != 0 {
		t.Errorf("case 3 (empty stamp): expected exit 0 (changed), got %d", code)
	}
}

// TestSubcommandStyle verifies that "dk-redo ifchange" subcommand style works.
func TestSubcommandStyle(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	// Use subcommand style: dk-redo ifchange ...
	_, _, code := runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "sub", f1)
	if code != 0 {
		t.Errorf("expected exit 0 (first run via subcommand), got %d", code)
	}
}

// TestSymlinkStyle creates a symlink dk-ifchange -> dk-redo and uses it.
func TestSymlinkStyle(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	// Create a symlink dk-ifchange -> dk-redo.
	symlinkPath := filepath.Join(filepath.Dir(binaryPath), "dk-ifchange")
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	defer os.Remove(symlinkPath)

	// Run via the symlink — no "ifchange" subcommand needed.
	_, _, code := testutil.RunBinary(t, symlinkPath, "", "--stamps-dir", stampsDir, "sym", f1)
	if code != 0 {
		t.Errorf("expected exit 0 (first run via symlink), got %d", code)
	}
}

// TestLabelWithSlash verifies that labels with "/" are escaped in filenames.
func TestLabelWithSlash(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	label := "output/config.json"

	// Stamp with slash-containing label.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, label, f1)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Verify the stamp file uses escaped name.
	escapedName := stamp.EscapeLabel(label)
	stampPath := filepath.Join(stampsDir, escapedName)
	if _, err := os.Stat(stampPath); err != nil {
		t.Errorf("expected stamp file at %s (escaped), got error: %v", stampPath, err)
	}
	if escapedName != "output%2Fconfig.json" {
		t.Errorf("expected escaped label output%%2Fconfig.json, got %s", escapedName)
	}

	// ifchange with same label should work.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, label, f1)
	if code != 1 {
		t.Errorf("expected exit 1 (unchanged), got %d", code)
	}
}

// TestForceChanged verifies that dk-ifchange -n always exits 0.
func TestForceChanged(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	f1 := testutil.WriteTempFile(t, dir, "a.txt", "aaa")

	// Stamp the file.
	_, _, code := runDkRedo(t, "", "stamp", "--stamps-dir", stampsDir, "force", f1)
	if code != 0 {
		t.Fatalf("stamp failed with exit %d", code)
	}

	// Without -n, should be unchanged.
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "force", f1)
	if code != 1 {
		t.Fatalf("expected exit 1 (unchanged) without -n, got %d", code)
	}

	// With -n, should always exit 0 (forced changed).
	_, _, code = runDkRedo(t, "", "ifchange", "--stamps-dir", stampsDir, "-n", "force", f1)
	if code != 0 {
		t.Errorf("expected exit 0 (force changed with -n), got %d", code)
	}
}

// TestUnknownSymlink creates a symlink dk-bogus -> dk-redo and verifies exit 2.
func TestUnknownSymlink(t *testing.T) {
	// Create a symlink dk-bogus -> dk-redo.
	symlinkPath := filepath.Join(filepath.Dir(binaryPath), "dk-bogus")
	if err := os.Symlink(binaryPath, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	defer os.Remove(symlinkPath)

	// Run via the symlink — should fail with exit 2 (unknown command).
	_, _, code := testutil.RunBinary(t, symlinkPath, "")
	if code != 2 {
		t.Errorf("expected exit 2 (unknown command), got %d", code)
	}
}
