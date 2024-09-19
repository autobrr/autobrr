package sanitize

import (
	"strings"
)

func String(str string) string {
	str = strings.TrimSpace(str)
	return str
}

func FilterString(str string) string {
	// replace newline with comma
	str = strings.ReplaceAll(str, "\r", ",")
	str = strings.ReplaceAll(str, "\n", ",")
	str = strings.ReplaceAll(str, "\v", ",")
	str = strings.ReplaceAll(str, "\t", " ")
	str = strings.ReplaceAll(str, "\f", "")

	str = repeatedReplaceAll(str, "  ", " ")
	str = repeatedReplaceAll(str, ", ", ",")
	str = repeatedReplaceAll(str, " ,", ",")
	str = repeatedReplaceAll(str, ",,", ",")

	str = strings.Trim(str, ", ")
	return str
}

func repeatedReplaceAll(src, old, new string) string {
	for i := 0; i != len(src); {
		i = len(src)
		src = strings.ReplaceAll(src, old, new)
	}

	return src
}

/*
var interestingChars = regexp.MustCompile(`[^,\r\n\t\f\v]+`)

func FilterString(str string) string {
	str = strings.Join(interestingChars.FindAllString(str, -1), ",")
	for i := 0; i != len(str); {
		i = len(str)
		str = strings.ReplaceAll(str, "  ", " ")
	}

	str = strings.ReplaceAll(str, " ,", ",")
	str = strings.ReplaceAll(str, ", ", ",")
	for i := 0; i != len(str); {
		i = len(str)
		str = strings.ReplaceAll(str, ",,", ",")
	}
	str = strings.Trim(str, ", ")
	return str
}
*/
