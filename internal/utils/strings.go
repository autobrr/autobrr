package utils

// StrSliceContains check if slice contains string
func StrSliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
