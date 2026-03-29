package main

import "fmt"

// AliasNames lists all valid alias names for error messages.
var AliasNames = []string{"ifchange", "stamp", "always", "fnames"}

// ExpandAlias expands an alias name and its args into +operation args.
// Returns the expanded args (e.g., ["+add-names", "a.c", "+check"]).
func ExpandAlias(alias string, args []string) ([]string, error) {
	switch alias {
	case "ifchange":
		return expandIfchange(args), nil
	case "stamp":
		return expandStamp(args), nil
	case "always":
		return []string{"+clear-facts"}, nil
	case "fnames":
		return expandFnames(args), nil
	default:
		return nil, fmt.Errorf("unknown alias %q (valid: ifchange, stamp, always, fnames)", alias)
	}
}

func expandIfchange(args []string) []string {
	if len(args) == 0 {
		return []string{"+check"}
	}
	result := []string{"+add-names"}
	result = append(result, args...)
	result = append(result, "+check")
	return result
}

func expandStamp(args []string) []string {
	appendMode := false
	fileArgs := args

	if len(args) > 0 && args[0] == "--append" {
		appendMode = true
		fileArgs = args[1:]
	}

	if appendMode {
		result := []string{"+add-names"}
		result = append(result, fileArgs...)
		result = append(result, "+stamp-facts")
		return result
	}

	result := []string{"+remove-names", "+add-names"}
	result = append(result, fileArgs...)
	result = append(result, "+stamp-facts")
	return result
}

func expandFnames(args []string) []string {
	result := []string{"+names", "-e"}
	result = append(result, args...)
	return result
}
