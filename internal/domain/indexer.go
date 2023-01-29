package domain

import (
	"bytes"
	"context"
	"net/url"
	"text/template"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
)

type IndexerRepo interface {
	Store(ctx context.Context, indexer Indexer) (*Indexer, error)
	Update(ctx context.Context, indexer Indexer) (*Indexer, error)
	List(ctx context.Context) ([]Indexer, error)
	Delete(ctx context.Context, id int) error
	FindByFilterID(ctx context.Context, id int) ([]Indexer, error)
	FindByID(ctx context.Context, id int) (*Indexer, error)
}

type Indexer struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Identifier     string            `json:"identifier"`
	Enabled        bool              `json:"enabled"`
	Implementation string            `json:"implementation"`
	BaseURL        string            `json:"base_url,omitempty"`
	Settings       map[string]string `json:"settings,omitempty"`
}

type IndexerDefinition struct {
	ID             int               `json:"id,omitempty"`
	Name           string            `json:"name"`
	Identifier     string            `json:"identifier"`
	Implementation string            `json:"implementation"`
	BaseURL        string            `json:"base_url,omitempty"`
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
	RSS            *FeedSettings     `json:"rss,omitempty"`
}

func (i IndexerDefinition) HasApi() bool {
	for _, a := range i.Supports {
		if a == "api" {
			return true
		}
	}
	return false
}

type IndexerDefinitionCustom struct {
	ID             int               `json:"id,omitempty"`
	Name           string            `json:"name"`
	Identifier     string            `json:"identifier"`
	Implementation string            `json:"implementation"`
	BaseURL        string            `json:"base_url,omitempty"`
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
	RSS            *FeedSettings     `json:"rss,omitempty"`
	Parse          *IndexerIRCParse  `json:"parse,omitempty"`
}

func (i *IndexerDefinitionCustom) ToIndexerDefinition() *IndexerDefinition {
	d := &IndexerDefinition{
		ID:             i.ID,
		Name:           i.Name,
		Identifier:     i.Identifier,
		Implementation: i.Implementation,
		BaseURL:        i.BaseURL,
		Enabled:        i.Enabled,
		Description:    i.Description,
		Language:       i.Language,
		Privacy:        i.Privacy,
		Protocol:       i.Protocol,
		URLS:           i.URLS,
		Supports:       i.Supports,
		Settings:       i.Settings,
		SettingsMap:    i.SettingsMap,
		IRC:            i.IRC,
		Torznab:        i.Torznab,
		RSS:            i.RSS,
	}

	if i.IRC != nil && i.Parse != nil {
		i.IRC.Parse = i.Parse
	}

	return d
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

type FeedSettings struct {
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
	Parse       *IndexerIRCParse  `json:"parse,omitempty"`
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

type IndexerIRCParse struct {
	Type          string                `json:"type"`
	ForceSizeUnit string                `json:"forcesizeunit"`
	Lines         []IndexerIRCParseLine `json:"lines"`
	Match         IndexerIRCParseMatch  `json:"match"`
}

type IndexerIRCParseLine struct {
	Test    []string `json:"test"`
	Pattern string   `json:"pattern"`
	Vars    []string `json:"vars"`
	Ignore  bool     `json:"ignore"`
}

type IndexerIRCParseMatch struct {
	TorrentURL  string   `json:"torrenturl"`
	TorrentName string   `json:"torrentname"`
	InfoURL     string   `json:"infourl"`
	Encode      []string `json:"encode"`
}

type IndexerIRCParseMatched struct {
	InfoURL     string
	TorrentURL  string
	TorrentName string
}

func (p *IndexerIRCParse) ParseMatch(baseURL string, vars map[string]string) (*IndexerIRCParseMatched, error) {
	matched := &IndexerIRCParseMatched{}

	// handle url encode of values
	for _, e := range p.Match.Encode {
		if v, ok := vars[e]; ok {
			// url encode  value
			t := url.QueryEscape(v)
			vars[e] = t
		}
	}

	if p.Match.InfoURL != "" {
		// setup text template to inject variables into
		tmpl, err := template.New("infourl").Funcs(sprig.TxtFuncMap()).Parse(p.Match.InfoURL)
		if err != nil {
			return nil, errors.New("could not create info url template")
		}

		var urlBytes bytes.Buffer
		if err := tmpl.Execute(&urlBytes, &vars); err != nil {
			return nil, errors.New("could not write info url template output")
		}

		templateUrl := urlBytes.String()
		parsedUrl, err := url.Parse(templateUrl)
		if err != nil {
			return nil, err
		}

		// for backwards compatibility remove Host and Scheme to rebuild url
		if parsedUrl.Host != "" {
			parsedUrl.Host = ""
		}
		if parsedUrl.Scheme != "" {
			parsedUrl.Scheme = ""
		}

		// join baseURL with query
		baseUrlPath, err := url.JoinPath(baseURL, parsedUrl.Path)
		if err != nil {
			return nil, errors.Wrap(err, "could not join info url")
		}

		// reconstruct url
		infoUrl, _ := url.Parse(baseUrlPath)
		infoUrl.RawQuery = parsedUrl.RawQuery

		matched.InfoURL = infoUrl.String()
	}

	if p.Match.TorrentURL != "" {
		// setup text template to inject variables into
		tmpl, err := template.New("torrenturl").Funcs(sprig.TxtFuncMap()).Parse(p.Match.TorrentURL)
		if err != nil {
			return nil, errors.New("could not create torrent url template")
		}

		var urlBytes bytes.Buffer
		if err := tmpl.Execute(&urlBytes, &vars); err != nil {
			return nil, errors.New("could not write torrent url template output")
		}

		templateUrl := urlBytes.String()
		parsedUrl, err := url.Parse(templateUrl)
		if err != nil {
			return nil, err
		}

		// for backwards compatibility remove Host and Scheme to rebuild url
		if parsedUrl.Host != "" {
			parsedUrl.Host = ""
		}
		if parsedUrl.Scheme != "" {
			parsedUrl.Scheme = ""
		}

		// join baseURL with query
		baseUrlPath, err := url.JoinPath(baseURL, parsedUrl.Path)
		if err != nil {
			return nil, errors.Wrap(err, "could not join torrent url")
		}

		// reconstruct url
		torrentUrl, _ := url.Parse(baseUrlPath)
		torrentUrl.RawQuery = parsedUrl.RawQuery

		matched.TorrentURL = torrentUrl.String()
	}

	if p.Match.TorrentName != "" {
		// setup text template to inject variables into
		tmplName, err := template.New("torrentname").Funcs(sprig.TxtFuncMap()).Parse(p.Match.TorrentName)
		if err != nil {
			return nil, err
		}

		var nameBytes bytes.Buffer
		if err := tmplName.Execute(&nameBytes, &vars); err != nil {
			return nil, errors.New("could not write torrent name template output")
		}

		matched.TorrentName = nameBytes.String()
	}

	return matched, nil
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
