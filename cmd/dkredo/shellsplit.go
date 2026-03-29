package main

// ShellSplit performs POSIX-like shell splitting of a string.
// Handles single quotes, double quotes, and backslash escapes.
func ShellSplit(s string) []string {
	var result []string
	var current []byte
	inSingle := false
	inDouble := false

	i := 0
	for i < len(s) {
		ch := s[i]

		if inSingle {
			if ch == '\'' {
				inSingle = false
			} else {
				current = append(current, ch)
			}
			i++
			continue
		}

		if inDouble {
			if ch == '\\' && i+1 < len(s) {
				next := s[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' || next == '\n' {
					current = append(current, next)
					i += 2
					continue
				}
			}
			if ch == '"' {
				inDouble = false
			} else {
				current = append(current, ch)
			}
			i++
			continue
		}

		if ch == '\\' && i+1 < len(s) {
			current = append(current, s[i+1])
			i += 2
			continue
		}

		if ch == '\'' {
			inSingle = true
			i++
			continue
		}

		if ch == '"' {
			inDouble = true
			i++
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' {
			if len(current) > 0 {
				result = append(result, string(current))
				current = current[:0]
			}
			i++
			continue
		}

		current = append(current, ch)
		i++
	}

	if len(current) > 0 {
		result = append(result, string(current))
	}

	return result
}
