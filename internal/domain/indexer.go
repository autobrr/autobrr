// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

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
	GetBy(ctx context.Context, req GetIndexerRequest) (*Indexer, error)
	ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error
}

type Indexer struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	Identifier         string            `json:"identifier"`
	IdentifierExternal string            `json:"identifier_external"`
	Enabled            bool              `json:"enabled"`
	Implementation     string            `json:"implementation"`
	BaseURL            string            `json:"base_url,omitempty"`
	UseProxy           bool              `json:"use_proxy"`
	Proxy              *Proxy            `json:"proxy"`
	ProxyID            int64             `json:"proxy_id"`
	Settings           map[string]string `json:"settings,omitempty"`
}

func (i Indexer) ImplementationIsFeed() bool {
	return i.Implementation == "rss" || i.Implementation == "torznab" || i.Implementation == "newznab"
}

type IndexerMinimal struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Identifier         string `json:"identifier"`
	IdentifierExternal string `json:"identifier_external"`
}

func (m IndexerMinimal) GetExternalIdentifier() string {
	if m.IdentifierExternal != "" {
		return m.IdentifierExternal
	}

	return m.Identifier
}

type IndexerDefinition struct {
	ID                 int               `json:"id,omitempty"`
	Name               string            `json:"name"`
	Identifier         string            `json:"identifier"`
	IdentifierExternal string            `json:"identifier_external"`
	Implementation     string            `json:"implementation"`
	BaseURL            string            `json:"base_url,omitempty"`
	Enabled            bool              `json:"enabled"`
	Description        string            `json:"description"`
	Language           string            `json:"language"`
	Privacy            string            `json:"privacy"`
	Protocol           string            `json:"protocol"`
	URLS               []string          `json:"urls"`
	Supports           []string          `json:"supports"`
	UseProxy           bool              `json:"use_proxy"`
	ProxyID            int64             `json:"proxy_id"`
	Settings           []IndexerSetting  `json:"settings,omitempty"`
	SettingsMap        map[string]string `json:"-"`
	IRC                *IndexerIRC       `json:"irc,omitempty"`
	Torznab            *Torznab          `json:"torznab,omitempty"`
	Newznab            *Newznab          `json:"newznab,omitempty"`
	RSS                *FeedSettings     `json:"rss,omitempty"`
}

type IndexerImplementation string

const (
	IndexerImplementationIRC     IndexerImplementation = "irc"
	IndexerImplementationTorznab IndexerImplementation = "torznab"
	IndexerImplementationNewznab IndexerImplementation = "newznab"
	IndexerImplementationRSS     IndexerImplementation = "rss"
	IndexerImplementationLegacy  IndexerImplementation = ""
)

func (i IndexerImplementation) String() string {
	switch i {
	case IndexerImplementationIRC:
		return "irc"
	case IndexerImplementationTorznab:
		return "torznab"
	case IndexerImplementationNewznab:
		return "newznab"
	case IndexerImplementationRSS:
		return "rss"
	case IndexerImplementationLegacy:
		return ""
	}

	return ""
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
	Newznab        *Newznab          `json:"newznab,omitempty"`
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
		Newznab:        i.Newznab,
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

type Newznab struct {
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
	Type          string                                  `json:"type"`
	ForceSizeUnit string                                  `json:"forcesizeunit"`
	Lines         []IndexerIRCParseLine                   `json:"lines"`
	Match         IndexerIRCParseMatch                    `json:"match"`
	Mappings      map[string]map[string]map[string]string `json:"mappings"`
}

type LineTest struct {
	Line   string            `json:"line"`
	Expect map[string]string `json:"expect"`
}

type IndexerIRCParseLine struct {
	Tests   []LineTest `json:"tests"`
	Pattern string     `json:"pattern"`
	Vars    []string   `json:"vars"`
	Ignore  bool       `json:"ignore"`
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

func parseTemplateURL(baseURL, sourceURL string, vars map[string]string, basename string) (*url.URL, error) {
	// setup text template to inject variables into
	tmpl, err := template.New(basename).Funcs(sprig.TxtFuncMap()).Parse(sourceURL)
	if err != nil {
		return nil, errors.New("could not create %s url template", basename)
	}

	var urlBytes bytes.Buffer
	if err := tmpl.Execute(&urlBytes, &vars); err != nil {
		return nil, errors.New("could not write %s url template output", basename)
	}

	templateUrl := urlBytes.String()
	parsedUrl, err := url.Parse(templateUrl)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse template url: %q", templateUrl)
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
		return nil, errors.Wrap(err, "could not join %s url", basename)
	}

	// reconstruct url
	infoUrl, err := url.Parse(baseUrlPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse %s url", basename)
	}

	infoUrl.RawQuery = parsedUrl.RawQuery

	return infoUrl, nil
}

func (p *IndexerIRCParseMatch) ParseURLs(baseURL string, vars map[string]string, rls *Release) error {
	// handle url encode of values
	for _, e := range p.Encode {
		if v, ok := vars[e]; ok {
			// url encode  value
			t := url.QueryEscape(v)
			vars[e] = t
		}
	}

	if p.InfoURL != "" {
		infoURL, err := parseTemplateURL(baseURL, p.InfoURL, vars, "infourl")
		if err != nil {
			return err
		}

		rls.InfoURL = infoURL.String()
	}

	if p.TorrentURL != "" {
		downloadURL, err := parseTemplateURL(baseURL, p.TorrentURL, vars, "torrenturl")
		if err != nil {
			return err
		}

		rls.DownloadURL = downloadURL.String()
	}

	return nil
}

func (p *IndexerIRCParseMatch) ParseTorrentName(vars map[string]string, rls *Release) error {
	if p.TorrentName != "" {
		// setup text template to inject variables into
		tmplName, err := template.New("torrentname").Funcs(sprig.TxtFuncMap()).Parse(p.TorrentName)
		if err != nil {
			return err
		}

		var nameBytes bytes.Buffer
		if err := tmplName.Execute(&nameBytes, &vars); err != nil {
			return errors.New("could not write torrent name template output")
		}

		rls.TorrentName = nameBytes.String()
	}

	return nil
}

func (p *IndexerIRCParse) MapCustomVariables(vars map[string]string) error {
	for varsKey, varsKeyMap := range p.Mappings {
		varsValue, ok := vars[varsKey]
		if !ok {
			continue
		}

		keyValueMap, ok := varsKeyMap[varsValue]
		if !ok {
			continue
		}

		for k, v := range keyValueMap {
			vars[k] = v
		}
	}

	return nil
}

func (p *IndexerIRCParse) Parse(def *IndexerDefinition, vars map[string]string, rls *Release) error {
	if err := p.MapCustomVariables(vars); err != nil {
		return errors.Wrap(err, "could not map custom variables for release")
	}

	if err := rls.MapVars(def, vars); err != nil {
		return errors.Wrap(err, "could not map variables for release")
	}

	baseUrl := def.BaseURL
	if baseUrl == "" {
		if len(def.URLS) == 0 {
			return errors.New("could not find a valid indexer baseUrl")
		}

		baseUrl = def.URLS[0]
	}

	// merge vars from regex captures on announce and vars from settings
	mergedVars := mergeVars(vars, def.SettingsMap)

	// parse urls
	if err := def.IRC.Parse.Match.ParseURLs(baseUrl, mergedVars, rls); err != nil {
		return errors.Wrap(err, "could not parse urls for release")
	}

	// parse torrent var
	if err := def.IRC.Parse.Match.ParseTorrentName(mergedVars, rls); err != nil {
		return errors.Wrap(err, "could not parse release name")
	}

	var parser IRCParser

	switch def.Identifier {
	case "ggn":
		parser = IRCParserGazelleGames{}
	case "ops":
		parser = IRCParserOrpheus{}
	case "redacted":
		parser = IRCParserRedacted{}
	default:
		parser = IRCParserDefault{}
	}

	if err := parser.Parse(rls, vars); err != nil {
		return errors.Wrap(err, "could not parse release")
	}

	if v, ok := def.SettingsMap["cookie"]; ok {
		rls.RawCookie = v
	}

	return nil
}

type TorrentBasic struct {
	Id          string `json:"Id"`
	TorrentId   string `json:"TorrentId,omitempty"`
	InfoHash    string `json:"InfoHash"`
	Size        string `json:"Size"`
	Uploader    string `json:"Uploader"`
	RecordLabel string `json:"RecordLabel"`
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

type IndexerTestApiRequest struct {
	IndexerId  int    `json:"id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	ApiUser    string `json:"api_user,omitempty"`
	ApiKey     string `json:"api_key"`
}

type GetIndexerRequest struct {
	ID         int
	Identifier string
	Name       string
}
