package domain

import (
	"bytes"
	"context"
	"net/url"
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"
)

type IndexerRepo interface {
	Store(ctx context.Context, indexer Indexer) (*Indexer, error)
	Update(ctx context.Context, indexer Indexer) (*Indexer, error)
	List(ctx context.Context) ([]Indexer, error)
	Delete(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]Indexer, error)
}

type Indexer struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Identifier     string            `json:"identifier"`
	Enabled        bool              `json:"enabled"`
	Implementation string            `json:"implementation"`
	Settings       map[string]string `json:"settings,omitempty"`
}

type IndexerDefinition struct {
	ID             int               `json:"id,omitempty"`
	Name           string            `json:"name"`
	Identifier     string            `json:"identifier"`
	Implementation string            `json:"implementation"`
	Enabled        bool              `json:"enabled,omitempty"`
	Description    string            `json:"description"`
	Language       string            `json:"language"`
	Privacy        string            `json:"privacy"`
	Protocol       string            `json:"protocol"`
	URLS           []string          `json:"urls"`
	Supports       []string          `json:"supports"`
	Settings       []IndexerSetting  `json:"settings,omitempty"`
	SettingsMap    map[string]string `json:"-"`
	IRC            *IndexerIRC       `json:"irc,omitempty"`
	Torznab        *Torznab          `json:"torznab,omitempty"`
	Parse          *IndexerParse     `json:"parse,omitempty"`
}

func (i IndexerDefinition) HasApi() bool {
	for _, a := range i.Supports {
		if a == "api" {
			return true
		}
	}
	return false
}

type IndexerSetting struct {
	Name        string `json:"name"`
	Required    bool   `json:"required,omitempty"`
	Type        string `json:"type"`
	Value       string `json:"value,omitempty"`
	Label       string `json:"label"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	Help        string `json:"help,omitempty"`
	Regex       string `json:"regex,omitempty"`
}

type Torznab struct {
	MinInterval int              `json:"minInterval"`
	Settings    []IndexerSetting `json:"settings"`
}

type IndexerIRC struct {
	Network     string            `json:"network"`
	Server      string            `json:"server"`
	Port        int               `json:"port"`
	TLS         bool              `json:"tls"`
	Channels    []string          `json:"channels"`
	Announcers  []string          `json:"announcers"`
	SettingsMap map[string]string `json:"-"`
	Settings    []IndexerSetting  `json:"settings"`
}

func (i IndexerIRC) ValidAnnouncer(announcer string) bool {
	for _, a := range i.Announcers {
		if a == announcer {
			return true
		}
	}
	return false
}

func (i IndexerIRC) ValidChannel(channel string) bool {
	for _, a := range i.Channels {
		if a == channel {
			return true
		}
	}
	return false
}

type IndexerParse struct {
	Type          string                `json:"type"`
	ForceSizeUnit string                `json:"forcesizeunit"`
	Lines         []IndexerParseExtract `json:"lines"`
	Match         IndexerParseMatch     `json:"match"`
}

type IndexerParseExtract struct {
	Test    []string `json:"test"`
	Pattern string   `json:"pattern"`
	Vars    []string `json:"vars"`
}

type IndexerParseMatch struct {
	TorrentURL string   `json:"torrenturl"`
	Encode     []string `json:"encode"`
}

func (p *IndexerParse) ParseTorrentUrl(vars map[string]string, extraVars map[string]string, release *Release) error {
	tmpVars := map[string]string{}

	// copy vars to new tmp map
	for k, v := range vars {
		tmpVars[k] = v
	}

	// merge extra vars with vars
	if extraVars != nil {
		for k, v := range extraVars {
			tmpVars[k] = v
		}
	}

	// handle url encode of values
	if p.Match.Encode != nil {
		for _, e := range p.Match.Encode {
			if v, ok := tmpVars[e]; ok {
				// url encode  value
				t := url.QueryEscape(v)
				tmpVars[e] = t
			}
		}
	}

	// setup text template to inject variables into
	tmpl, err := template.New("torrenturl").Parse(p.Match.TorrentURL)
	if err != nil {
		log.Error().Err(err).Msg("could not create torrent url template")
		return err
	}

	var urlBytes bytes.Buffer
	err = tmpl.Execute(&urlBytes, &tmpVars)
	if err != nil {
		log.Error().Err(err).Msg("could not write torrent url template output")
		return err
	}

	release.TorrentURL = urlBytes.String()

	// handle cookies
	if v, ok := extraVars["cookie"]; ok {
		release.RawCookie = v
	}

	return nil
}

type TorrentBasic struct {
	Id        string `json:"Id"`
	TorrentId string `json:"TorrentId,omitempty"`
	InfoHash  string `json:"InfoHash"`
	Size      string `json:"Size"`
}

func (t TorrentBasic) ReleaseSizeBytes() uint64 {
	if t.Size == "" {
		return 0
	}

	releaseSizeBytes, err := humanize.ParseBytes(t.Size)
	if err != nil {
		// log could not parse into bytes
		return 0
	}
	return releaseSizeBytes
}
