package releaseinfo

import (
	"reflect"
	"strconv"
	"strings"
)

// ReleaseInfo is the resulting structure returned by Parse
type ReleaseInfo struct {
	Title      string
	Season     int
	Episode    int
	Year       int
	Resolution string
	Source     string
	Codec      string
	Container  string
	Audio      string
	Group      string
	Region     string
	Extended   bool
	Hardcoded  bool
	Proper     bool
	Repack     bool
	Widescreen bool
	Website    string
	Language   string
	Sbs        string
	Unrated    bool
	Size       string
	ThreeD     bool
}

func setField(tor *ReleaseInfo, field, raw, val string) {
	ttor := reflect.TypeOf(tor)
	torV := reflect.ValueOf(tor)
	field = strings.Title(field)
	v, _ := ttor.Elem().FieldByName(field)
	//fmt.Printf("    field=%v, type=%+v, value=%v, raw=%v\n", field, v.Type, val, raw)
	switch v.Type.Kind() {
	case reflect.Bool:
		torV.Elem().FieldByName(field).SetBool(true)
	case reflect.Int:
		clean, _ := strconv.ParseInt(val, 10, 64)
		torV.Elem().FieldByName(field).SetInt(clean)
	case reflect.Uint:
		clean, _ := strconv.ParseUint(val, 10, 64)
		torV.Elem().FieldByName(field).SetUint(clean)
	case reflect.String:
		torV.Elem().FieldByName(field).SetString(val)
	}
}

// Parse breaks up the given filename in TorrentInfo
func Parse(filename string) (*ReleaseInfo, error) {
	tor := &ReleaseInfo{}
	//fmt.Printf("filename %q\n", filename)

	var startIndex, endIndex = 0, len(filename)
	cleanName := strings.Replace(filename, "_", " ", -1)
	for _, pattern := range patterns {
		matches := pattern.re.FindAllStringSubmatch(cleanName, -1)
		if len(matches) == 0 {
			continue
		}
		matchIdx := 0
		if pattern.last {
			// Take last occurrence of element.
			matchIdx = len(matches) - 1
		}
		//fmt.Printf("  %s: pattern:%q match:%#v\n", pattern.name, pattern.re, matches[matchIdx])

		index := strings.Index(cleanName, matches[matchIdx][1])
		if index == 0 {
			startIndex = len(matches[matchIdx][1])
			//fmt.Printf("    startIndex moved to %d [%q]\n", startIndex, filename[startIndex:endIndex])
		} else if index < endIndex {
			endIndex = index
			//fmt.Printf("    endIndex moved to %d [%q]\n", endIndex, filename[startIndex:endIndex])
		}
		setField(tor, pattern.name, matches[matchIdx][1], matches[matchIdx][2])
	}

	// Start process for title
	//fmt.Println("  title: <internal>")
	raw := strings.Split(filename[startIndex:endIndex], "(")[0]
	cleanName = raw
	if strings.HasPrefix(cleanName, "- ") {
		cleanName = raw[2:]
	}
	if strings.ContainsRune(cleanName, '.') && !strings.ContainsRune(cleanName, ' ') {
		cleanName = strings.Replace(cleanName, ".", " ", -1)
	}
	cleanName = strings.Replace(cleanName, "_", " ", -1)
	//cleanName = re.sub('([\[\(_]|- )$', '', cleanName).strip()
	setField(tor, "title", raw, strings.TrimSpace(cleanName))

	return tor, nil
}
