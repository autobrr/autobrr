package domain

import (
	"bytes"
	"strings"
	"text/template"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

type Macro struct {
	TorrentName     string
	TorrentPathName string
	TorrentHash     string
	TorrentUrl      string
	TorrentDataRawBytes	[]byte
	Indexer         string
	Title           string
	Resolution      string
	Source          string
	HDR             string
	FilterName      string
	Size		uint64
	Season          int
	Episode         int
	Year            int
	CurrentYear     int
	CurrentMonth    int
	CurrentDay      int
	CurrentHour     int
	CurrentMinute   int
	CurrentSecond   int
}

func NewMacro(release Release) Macro {
	currentTime := time.Now()

	ma := Macro{
		TorrentName:     release.TorrentName,
		TorrentUrl:      release.TorrentURL,
		TorrentPathName: release.TorrentTmpFile,
		TorrentDataRawBytes: release.TorrentDataRawBytes,
		TorrentHash:     release.TorrentHash,
		Indexer:         release.Indexer,
		Title:           release.Title,
		Resolution:      release.Resolution,
		Source:          release.Source,
		HDR:             strings.Join(release.HDR, ", "),
		FilterName:      release.FilterName,
		Size:            release.Size,
		Season:          release.Season,
		Episode:         release.Episode,
		Year:            release.Year,
		CurrentYear:     currentTime.Year(),
		CurrentMonth:    int(currentTime.Month()),
		CurrentDay:      currentTime.Day(),
		CurrentHour:     currentTime.Hour(),
		CurrentMinute:   currentTime.Minute(),
		CurrentSecond:   currentTime.Second(),
	}

	return ma
}

// Parse takes a string and replaces valid vars
func (m Macro) Parse(text string) (string, error) {

	// setup template
	tmpl, err := template.New("macro").Parse(text)
	if err != nil {
		return "", errors.Wrap(err, "could parse macro template")
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, m)
	if err != nil {
		return "", errors.Wrap(err, "could not parse macro")
	}

	return tpl.String(), nil
}
