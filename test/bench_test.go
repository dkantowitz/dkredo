//go:build integration

package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dkantowitz/dk-redo/internal/testutil"
)

// BenchmarkIfchangeUnchanged10 benchmarks dk-ifchange with 10 unchanged files.
// Target: <10ms.
func BenchmarkIfchangeUnchanged10(b *testing.B) {
	dir := b.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	// Create 10 files.
	var files []string
	for i := 0; i < 10; i++ {
		f := testutil.WriteTempFile(b, dir, fmt.Sprintf("file%03d.txt", i), fmt.Sprintf("content-%d", i))
		files = append(files, f)
	}

	// Stamp them.
	args := append([]string{"stamp", "--stamps-dir", stampsDir, "bench10"}, files...)
	runBenchBinary(b, args...)

	// Build ifchange args.
	ifchangeArgs := append([]string{"ifchange", "--stamps-dir", stampsDir, "bench10"}, files...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchBinary(b, ifchangeArgs...)
	}
	b.StopTimer()

	elapsed := b.Elapsed()
	perOp := elapsed / time.Duration(b.N)
	if perOp > 10*time.Millisecond {
		b.Logf("WARNING: per-op time %v exceeds 10ms target", perOp)
	}
}

// BenchmarkIfchangeUnchanged300 benchmarks dk-ifchange with 300 files across
// 10 labels (30 files each). Target: <300ms total.
func BenchmarkIfchangeUnchanged300(b *testing.B) {
	dir := b.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	// Create 10 labels, 30 files each.
	type labelFiles struct {
		label string
		files []string
	}
	var labels []labelFiles
	for l := 0; l < 10; l++ {
		label := fmt.Sprintf("label%02d", l)
		var files []string
		for f := 0; f < 30; f++ {
			path := testutil.WriteTempFile(b, dir, fmt.Sprintf("l%02d/file%03d.txt", l, f), fmt.Sprintf("content-%d-%d", l, f))
			files = append(files, path)
		}
		labels = append(labels, labelFiles{label: label, files: files})

		// Stamp each label.
		args := append([]string{"stamp", "--stamps-dir", stampsDir, label}, files...)
		runBenchBinary(b, args...)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, lf := range labels {
			args := append([]string{"ifchange", "--stamps-dir", stampsDir, lf.label}, lf.files...)
			runBenchBinary(b, args...)
		}
	}
	b.StopTimer()

	elapsed := b.Elapsed()
	perOp := elapsed / time.Duration(b.N)
	if perOp > 300*time.Millisecond {
		b.Logf("WARNING: per-op time %v exceeds 300ms target", perOp)
	}
}

// BenchmarkStamp100 benchmarks dk-stamp with 100 files.
func BenchmarkStamp100(b *testing.B) {
	dir := b.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	// Create 100 files.
	var files []string
	for i := 0; i < 100; i++ {
		f := testutil.WriteTempFile(b, dir, fmt.Sprintf("file%03d.txt", i), fmt.Sprintf("content-%d", i))
		files = append(files, f)
	}

	args := append([]string{"stamp", "--stamps-dir", stampsDir, "bench100"}, files...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchBinary(b, args...)
	}
}

// BenchmarkStartupOverhead benchmarks the startup cost of the dk-redo binary
// with a --help invocation.
func BenchmarkStartupOverhead(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runBenchBinary(b, "--help")
	}
}

// runBenchBinary runs the dk-redo binary for benchmarks.
// It uses the same binaryPath set by TestMain.
func runBenchBinary(b testing.TB, args ...string) {
	b.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Allow non-zero exit codes (e.g. exit 1 for unchanged).
		if _, ok := err.(*exec.ExitError); !ok {
			b.Fatalf("failed to run binary: %v", err)
		}
	}
}
