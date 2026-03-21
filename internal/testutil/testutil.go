// Package testutil provides shared test helpers for dk-redo tests.
package testutil

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

// WriteTempFile creates a file with the given name and content inside dir.
// It returns the full path to the created file.
// If dir is empty, t.TempDir() is used.
func WriteTempFile(t testing.TB, dir, name, content string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	path := filepath.Join(dir, name)
	// Ensure parent directories exist.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// WriteTempDir creates a directory with the given name inside dir and
// populates it with the provided files (name -> content mapping).
// It returns the full path to the created directory.
// If dir is empty, t.TempDir() is used.
func WriteTempDir(t testing.TB, dir, name string, files map[string]string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	root := filepath.Join(dir, name)
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	for fname, content := range files {
		path := filepath.Join(root, fname)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// RunBinary executes the compiled dk-redo binary with the given arguments.
// The stampsDir is set via the DK_REDO_STAMPS_DIR environment variable
// (if non-empty) so the binary writes stamps to a controlled location.
// It returns stdout, stderr, and the exit code.
func RunBinary(t *testing.T, binary, stampsDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if stampsDir != "" {
		cmd.Env = append(os.Environ(), "DK_REDO_STAMPS_DIR="+stampsDir)
	}

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			t.Fatalf("failed to run binary %s: %v", binary, err)
		}
	}

	return stdout, stderr, exitCode
}
