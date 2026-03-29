package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var aliasSymlinks = []string{
	"dkr-ifchange",
	"dkr-stamp",
	"dkr-always",
	"dkr-fnames",
}

// Install copies the running binary to targetDir and creates alias symlinks.
func Install(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err != nil {
		return fmt.Errorf("target directory does not exist: %s", targetDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("target is not a directory: %s", targetDir)
	}

	// Check writable by attempting to create a temp file
	testFile := filepath.Join(targetDir, ".dkredo-install-test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("target directory not writable: %s", targetDir)
	}
	f.Close()
	os.Remove(testFile)

	// Find the running binary
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find running binary: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path: %w", err)
	}

	// Copy binary
	destBin := filepath.Join(targetDir, "dkredo")
	if err := copyFile(exe, destBin); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	if err := os.Chmod(destBin, 0755); err != nil {
		return fmt.Errorf("chmod binary: %w", err)
	}

	// Create symlinks
	for _, name := range aliasSymlinks {
		link := filepath.Join(targetDir, name)
		os.Remove(link) // remove existing
		if err := os.Symlink("dkredo", link); err != nil {
			return fmt.Errorf("create symlink %s: %w", name, err)
		}
	}

	fmt.Fprintf(os.Stderr, "installed dkredo + %d symlinks to %s\n", len(aliasSymlinks), targetDir)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
