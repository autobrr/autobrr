package sanitize

import (
	"regexp"
	"strings"
)

func String(str string) string {
	str = strings.TrimSpace(str)
	return str
}

var interestingChars = regexp.MustCompile(`([^,\s]+)`)

func FilterString(str string) string {
	return strings.Join(interestingChars.FindAllString(str, -1), ",")
}
