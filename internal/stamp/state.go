package stamp

import "sort"

// StampState holds the in-memory representation of a label's stamp file.
type StampState struct {
	Label    string
	Entries  []Entry
	Modified bool
}

// Entry represents one file in the stamp.
type Entry struct {
	Path  string
	Facts string // raw fact string, empty if no facts computed
}

// NewStampState creates a new empty stamp state for the given label.
func NewStampState(label string) *StampState {
	return &StampState{
		Label:   label,
		Entries: nil,
	}
}

// AddEntry adds a file to the stamp if not already present.
// Returns true if the entry was added (new), false if it already existed.
func (s *StampState) AddEntry(path, facts string) bool {
	if s.FindEntry(path) != nil {
		return false
	}
	s.Entries = append(s.Entries, Entry{Path: path, Facts: facts})
	s.sortEntries()
	s.Modified = true
	return true
}

// FindEntry returns a pointer to the entry with the given path, or nil.
func (s *StampState) FindEntry(path string) *Entry {
	for i := range s.Entries {
		if s.Entries[i].Path == path {
			return &s.Entries[i]
		}
	}
	return nil
}

// RemoveEntry removes the entry with the given path.
// Returns true if an entry was removed.
func (s *StampState) RemoveEntry(path string) bool {
	for i, e := range s.Entries {
		if e.Path == path {
			s.Entries = append(s.Entries[:i], s.Entries[i+1:]...)
			s.Modified = true
			return true
		}
	}
	return false
}

func (s *StampState) sortEntries() {
	sort.Slice(s.Entries, func(i, j int) bool {
		return s.Entries[i].Path < s.Entries[j].Path
	})
}
