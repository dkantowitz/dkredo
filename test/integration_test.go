//go:build integration

package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
// pre-built binary path. The stampsDir controls where stamp files are written.
func runDkRedo(t *testing.T, stampsDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	return testutil.RunBinary(t, binaryPath, stampsDir, args...)
}

// TestNothing is a placeholder test to verify the integration test skeleton
// compiles and runs. Implementation tickets will add real tests.
func TestNothing(t *testing.T) {
	// Verify the binary was built successfully.
	if binaryPath == "" {
		t.Fatal("binary was not built")
	}
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("binary not found at %s: %v", binaryPath, err)
	}
}
