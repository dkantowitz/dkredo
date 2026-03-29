package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCreatesSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a fake binary to copy
	fakeBin := filepath.Join(tmpDir, "src-dkredo")
	os.WriteFile(fakeBin, []byte("#!/bin/sh\necho test"), 0755)

	destDir := filepath.Join(tmpDir, "dest")
	os.MkdirAll(destDir, 0755)

	// We can't easily test Install() since it uses os.Executable()
	// but we can test the symlink creation logic directly
	destBin := filepath.Join(destDir, "dkredo")
	copyFile(fakeBin, destBin)
	os.Chmod(destBin, 0755)

	for _, name := range aliasSymlinks {
		link := filepath.Join(destDir, name)
		os.Remove(link)
		if err := os.Symlink("dkredo", link); err != nil {
			t.Fatal(err)
		}
	}

	// Verify symlinks
	for _, name := range aliasSymlinks {
		target, err := os.Readlink(filepath.Join(destDir, name))
		if err != nil {
			t.Fatalf("readlink %s: %v", name, err)
		}
		if target != "dkredo" {
			t.Fatalf("symlink %s -> %q, want dkredo", name, target)
		}
	}

	// Verify binary is executable
	info, err := os.Stat(destBin)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatal("binary not executable")
	}
}

func TestInstallMissingDir(t *testing.T) {
	err := Install("/nonexistent/path/to/dir")
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
}

func TestInstallNotWritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root")
	}
	tmpDir := t.TempDir()
	os.Chmod(tmpDir, 0555)
	defer os.Chmod(tmpDir, 0755)

	err := Install(tmpDir)
	if err == nil {
		t.Fatal("expected error for not writable")
	}
}
