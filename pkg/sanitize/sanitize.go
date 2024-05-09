package sanitize

import "strings"

func String(str string) string {
	str = strings.TrimSpace(str)
	return str
}

func FilterString(str string) string {
	str = strings.TrimSpace(str)
	str = strings.Trim(str, ",")

	// replace newline with comma
	str = strings.ReplaceAll(str, "\n", ",")
	str = strings.ReplaceAll(str, ",,", ",")

	return str
}
