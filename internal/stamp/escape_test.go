package stamp

import "testing"

func TestEscapeLabel(t *testing.T) {
	tests := []struct{ in, want string }{
		{"firmware.bin", "firmware.bin"},
		{"output/config.json", "output%2Fconfig.json"},
		{"100%done", "100%25done"},
		{"output/100%/file", "output%2F100%25%2Ffile"},
	}
	for _, tt := range tests {
		got := EscapeLabel(tt.in)
		if got != tt.want {
			t.Errorf("EscapeLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
		back := UnescapeLabel(got)
		if back != tt.in {
			t.Errorf("roundtrip failed: %q -> %q -> %q", tt.in, got, back)
		}
	}
}

func TestEncodePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"src/main.c", "src/main.c"},
		{"my file.c", "my file.c"},
		{"dir\tname/file", "dir%09name/file"},
		{"a\nb", "a%0Ab"},
		{"100%/file", "100%25/file"},
	}
	for _, tt := range tests {
		got := EncodePath(tt.in)
		if got != tt.want {
			t.Errorf("EncodePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
		back := DecodePath(got)
		if back != tt.in {
			t.Errorf("roundtrip failed: %q -> %q -> %q", tt.in, got, back)
		}
	}
}
