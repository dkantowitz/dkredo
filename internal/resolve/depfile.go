package resolve

import (
	"fmt"
	"os"
	"strings"
)

// ParseDepfile parses a gcc -MD/-MMD makefile dependency file.
// Returns the dependency paths (everything after the first ':').
func ParseDepfile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("resolve: read depfile %s: %w", path, err)
	}

	content := string(data)
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	// Join continuation lines (backslash + newline)
	content = strings.ReplaceAll(content, "\\\n", " ")
	content = strings.ReplaceAll(content, "\\\r\n", " ")

	var allPaths []string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find the colon that separates targets from dependencies
		colonIdx := findDepColon(line)
		if colonIdx < 0 {
			return nil, fmt.Errorf("resolve: malformed depfile %s: no colon in line %q", path, line)
		}

		deps := line[colonIdx+1:]
		paths := splitDepPaths(deps)
		allPaths = append(allPaths, paths...)
	}

	return allPaths, nil
}

// findDepColon finds the colon separator in a dep line, skipping drive letters on Windows.
func findDepColon(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] == ':' {
			// Skip Windows drive letters like C:
			if i == 1 && len(line) > 2 && (line[2] == '/' || line[2] == '\\') {
				continue
			}
			return i
		}
	}
	return -1
}

// splitDepPaths splits dependency paths, handling escaped spaces.
func splitDepPaths(deps string) []string {
	var paths []string
	var current strings.Builder

	i := 0
	for i < len(deps) {
		ch := deps[i]
		if ch == '\\' && i+1 < len(deps) && deps[i+1] == ' ' {
			current.WriteByte(' ')
			i += 2
			continue
		}
		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				paths = append(paths, current.String())
				current.Reset()
			}
			i++
			continue
		}
		current.WriteByte(ch)
		i++
	}
	if current.Len() > 0 {
		paths = append(paths, current.String())
	}
	return paths
}
