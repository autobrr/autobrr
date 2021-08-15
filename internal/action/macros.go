package action

import (
	"bytes"
	"text/template"
)

type Macro struct {
	TorrentName     string
	TorrentPathName string
	TorrentUrl      string
}

// Parse takes a string and replaces valid vars
func (m Macro) Parse(text string) (string, error) {

	// setup template
	tmpl, err := template.New("macro").Parse(text)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, m)
	if err != nil {
		return "", err
	}

	return tpl.String(), nil
}
