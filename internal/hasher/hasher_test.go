package hasher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileFactsExistingFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "hello.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, err := FileFacts(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(facts, "blake3:") {
		t.Fatalf("expected blake3 prefix, got %q", facts)
	}
	if !strings.Contains(facts, "size:5") {
		t.Fatalf("expected size:5, got %q", facts)
	}
	// blake3 hex should be 64 chars
	parsed := ParseFacts(facts)
	if len(parsed["blake3"]) != 64 {
		t.Fatalf("blake3 hex should be 64 chars, got %d: %q", len(parsed["blake3"]), parsed["blake3"])
	}
}

func TestFileFactsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "empty")
	os.WriteFile(f, []byte{}, 0644)

	facts, err := FileFacts(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(facts, "size:0") {
		t.Fatalf("expected size:0, got %q", facts)
	}
}

func TestFileFactsMissingFile(t *testing.T) {
	facts, err := FileFacts("/nonexistent/path/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if facts != "missing:true" {
		t.Fatalf("expected missing:true, got %q", facts)
	}
}

func TestFileFactsPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root")
	}
	dir := t.TempDir()
	f := filepath.Join(dir, "noperm")
	os.WriteFile(f, []byte("test"), 0000)
	defer os.Chmod(f, 0644)

	_, err := FileFacts(f)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestFileFactsFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	os.WriteFile(target, []byte("content"), 0644)
	link := filepath.Join(dir, "link")
	os.Symlink(target, link)

	targetFacts, err := FileFacts(target)
	if err != nil {
		t.Fatal(err)
	}
	linkFacts, err := FileFacts(link)
	if err != nil {
		t.Fatal(err)
	}
	if targetFacts != linkFacts {
		t.Fatalf("symlink facts differ: target=%q link=%q", targetFacts, linkFacts)
	}
}

func TestFileFactsDeterministic(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("deterministic"), 0644)

	facts1, _ := FileFacts(f)
	facts2, _ := FileFacts(f)
	if facts1 != facts2 {
		t.Fatalf("not deterministic: %q vs %q", facts1, facts2)
	}
}

func TestParseFacts(t *testing.T) {
	facts := ParseFacts("blake3:abcd1234 size:567")
	if facts["blake3"] != "abcd1234" {
		t.Errorf("blake3 = %q", facts["blake3"])
	}
	if facts["size"] != "567" {
		t.Errorf("size = %q", facts["size"])
	}
}

func TestParseFactsMissing(t *testing.T) {
	facts := ParseFacts("missing:true")
	if facts["missing"] != "true" {
		t.Errorf("missing = %q", facts["missing"])
	}
}

func TestParseFactsUnknownKey(t *testing.T) {
	facts := ParseFacts("blake3:abc size:5 future:xyz")
	if facts["future"] != "xyz" {
		t.Errorf("future = %q", facts["future"])
	}
}

func TestParseFactsEmpty(t *testing.T) {
	facts := ParseFacts("")
	if len(facts) != 0 {
		t.Errorf("expected empty map, got %v", facts)
	}
}

func TestCheckFactUnchanged(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := FileFacts(f)
	changed, _, err := CheckFact(f, facts)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unchanged")
	}
}

func TestCheckFactContentChanged(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := FileFacts(f)
	os.WriteFile(f, []byte("world"), 0644) // same size, different content

	changed, _, err := CheckFact(f, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
}

func TestCheckFactSizeChanged(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("short"), 0644)

	facts, _ := FileFacts(f)
	os.WriteFile(f, []byte("much longer content here"), 0644)

	changed, reason, err := CheckFact(f, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if reason != "size differs" {
		t.Errorf("reason = %q, want 'size differs'", reason)
	}
}

func TestCheckFactFileAppeared(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")

	changed, _, err := CheckFact(f, "missing:true")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unchanged (file still missing)")
	}

	// Now create the file
	os.WriteFile(f, []byte("appeared"), 0644)
	changed, reason, err := CheckFact(f, "missing:true")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed (file appeared)")
	}
	if reason != "file appeared" {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckFactFileDisappeared(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("hello"), 0644)

	facts, _ := FileFacts(f)
	os.Remove(f)

	changed, reason, err := CheckFact(f, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	if reason != "file disappeared" {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckFactUnknownKey(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file")
	os.WriteFile(f, []byte("hello"), 0644)

	changed, reason, err := CheckFact(f, "future:xyz")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed (unknown key)")
	}
	if !strings.Contains(reason, "unknown") {
		t.Errorf("reason = %q", reason)
	}
}

func TestCheckFactNoFacts(t *testing.T) {
	changed, reason, err := CheckFact("/whatever", "")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed (no facts)")
	}
	if reason != "no facts recorded" {
		t.Errorf("reason = %q", reason)
	}
}
