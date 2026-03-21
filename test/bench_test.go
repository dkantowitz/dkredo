//go:build integration

package test

import (
	"testing"
)

// Benchmark skeletons for dk-redo performance regression tests.
// Implementation tickets will fill in the benchmark bodies.
//
// Primary target: 300 files across 10 labels checked in < 300ms.

// BenchmarkIfchangeUnchanged10 benchmarks dk-ifchange with 10 unchanged files.
func BenchmarkIfchangeUnchanged10(b *testing.B) {
	b.Skip("not yet implemented — waiting for ifchange implementation")
}

// BenchmarkIfchangeUnchanged300 benchmarks dk-ifchange with 300 files across
// 10 labels (30 files each). This is the primary regression target: total
// check time should be under 300ms.
func BenchmarkIfchangeUnchanged300(b *testing.B) {
	b.Skip("not yet implemented — waiting for ifchange implementation")
}

// BenchmarkStamp100 benchmarks dk-stamp with 100 small files.
func BenchmarkStamp100(b *testing.B) {
	b.Skip("not yet implemented — waiting for stamp implementation")
}

// BenchmarkStartupOverhead benchmarks the startup cost of the dk-redo binary
// with a no-op invocation.
func BenchmarkStartupOverhead(b *testing.B) {
	b.Skip("not yet implemented — waiting for CLI implementation")
}
