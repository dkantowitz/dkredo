package stamp

import "testing"

func TestNewStampState(t *testing.T) {
	s := NewStampState("my-label")
	if s.Label != "my-label" {
		t.Fatal("label mismatch")
	}
	if len(s.Entries) != 0 {
		t.Fatal("expected empty entries")
	}
	if s.Modified {
		t.Fatal("new state should not be modified")
	}
}

func TestStampStateAddEntry(t *testing.T) {
	s := NewStampState("test")
	added := s.AddEntry("src/main.c", "")
	if !added {
		t.Fatal("expected entry to be added")
	}
	if len(s.Entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if !s.Modified {
		t.Fatal("should be modified after add")
	}
}

func TestStampStateAddDuplicate(t *testing.T) {
	s := NewStampState("test")
	s.AddEntry("src/main.c", "")
	added := s.AddEntry("src/main.c", "")
	if added {
		t.Fatal("duplicate should not be added")
	}
	if len(s.Entries) != 1 {
		t.Fatal("expected 1 entry")
	}
}

func TestStampStateAddPreservesExistingFacts(t *testing.T) {
	s := NewStampState("test")
	s.AddEntry("a.c", "blake3:abc size:100")
	s.AddEntry("b.c", "")
	e := s.FindEntry("a.c")
	if e.Facts != "blake3:abc size:100" {
		t.Fatal("existing facts should be preserved")
	}
}

func TestStampStateFindEntry(t *testing.T) {
	s := NewStampState("test")
	s.AddEntry("src/a.c", "")
	s.AddEntry("src/b.c", "")
	e := s.FindEntry("src/a.c")
	if e == nil || e.Path != "src/a.c" {
		t.Fatal("should find existing entry")
	}
	if s.FindEntry("nonexistent") != nil {
		t.Fatal("should not find nonexistent entry")
	}
}

func TestStampStateRemoveEntry(t *testing.T) {
	s := NewStampState("test")
	s.AddEntry("a.c", "")
	s.AddEntry("b.c", "")
	s.Modified = false
	removed := s.RemoveEntry("a.c")
	if !removed {
		t.Fatal("should have removed entry")
	}
	if len(s.Entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if s.Entries[0].Path != "b.c" {
		t.Fatal("wrong entry remaining")
	}
	if !s.Modified {
		t.Fatal("should be modified after remove")
	}
}

func TestStampStateRemoveNonexistent(t *testing.T) {
	s := NewStampState("test")
	s.AddEntry("a.c", "")
	removed := s.RemoveEntry("nonexistent")
	if removed {
		t.Fatal("should not remove nonexistent")
	}
}

func TestStampStateEntriesSorted(t *testing.T) {
	s := NewStampState("test")
	s.AddEntry("c.c", "")
	s.AddEntry("a.c", "")
	s.AddEntry("b.c", "")
	if s.Entries[0].Path != "a.c" || s.Entries[1].Path != "b.c" || s.Entries[2].Path != "c.c" {
		t.Fatalf("entries not sorted: %v", s.Entries)
	}
}
