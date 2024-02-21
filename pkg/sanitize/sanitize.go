package sanitize

import "strings"

func String(str string) string {
	str = strings.TrimSpace(str)
	return str
}
