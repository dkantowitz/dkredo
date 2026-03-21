package stamp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dkantowitz/dk-redo/internal/hasher"
)

func TestWriteThenReadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "test-label",
		Files: []FileFact{
			{Path: "b.txt", Facts: "blake3:abcd1234 size:100"},
			{Path: "a.txt", Facts: "blake3:efgh5678 size:200"},
		},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(stampsDir, "test-label")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got == nil {
		t.Fatal("Read returned nil")
	}
	if got.Label != "test-label" {
		t.Errorf("Label = %q, want %q", got.Label, "test-label")
	}
	// Write sorts by path, so a.txt should come first
	if len(got.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(got.Files))
	}
	if got.Files[0].Path != "a.txt" {
		t.Errorf("Files[0].Path = %q, want %q", got.Files[0].Path, "a.txt")
	}
	if got.Files[0].Facts != "blake3:efgh5678 size:200" {
		t.Errorf("Files[0].Facts = %q, want %q", got.Files[0].Facts, "blake3:efgh5678 size:200")
	}
	if got.Files[1].Path != "b.txt" {
		t.Errorf("Files[1].Path = %q, want %q", got.Files[1].Path, "b.txt")
	}
}

func TestWriteCreatesStampsDir(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "x",
		Files: []FileFact{{Path: "f.txt", Facts: "missing:true"}},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	info, err := os.Stat(stampsDir)
	if err != nil {
		t.Fatalf("stampsDir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("stampsDir is not a directory")
	}
}

func TestWriteIsAtomic(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "atomic",
		Files: []FileFact{{Path: "f.txt", Facts: "blake3:abc size:10"}},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Verify no temp files remain
	entries, err := os.ReadDir(stampsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".stamp-tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}

	// Verify the stamp file exists with correct name
	stampPath := filepath.Join(stampsDir, "atomic")
	if _, err := os.Stat(stampPath); err != nil {
		t.Errorf("stamp file not found at expected path: %v", err)
	}
}

func TestReadMissingStamp(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	got, err := Read(stampsDir, "nonexistent")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestReadCorruptStamp(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0o755)

	// Write a file with no tab delimiter
	stampPath := filepath.Join(stampsDir, "corrupt")
	os.WriteFile(stampPath, []byte("no-tab-here\n"), 0o644)

	_, err := Read(stampsDir, "corrupt")
	if err == nil {
		t.Fatal("expected error for corrupt stamp")
	}
	if !strings.Contains(err.Error(), "missing tab") {
		t.Errorf("error = %q, want it to mention missing tab", err.Error())
	}
}

func TestReadAdversarialBinaryInput(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0o755)

	// Binary data with NUL bytes
	stampPath := filepath.Join(stampsDir, "binary")
	os.WriteFile(stampPath, []byte("path\t\x00binary\n"), 0o644)

	_, err := Read(stampsDir, "binary")
	if err == nil {
		t.Fatal("expected error for binary input")
	}
	if !strings.Contains(err.Error(), "binary data") {
		t.Errorf("error = %q, want it to mention binary data", err.Error())
	}
}

func TestReadAdversarialLongLine(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0o755)

	// Create a line longer than maxLineLength
	longLine := strings.Repeat("x", maxLineLength+100) + "\tfacts\n"
	stampPath := filepath.Join(stampsDir, "longline")
	os.WriteFile(stampPath, []byte(longLine), 0o644)

	_, err := Read(stampsDir, "longline")
	if err == nil {
		t.Fatal("expected error for very long line")
	}
}

func TestCompareUnchanged(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aabbccdd size:42"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "aabbccdd", Size: 42}},
	}

	result := Compare(s, current)
	if result.Changed {
		t.Errorf("expected Changed=false, got true; ChangedFiles=%+v", result.ChangedFiles)
	}
}

func TestCompareChangedHash(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aabbccdd size:42"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "11223344", Size: 42}},
	}

	result := Compare(s, current)
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if len(result.ChangedFiles) != 1 {
		t.Fatalf("expected 1 changed file, got %d", len(result.ChangedFiles))
	}
	if result.ChangedFiles[0].Reason != "modified" {
		t.Errorf("Reason = %q, want %q", result.ChangedFiles[0].Reason, "modified")
	}
}

func TestCompareChangedFileList(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aaaa size:10"},
			{Path: "b.txt", Facts: "blake3:bbbb size:20"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "aaaa", Size: 10}},
		{Path: "c.txt", Facts: hasher.Facts{Blake3: "cccc", Size: 30}},
	}

	result := Compare(s, current)
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}

	reasons := make(map[string]string)
	for _, cf := range result.ChangedFiles {
		reasons[cf.Path] = cf.Reason
	}

	if reasons["b.txt"] != "removed" {
		t.Errorf("b.txt reason = %q, want %q", reasons["b.txt"], "removed")
	}
	if reasons["c.txt"] != "added" {
		t.Errorf("c.txt reason = %q, want %q", reasons["c.txt"], "added")
	}
}

func TestCompareSizeFastPath(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aabbccdd size:42"},
		},
	}

	// Size differs, hash also differs — but size check should trigger first
	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "different", Size: 100}},
	}

	result := Compare(s, current)
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0].Reason != "modified" {
		t.Errorf("expected modified reason, got %+v", result.ChangedFiles)
	}
}

func TestCompareMissingAppeared(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "missing:true"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "aabb", Size: 10, Missing: false}},
	}

	result := Compare(s, current)
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0].Reason != "appeared" {
		t.Errorf("expected appeared reason, got %+v", result.ChangedFiles)
	}
}

func TestCompareFileDisappeared(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aabb size:10"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Missing: true, Size: -1}},
	}

	result := Compare(s, current)
	if !result.Changed {
		t.Fatal("expected Changed=true")
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0].Reason != "disappeared" {
		t.Errorf("expected disappeared reason, got %+v", result.ChangedFiles)
	}
}

func TestCompareMissingStillMissing(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "missing:true"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Missing: true, Size: -1}},
	}

	result := Compare(s, current)
	if result.Changed {
		t.Errorf("expected Changed=false for still-missing file")
	}
}

func TestCompareUnknownFacts(t *testing.T) {
	s := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aabb futurekey:xyz size:10"},
		},
	}

	current := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "aabb", Size: 10}},
	}

	result := Compare(s, current)
	if !result.Changed {
		t.Fatal("expected Changed=true for unknown facts")
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0].Reason != "unknown_facts" {
		t.Errorf("expected unknown_facts reason, got %+v", result.ChangedFiles)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for unknown facts")
	}
}

func TestAppendMergesNewFiles(t *testing.T) {
	existing := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aaaa size:10"},
		},
	}

	newFacts := []hasher.FileFacts{
		{Path: "b.txt", Facts: hasher.Facts{Blake3: "bbbb", Size: 20}},
	}

	result := Append(existing, newFacts)
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}
	// Should be sorted
	if result.Files[0].Path != "a.txt" || result.Files[1].Path != "b.txt" {
		t.Errorf("unexpected order: %+v", result.Files)
	}
}

func TestAppendUpdatesExistingFacts(t *testing.T) {
	existing := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:old size:10"},
		},
	}

	newFacts := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "new", Size: 99}},
	}

	result := Append(existing, newFacts)
	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	if result.Files[0].Facts != "blake3:new size:99" {
		t.Errorf("Facts = %q, want %q", result.Files[0].Facts, "blake3:new size:99")
	}
}

func TestAppendPreservesUnmentionedFiles(t *testing.T) {
	existing := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aaaa size:10"},
			{Path: "b.txt", Facts: "blake3:bbbb size:20"},
		},
	}

	newFacts := []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "updated", Size: 15}},
	}

	result := Append(existing, newFacts)
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}

	fileMap := make(map[string]string)
	for _, f := range result.Files {
		fileMap[f.Path] = f.Facts
	}

	if fileMap["b.txt"] != "blake3:bbbb size:20" {
		t.Errorf("b.txt facts changed: %q", fileMap["b.txt"])
	}
	if fileMap["a.txt"] != "blake3:updated size:15" {
		t.Errorf("a.txt not updated: %q", fileMap["a.txt"])
	}
}

func TestRoundtripWithSpacesInPaths(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "spaces",
		Files: []FileFact{
			{Path: "path with spaces/file name.txt", Facts: "blake3:1234 size:50"},
		},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(stampsDir, "spaces")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Files[0].Path != "path with spaces/file name.txt" {
		t.Errorf("Path = %q, want %q", got.Files[0].Path, "path with spaces/file name.txt")
	}
}

func TestPathWithTabEncoded(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "tabs",
		Files: []FileFact{
			{Path: "path\twith\ttabs.txt", Facts: "blake3:abcd size:10"},
		},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read raw file to verify encoding
	raw, err := os.ReadFile(filepath.Join(stampsDir, "tabs"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "%09") {
		t.Errorf("raw file should contain %%09, got: %q", string(raw))
	}

	got, err := Read(stampsDir, "tabs")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Files[0].Path != "path\twith\ttabs.txt" {
		t.Errorf("Path = %q, want %q", got.Files[0].Path, "path\twith\ttabs.txt")
	}
}

func TestPathWithPercentEncoded(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "pct",
		Files: []FileFact{
			{Path: "100%done.txt", Facts: "blake3:abcd size:10"},
		},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read raw to verify encoding
	raw, err := os.ReadFile(filepath.Join(stampsDir, "pct"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "%25") {
		t.Errorf("raw file should contain %%25, got: %q", string(raw))
	}

	got, err := Read(stampsDir, "pct")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Files[0].Path != "100%done.txt" {
		t.Errorf("Path = %q, want %q", got.Files[0].Path, "100%done.txt")
	}
}

func TestLabelWithSlash(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")

	s := &Stamp{
		Label: "group/sub-label",
		Files: []FileFact{
			{Path: "f.txt", Facts: "blake3:1234 size:5"},
		},
	}

	if err := Write(stampsDir, s); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// The stamp file should be at .stamps/group%2Fsub-label (flat, not nested)
	expectedFile := filepath.Join(stampsDir, "group%2Fsub-label")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("stamp file not at expected path %s: %v", expectedFile, err)
	}

	got, err := Read(stampsDir, "group/sub-label")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got == nil {
		t.Fatal("Read returned nil")
	}
	if got.Label != "group/sub-label" {
		t.Errorf("Label = %q, want %q", got.Label, "group/sub-label")
	}
}

func TestFormatFacts(t *testing.T) {
	tests := []struct {
		name  string
		facts hasher.Facts
		want  string
	}{
		{
			name:  "existing file",
			facts: hasher.Facts{Blake3: "aabbccdd", Size: 42},
			want:  "blake3:aabbccdd size:42",
		},
		{
			name:  "missing file",
			facts: hasher.Facts{Missing: true, Size: -1},
			want:  "missing:true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFacts(tt.facts)
			if got != tt.want {
				t.Errorf("FormatFacts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppendWithMissingFile(t *testing.T) {
	existing := &Stamp{
		Label: "test",
		Files: []FileFact{
			{Path: "a.txt", Facts: "blake3:aaaa size:10"},
		},
	}

	newFacts := []hasher.FileFacts{
		{Path: "b.txt", Facts: hasher.Facts{Missing: true, Size: -1}},
	}

	result := Append(existing, newFacts)
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(result.Files))
	}

	fileMap := make(map[string]string)
	for _, f := range result.Files {
		fileMap[f.Path] = f.Facts
	}
	if fileMap["b.txt"] != "missing:true" {
		t.Errorf("b.txt = %q, want %q", fileMap["b.txt"], "missing:true")
	}
}

func TestReadEmptyStamp(t *testing.T) {
	dir := t.TempDir()
	stampsDir := filepath.Join(dir, ".stamps")
	os.MkdirAll(stampsDir, 0o755)

	// Write an empty file
	os.WriteFile(filepath.Join(stampsDir, "empty"), []byte(""), 0o644)

	got, err := Read(stampsDir, "empty")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got == nil {
		t.Fatal("Read returned nil for empty stamp")
	}
	if len(got.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(got.Files))
	}
}

func TestAppendPreservesLabel(t *testing.T) {
	existing := &Stamp{
		Label: "my-label",
		Files: []FileFact{},
	}

	result := Append(existing, []hasher.FileFacts{
		{Path: "a.txt", Facts: hasher.Facts{Blake3: "aaaa", Size: 10}},
	})

	if result.Label != "my-label" {
		t.Errorf("Label = %q, want %q", result.Label, "my-label")
	}
}
