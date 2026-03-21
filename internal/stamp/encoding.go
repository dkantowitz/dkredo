package stamp

import "strings"

// EscapeLabel converts a label to a safe stamp filename.
// Encodes % as %25 (must be first!), then / as %2F.
func EscapeLabel(label string) string {
	s := strings.ReplaceAll(label, "%", "%25")
	s = strings.ReplaceAll(s, "/", "%2F")
	return s
}

// UnescapeLabel recovers the original label from a stamp filename.
// Decodes %2F to / first, then %25 to % (must be last!).
func UnescapeLabel(escaped string) string {
	s := strings.ReplaceAll(escaped, "%2F", "/")
	s = strings.ReplaceAll(s, "%25", "%")
	return s
}

// EncodePath percent-encodes a path for use in stamp file lines.
// Encodes % as %25 (first!), tab as %09, newline as %0A.
func EncodePath(path string) string {
	s := strings.ReplaceAll(path, "%", "%25")
	s = strings.ReplaceAll(s, "\t", "%09")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// DecodePath recovers the original path from a stamp line.
// Decodes %09 and %0A first, then %25 last.
func DecodePath(encoded string) string {
	s := strings.ReplaceAll(encoded, "%09", "\t")
	s = strings.ReplaceAll(s, "%0A", "\n")
	s = strings.ReplaceAll(s, "%25", "%")
	return s
}
