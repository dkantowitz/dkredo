package stamp

import "strings"

// EscapeLabel converts a label to a safe filename.
// / → %2F, % → %25
func EscapeLabel(label string) string {
	label = strings.ReplaceAll(label, "%", "%25")
	label = strings.ReplaceAll(label, "/", "%2F")
	return label
}

// UnescapeLabel reverses EscapeLabel.
func UnescapeLabel(filename string) string {
	filename = strings.ReplaceAll(filename, "%2F", "/")
	filename = strings.ReplaceAll(filename, "%25", "%")
	return filename
}

// EncodePath encodes a path for storage in a stamp file.
// \t → %09, \n → %0A, % → %25
func EncodePath(path string) string {
	path = strings.ReplaceAll(path, "%", "%25")
	path = strings.ReplaceAll(path, "\t", "%09")
	path = strings.ReplaceAll(path, "\n", "%0A")
	return path
}

// DecodePath reverses EncodePath.
func DecodePath(encoded string) string {
	encoded = strings.ReplaceAll(encoded, "%09", "\t")
	encoded = strings.ReplaceAll(encoded, "%0A", "\n")
	encoded = strings.ReplaceAll(encoded, "%25", "%")
	return encoded
}
