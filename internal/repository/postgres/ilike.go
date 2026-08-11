package postgres

import "strings"

// escapeILIKE escapes wildcard characters in a string so it can be safely
// embedded in an ILIKE / LIKE pattern with ESCAPE '\'.
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}
