package resolve

import "dkredo/internal/stamp"

// FilterEntries returns entries matching any of the given filters.
// If filters is empty, returns all entries.
func FilterEntries(entries []stamp.Entry, filters []string) []stamp.Entry {
	if len(filters) == 0 {
		result := make([]stamp.Entry, len(entries))
		copy(result, entries)
		return result
	}
	var result []stamp.Entry
	for _, e := range entries {
		for _, f := range filters {
			if MatchesFilter(e.Path, f) {
				result = append(result, e)
				break
			}
		}
	}
	return result
}
