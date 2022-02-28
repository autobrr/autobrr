package action

import (
	"bytes"
	"text/template"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
)

type Macro struct {
	TorrentName     string
	TorrentPathName string
	TorrentHash     string
	TorrentUrl      string
	Indexer         string
	Resolution      string
	Source          string
	HDR             string
	Season          int
	Episode         int
	Year            int
	Month           int
	Day             int
	Hour            int
	Minute          int
	Second          int
}

func NewMacro(release domain.Release) Macro {
	currentTime := time.Now()

	ma := Macro{
		TorrentName:     release.TorrentName,
		TorrentUrl:      release.TorrentURL,
		TorrentPathName: release.TorrentTmpFile,
		TorrentHash:     release.TorrentHash,
		Indexer:         release.Indexer,
		Resolution:      release.Resolution,
		Source:          release.Source,
		HDR:             release.HDR,
		Season:          release.Season,
		Episode:         release.Episode,
		Year:            currentTime.Year(),
		Month:           int(currentTime.Month()),
		Day:             currentTime.Day(),
		Hour:            currentTime.Hour(),
		Minute:          currentTime.Minute(),
		Second:          currentTime.Second(),
	}

	return ma
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
