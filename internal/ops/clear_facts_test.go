package ops

import (
	"testing"

	"dkredo/internal/stamp"
)

func TestClearFactsAll(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.c", "blake3:def size:200")
	state.Modified = false

	err := ClearFacts(state, []string{}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if state.Entries[0].Facts != "" {
		t.Fatalf("a.c facts not cleared: %q", state.Entries[0].Facts)
	}
	if state.Entries[1].Facts != "" {
		t.Fatalf("b.c facts not cleared: %q", state.Entries[1].Facts)
	}
	if state.Entries[0].Path != "a.c" || state.Entries[1].Path != "b.c" {
		t.Fatal("names should be preserved")
	}
	if !state.Modified {
		t.Fatal("should be modified")
	}
}

func TestClearFactsByFilter(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "blake3:abc size:100")
	state.AddEntry("b.h", "blake3:def size:200")

	err := ClearFacts(state, []string{".h"}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	if state.FindEntry("a.c").Facts != "blake3:abc size:100" {
		t.Fatal("a.c facts should be untouched")
	}
	if state.FindEntry("b.h").Facts != "" {
		t.Fatal("b.h facts should be cleared")
	}
}

func TestClearFactsAlreadyEmpty(t *testing.T) {
	state := stamp.NewStampState("test")
	state.AddEntry("a.c", "")
	state.Modified = false

	err := ClearFacts(state, []string{}, nil, "/project", false)
	if err != nil {
		t.Fatal(err)
	}
	// Already empty — Modified should remain false
	if state.Modified {
		t.Fatal("should not be modified when clearing empty facts")
	}
}
