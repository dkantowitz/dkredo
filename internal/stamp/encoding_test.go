package stamp

import (
	"testing"
	"testing/quick"
)

func TestEscapeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"firmware.bin", "firmware.bin"},
		{"output/config.json", "output%2Fconfig.json"},
		{"100%done", "100%25done"},
		{"a/b%c/d", "a%2Fb%25c%2Fd"},
		{"foo%2Fbar", "foo%252Fbar"}, // literal %2F in label
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EscapeLabel(tt.input)
			if got != tt.want {
				t.Errorf("EscapeLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnescapeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"firmware.bin", "firmware.bin"},
		{"output%2Fconfig.json", "output/config.json"},
		{"100%25done", "100%done"},
		{"foo%252Fbar", "foo%2Fbar"}, // recovers literal %2F
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := UnescapeLabel(tt.input)
			if got != tt.want {
				t.Errorf("UnescapeLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"src/main.c", "src/main.c"},           // slashes fine in paths
		{"dir\tname/file", "dir%09name/file"},   // tab encoded
		{"100%/file", "100%25/file"},             // percent encoded
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := EncodePath(tt.input)
			if got != tt.want {
				t.Errorf("EncodePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"src/main.c", "src/main.c"},
		{"dir%09name/file", "dir\tname/file"},
		{"100%25/file", "100%/file"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DecodePath(tt.input)
			if got != tt.want {
				t.Errorf("DecodePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeLabelRoundtrip(t *testing.T) {
	f := func(s string) bool {
		return UnescapeLabel(EscapeLabel(s)) == s
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestEncodePathRoundtrip(t *testing.T) {
	f := func(s string) bool {
		return DecodePath(EncodePath(s)) == s
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}
