package shellquote

import "strings"

// Quote returns a POSIX shell-safe single-quoted string.
func Quote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
