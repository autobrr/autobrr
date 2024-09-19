package sanitize

import (
	"regexp"
	"strings"
)

func String(str string) string {
	str = strings.TrimSpace(str)
	return str
}

var interestingChars = regexp.MustCompile(`[^,\r\n\t\f\v]+`)

func FilterString(str string) string {
	str = String(str)
	str = strings.Join(interestingChars.FindAllString(str, -1), ",")
	for i := 0; i != len(str); {
		i = len(str)
		str = strings.ReplaceAll(str, "  ", " ")
	}

	str = strings.ReplaceAll(str, " ,", ",")
	str = strings.ReplaceAll(str, ", ", ",")
	return str
}
