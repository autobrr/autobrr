// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
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
	Newznab        *Newznab          `json:"newznab,omitempty"`
	RSS            *FeedSettings     `json:"rss,omitempty"`
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
	Type          string                `json:"type"`
	Parser        string                `json:"parser"`
	ForceSizeUnit string                `json:"forcesizeunit"`
	Lines         []IndexerIRCParseLine `json:"lines"`
	Match         IndexerIRCParseMatch  `json:"match"`
}

type IndexerIRCParseChannel struct {
	Name          string                `json:"name"`
	Parser        string                `json:"parser"`
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

// Helper function
func parseTemplateUrl(baseUrl, sourceUrl string, vars map[string]string, basename string) (string, error) {
	tmpl, err := template.New(basename).Funcs(sprig.TxtFuncMap()).Parse(sourceUrl)
	if err != nil {
		return "", fmt.Errorf("could not create %s template", basename)
	}

	var urlBytes bytes.Buffer
	if err := tmpl.Execute(&urlBytes, &vars); err != nil {
		return "", fmt.Errorf("could not write %s template output", basename)
	}

	templateUrl := urlBytes.String()
	parsedUrl, err := url.Parse(templateUrl)
	if err != nil {
		return "", err
	}

	// for backwards compatibility remove Host and Scheme to rebuild url
	if parsedUrl.Host != "" {
		parsedUrl.Host = ""
	}
	if parsedUrl.Scheme != "" {
		parsedUrl.Scheme = ""
	}

	// join baseURL with query
	baseUrlPath, err := url.JoinPath(baseUrl, parsedUrl.Path)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("could not join %s url", basename))
	}

	// reconstruct url
	finalUrl, _ := url.Parse(baseUrlPath)
	finalUrl.RawQuery = parsedUrl.RawQuery

	return finalUrl.String(), nil
}

func (p *IndexerIRCParseMatch) ParseUrls(baseURL string, vars map[string]string, rls *Release) error {
	// handle url encode of values
	for _, e := range p.Encode {
		if v, ok := vars[e]; ok {
			// url encode  value
			t := url.QueryEscape(v)
			vars[e] = t
		}
	}

	if p.InfoURL != "" {
		infoUrl, err := parseTemplateUrl(baseURL, p.InfoURL, vars, "infourl")
		if err != nil {
			return err
		}

		rls.InfoURL = infoUrl
	}

	if p.TorrentURL != "" {
		downloadUrl, err := parseTemplateUrl(baseURL, p.TorrentURL, vars, "torrenturl")
		if err != nil {
			return err
		}

		rls.TorrentURL = downloadUrl
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

func (p *IndexerIRCParse) Parse(def *IndexerDefinition, vars map[string]string, rls *Release) error {
	// map variables from regex capture onto release struct
	if err := rls.MapVars(def, vars); err != nil {
		return errors.Wrap(err, "could not map vars for release")
	}

	// set baseUrl to default domain
	baseUrl := def.URLS[0]

	// override baseUrl
	if def.BaseURL != "" {
		baseUrl = def.BaseURL
	}

	// merge vars from regex captures on announce and vars from settings
	mergedVars := mergeVars(vars, def.SettingsMap)

	// parse urls
	if err := def.IRC.Parse.Match.ParseUrls(baseUrl, mergedVars, rls); err != nil {
		return errors.Wrap(err, "could not parse urls for release")
	}

	// parse torrentName (AB,MAM)
	if err := def.IRC.Parse.Match.ParseTorrentName(mergedVars, rls); err != nil {
		return errors.Wrap(err, "could not parse torrentName for release")
	}

	var parser IRCParser

	switch def.Identifier {
	case "redacted":
		parser = IRCParserRedacted{}

	case "ops":
		parser = IRCParserOrpheus{}

	case "ggn":
		parser = IRCParserGazelleGames{}

	default:
		parser = IRCParserDefault{}
	}

	if err := parser.Parse(rls, vars); err != nil {
		return errors.Wrap(err, "could not parse release")
	}

	// handle optional cookies
	if v, ok := def.SettingsMap["cookie"]; ok {
		rls.RawCookie = v
	}

	return nil
}

type TorrentBasic struct {
	Id        string `json:"Id"`
	TorrentId string `json:"TorrentId,omitempty"`
	InfoHash  string `json:"InfoHash"`
	Size      string `json:"Size"`
	Uploader  string `json:"Uploader"`
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

type IRCParser interface {
	Parse(rls *Release, vars map[string]string) error
}

type IRCParserDefault struct{}

func (p IRCParserDefault) Parse(rls *Release, _ map[string]string) error {
	// parse fields
	// run before ParseMatch to not potentially use a reconstructed TorrentName
	rls.ParseString(rls.TorrentName)

	return nil
}

// IRCParserRedacted parser for Redacted announces
type IRCParserRedacted struct{}

func (p IRCParserRedacted) Parse(rls *Release, vars map[string]string) error {
	// create new torrentName
	title := vars["title"]
	year := vars["year"]
	category := vars["category"]
	releaseTags := vars["releaseTags"]

	re := regexp.MustCompile(`\| |/ |, `)
	cleanTags := re.ReplaceAllString(releaseTags, "")

	t := ParseReleaseTagString(cleanTags)

	audio := []string{}
	if t.AudioFormat != "" {
		audio = append(audio, t.AudioFormat)
	}
	if t.AudioBitrate != "" {
		audio = append(audio, t.AudioBitrate)
	}
	if t.Source != "" {
		audio = append(audio, t.Source)
	}

	// Name YEAR CD FLAC Lossless
	n := fmt.Sprintf("%s [%s] [%s] (%s)", title, year, category, strings.Join(audio, " "))

	//rls.TorrentName = n

	rls.ParseString(n)
	//rls.Title = title

	return nil
}

type IRCParserOrpheus struct{}

func (p IRCParserOrpheus) Parse(rls *Release, vars map[string]string) error {
	// since OPS uses en-dashes as separators, which causes moistari/rls to not the torrentName properly,
	// we replace the en-dashes with hyphens here
	torrentName := vars["torrentName"]
	rls.TorrentName = strings.ReplaceAll(torrentName, "–", "-")

	rls.ParseString(rls.TorrentName)

	return nil
}

type IRCParserGazelleGames struct{}

func (p IRCParserGazelleGames) Parse(rls *Release, vars map[string]string) error {
	torrentName := vars["torrentName"]
	category := vars["category"]

	releaseName := ""
	title := ""

	switch category {
	case "OST":
		// OST does not have the Title in Group naming convention
		releaseName = torrentName
	default:
		releaseName, title = splitInMiddle(torrentName, " in ")

		if releaseName == "" && title != "" {
			releaseName = torrentName
		}
	}

	rls.ParseString(releaseName)

	if title != "" {
		rls.Title = title
	}

	return nil
}

// mergeVars merge maps
func mergeVars(data ...map[string]string) map[string]string {
	tmpVars := map[string]string{}

	for _, vars := range data {
		// copy vars to new tmp map
		for k, v := range vars {
			tmpVars[k] = v
		}
	}
	return tmpVars
}

func splitInMiddle(s, sep string) (string, string) {
	parts := strings.Split(s, sep)
	l := len(parts)
	return strings.Join(parts[:l/2], sep), strings.Join(parts[l/2:], sep)
}
