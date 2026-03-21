package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadStdin(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		nullTerminated bool
		want           []string
	}{
		{
			name:           "newline terminated",
			input:          "a.c\nb.c\n",
			nullTerminated: false,
			want:           []string{"a.c", "b.c"},
		},
		{
			name:           "null terminated",
			input:          "a.c\x00b.c\x00",
			nullTerminated: true,
			want:           []string{"a.c", "b.c"},
		},
		{
			name:           "empty input",
			input:          "",
			nullTerminated: false,
			want:           nil,
		},
		{
			name:           "no trailing delimiter",
			input:          "a.c\nb.c",
			nullTerminated: false,
			want:           []string{"a.c", "b.c"},
		},
		{
			name:           "empty lines skipped",
			input:          "a.c\n\nb.c\n",
			nullTerminated: false,
			want:           []string{"a.c", "b.c"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadStdin(strings.NewReader(tc.input), tc.nullTerminated)
			if err != nil {
				t.Fatalf("ReadStdin returned error: %v", err)
			}
			if !sliceEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	t.Run("file args", func(t *testing.T) {
		got, err := Resolve([]string{"src/a.c", "src/b.c"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"src/a.c", "src/b.c"}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("dir arg", func(t *testing.T) {
		tmp := t.TempDir()
		// Create some files in the temp dir
		for _, name := range []string{"c.txt", "a.txt", "b.txt"} {
			if err := os.WriteFile(filepath.Join(tmp, name), []byte("hello"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		got, err := Resolve([]string{tmp}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []string{
			filepath.Join(tmp, "a.txt"),
			filepath.Join(tmp, "b.txt"),
			filepath.Join(tmp, "c.txt"),
		}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("mixed files and dirs", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "z.txt"), []byte("z"), 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := Resolve([]string{"standalone.c", tmp}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := []string{
			filepath.Join(tmp, "z.txt"),
			"standalone.c",
		}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("stdin newline", func(t *testing.T) {
		got, err := Resolve([]string{"-"}, strings.NewReader("x.c\ny.c\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"x.c", "y.c"}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("stdin null", func(t *testing.T) {
		got, err := Resolve([]string{"-0"}, strings.NewReader("x.c\x00y.c\x00"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"x.c", "y.c"}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("mixed with stdin at position", func(t *testing.T) {
		got, err := Resolve([]string{"a.c", "-", "b.c"}, strings.NewReader("x.c\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"a.c", "b.c", "x.c"}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("deduplication", func(t *testing.T) {
		got, err := Resolve([]string{"a.c", "-", "a.c"}, strings.NewReader("a.c\nb.c\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"a.c", "b.c"}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("stdin read only once", func(t *testing.T) {
		got, err := Resolve([]string{"-", "-"}, strings.NewReader("a.c\nb.c\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"a.c", "b.c"}
		if !sliceEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

// sliceEqual compares two string slices, treating nil and empty as equal.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
