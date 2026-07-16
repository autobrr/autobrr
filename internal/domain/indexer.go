// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"bytes"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/regexcache"

	"github.com/Masterminds/sprig/v3"
	"github.com/dustin/go-humanize"
)

type Indexer struct {
	ID                 int64                 `json:"id"`
	Name               string                `json:"name"`
	Identifier         string                `json:"identifier"`
	IdentifierExternal string                `json:"identifier_external"`
	Enabled            bool                  `json:"enabled"`
	Implementation     IndexerImplementation `json:"implementation"`
	BaseURL            string                `json:"base_url,omitempty"`
	UseProxy           bool                  `json:"use_proxy"`
	Proxy              *Proxy                `json:"proxy"`
	ProxyID            int64                 `json:"proxy_id"`
	Settings           map[string]string     `json:"settings,omitempty"`
	Archived           bool                  `json:"archived"`
	ArchivedAt         *time.Time            `json:"archived_at,omitempty"`
}

func (i Indexer) MarshalJSON() ([]byte, error) {
	// Define secret keys that should be redacted
	secretKeys := map[string]bool{
		"rsskey":       true,
		"rss_key":      true,
		"passkey":      true,
		"authkey":      true,
		"torrentpass":  true,
		"torrent_pass": true,
		"api_key":      true,
		"apikey":       true,
		"uid":          true,
		"key":          true,
		"token":        true,
		"cookie":       true,
	}

	// Create a copy of the settings map with redacted secrets
	redactedSettings := make(map[string]string)
	for key, value := range i.Settings {
		if secretKeys[strings.ToLower(key)] {
			redactedSettings[key] = RedactString(value)
		} else {
			redactedSettings[key] = value
		}
	}

	// Create alias type to avoid infinite recursion
	type Alias Indexer
	return json.Marshal(&struct {
		*Alias
		Settings map[string]string `json:"settings,omitempty"`
	}{
		Settings: redactedSettings,
		Alias:    (*Alias)(&i),
	})
}

func (i Indexer) ImplementationIsFeed() bool {
	return i.Implementation == IndexerImplementationRSS || i.Implementation == IndexerImplementationTorznab || i.Implementation == IndexerImplementationNewznab
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

// IndexerDeprecation describes an indexer whose definition has been removed.
// The canonical list lives in the embedded registry (internal/indexer/deprecations.go);
// the boot reconcile projects these rows into the indexer_deprecation table so friendly
// names and metadata survive even a hard-deleted indexer row.
type IndexerDeprecation struct {
	Identifier   string    `json:"identifier"`
	Name         string    `json:"name"`
	Reason       string    `json:"reason"`
	IssueURL     string    `json:"issue_url"`
	AliasOf      string    `json:"alias_of,omitempty"`
	DeprecatedAt time.Time `json:"deprecated_at"`
	// FilterCount is how many filters still reference this indexer. Computed on read
	// (not stored / not set by the registry).
	FilterCount int `json:"filter_count"`
}

type IndexerDefinition struct {
	Version            int                   `json:"-"`
	ID                 int                   `json:"id,omitempty"`
	Name               string                `json:"name"`
	Identifier         string                `json:"identifier"`
	IdentifierExternal string                `json:"identifier_external"`
	Implementation     IndexerImplementation `json:"implementation"`
	BaseURL            string                `json:"base_url,omitempty"`
	Enabled            bool                  `json:"enabled"`
	Description        string                `json:"description"`
	Language           string                `json:"language"`
	Privacy            string                `json:"privacy"`
	Protocol           string                `json:"protocol"`
	URLS               []string              `json:"urls"`
	Supports           []string              `json:"supports"`
	UseProxy           bool                  `json:"use_proxy"`
	ProxyID            int64                 `json:"proxy_id"`
	Settings           []IndexerSetting      `json:"settings,omitempty"`
	SettingsMap        map[string]string     `json:"-"`
	IRC                *IndexerIRCV2         `json:"irc,omitempty"`
	Feed               *FeedSettings         `json:"feed,omitempty"`
}

func (i *IndexerDefinition) Prepare() {
	if i.Implementation == IndexerImplementationLegacy {
		i.Implementation = IndexerImplementationIRC
	}

	if i.IRC != nil {
		i.IRC.ChannelsMap = map[string]*IndexerIRCV2Channel{}

		for _, channel := range i.IRC.Channels {
			ch := &IndexerIRCV2Channel{
				Name:       channel.Name,
				Announcers: channel.Announcers,
				Parse:      channel.Parse,
			}

			switch i.Identifier {
			case "ggn":
				ch.Parse.parser = IRCParserGazelleGames{}
			case "ops":
				ch.Parse.parser = IRCParserOrpheus{}
			case "redacted":
				ch.Parse.parser = IRCParserRedacted{}
			default:
				ch.Parse.parser = DefaultIRCParser
			}

			// key by lowercase channel name: the IRC and announce layers
			// canonicalize channel names to lowercase, so lookups use lowercase.
			i.IRC.ChannelsMap[strings.ToLower(channel.Name)] = ch
		}
	}
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
	ID             int                   `json:"id,omitempty"`
	Name           string                `json:"name"`
	Identifier     string                `json:"identifier"`
	Implementation IndexerImplementation `json:"implementation"`
	BaseURL        string                `json:"base_url,omitempty"`
	Enabled        bool                  `json:"enabled,omitempty"`
	Description    string                `json:"description"`
	Language       string                `json:"language"`
	Privacy        string                `json:"privacy"`
	Protocol       string                `json:"protocol"`
	URLS           []string              `json:"urls"`
	Supports       []string              `json:"supports"`
	Settings       []IndexerSetting      `json:"settings,omitempty"`
	SettingsMap    map[string]string     `json:"-"`
	IRC            *IndexerIRC           `json:"irc,omitempty"`
	Feed           *FeedSettings         `json:"feed,omitempty"`
	Parse          *IndexerIRCParse      `json:"parse,omitempty"`
}

func (i *IndexerDefinitionCustom) ToIndexerDefinition() *IndexerDefinition {
	if i.Implementation == IndexerImplementationLegacy {
		i.Implementation = IndexerImplementationIRC
	}

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
	}

	if i.IRC != nil && i.Parse != nil {
		i.IRC.Parse = i.Parse
	}

	if i.IRC != nil {
		d.IRC = &IndexerIRCV2{
			Network:     i.IRC.Network,
			Server:      i.IRC.Server,
			Port:        i.IRC.Port,
			TLS:         i.IRC.TLS,
			SettingsMap: i.IRC.SettingsMap,
			Settings:    i.IRC.Settings,
			Channels:    make([]IndexerIRCV2Channel, 0),
			ChannelsMap: map[string]*IndexerIRCV2Channel{},
		}

		for _, channelName := range i.IRC.Channels {
			channel := IndexerIRCV2Channel{
				Name:       channelName,
				Announcers: i.IRC.Announcers,
				Parse: &IndexerIRCV2Parse{
					Type:             i.IRC.Parse.Type,
					ForceSizeUnit:    i.IRC.Parse.ForceSizeUnit,
					SkipCleanMessage: false,
					Lines:            i.IRC.Parse.Lines,
					Match: IndexerIRCV2ParseMatch{
						DownloadURL: i.IRC.Parse.Match.TorrentURL,
						ReleaseName: i.IRC.Parse.Match.TorrentName,
						MagnetURI:   i.IRC.Parse.Match.MagnetURI,
						InfoURL:     i.IRC.Parse.Match.InfoURL,
						Encode:      i.IRC.Parse.Match.Encode,
					},
					Mappings: i.IRC.Parse.Mappings,
				},
			}

			switch i.Identifier {
			case "ggn":
				channel.Parse.parser = IRCParserGazelleGames{}
			case "ops":
				channel.Parse.parser = IRCParserOrpheus{}
			case "redacted":
				channel.Parse.parser = IRCParserRedacted{}
			default:
				channel.Parse.parser = DefaultIRCParser
			}

			d.IRC.Channels = append(d.IRC.Channels, channel)
			d.IRC.ChannelsMap[strings.ToLower(channelName)] = &channel
		}
	}

	return d
}

type SettingsFieldType string

const (
	SettingsFieldTypeText   SettingsFieldType = "text"
	SettingsFieldTypeSecret SettingsFieldType = "secret"
)

type IndexerSetting struct {
	Name        string            `json:"name"`
	Required    bool              `json:"required,omitempty"`
	Type        SettingsFieldType `json:"type"`
	Value       string            `json:"value,omitempty"`
	Label       string            `json:"label"`
	Default     string            `json:"default,omitempty"`
	Description string            `json:"description,omitempty"`
	Help        string            `json:"help,omitempty"`
	Regex       string            `json:"regex,omitempty"`
}

func (is IndexerSetting) MarshalJSON() ([]byte, error) {
	type Alias IndexerSetting

	redactedValue := is.Value
	if strings.ToLower(string(is.Type)) == "secret" {
		redactedValue = RedactString(is.Value)
	}

	return json.Marshal(&struct {
		*Alias
		Value string `json:"value,omitempty"`
	}{
		Value: redactedValue,
		Alias: (*Alias)(&is),
	})
}

// FeedSettings shared Feed settings for Torznab, Newznab and RSS
type FeedSettings struct {
	MinInterval int              `json:"minInterval"`
	Settings    []IndexerSetting `json:"settings"`
}

/*
Indexer definition v1 / custom / legacy
*/

type IndexerIRCParse struct {
	Type          string                `json:"type"`
	ForceSizeUnit string                `json:"forcesizeunit"`
	Lines         []IndexerIRCParseLine `json:"lines"`
	Match         IndexerIRCParseMatch  `json:"match"`
	Mappings      IRCMappings           `json:"mappings"`
}

type IndexerIRCParseMatch struct {
	TorrentURL  string   `json:"torrenturl"`
	TorrentName string   `json:"torrentname"`
	MagnetURI   string   `json:"magneturi"`
	InfoURL     string   `json:"infourl"`
	Encode      []string `json:"encode"`
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

type IRCMappings map[string]map[string]map[string]string

type IndexerIRCV2 struct {
	Network     string                          `json:"network"`
	Server      string                          `json:"server"`
	Port        int                             `json:"port"`
	TLS         bool                            `json:"tls"`
	SettingsMap map[string]string               `json:"-"`
	Settings    []IndexerSetting                `json:"settings"`
	Channels    []IndexerIRCV2Channel           `json:"channels"`
	ChannelsMap map[string]*IndexerIRCV2Channel `json:"-"`
}

// GetChannel returns the channel config for the given name. The lookup is
// case-insensitive: channel names are canonicalized to lowercase everywhere at
// runtime, so callers may pass any casing.
func (irc *IndexerIRCV2) GetChannel(name string) (*IndexerIRCV2Channel, bool) {
	channel, ok := irc.ChannelsMap[strings.ToLower(name)]
	return channel, ok
}

type IndexerIRCV2Channel struct {
	Name       string             `json:"name"`
	Announcers []string           `json:"announcers"`
	Parse      *IndexerIRCV2Parse `json:"parse,omitempty"`
}

type IndexerIRCV2Parse struct {
	Type             string                 `json:"type"`
	ForceSizeUnit    string                 `json:"forcesizeunit"`
	SkipCleanMessage bool                   `json:"skipcleanmessage"`
	Lines            []IndexerIRCParseLine  `json:"lines"`
	Match            IndexerIRCV2ParseMatch `json:"match"`
	Mappings         IRCMappings            `json:"mappings"`

	parser IRCParser
}

type IndexerIRCV2ParseMatch struct {
	DownloadURL string   `json:"downloadurl"`
	ReleaseName string   `json:"releaseName"`
	MagnetURI   string   `json:"magnetURI"`
	InfoURL     string   `json:"infoURL"`
	Encode      []string `json:"encode"`
}

type LineTest struct {
	Line   string            `json:"line"`
	Expect map[string]string `json:"expect"`
}

type IndexerIRCParseLine struct {
	Pattern string     `json:"pattern"`
	Vars    []string   `json:"vars"`
	Ignore  bool       `json:"ignore"`
	Tests   []LineTest `json:"tests"`
}

func (l IndexerIRCParseLine) ParseLine(tmpVars map[string]string, line string, ignore bool) (bool, error) {
	// an explicit vars list takes precedence and keeps the legacy positional mapping
	if len(l.Vars) > 0 {
		return parseLineExtract(l.Pattern, l.Vars, tmpVars, line)
	}

	// otherwise, if the pattern defines named capture groups, map those directly.
	// this mirrors parseLineExtract's single-match, case-sensitive semantics.
	if rxp, err := regexcache.Compile(l.Pattern); err == nil && hasNamedGroups(rxp) {
		return parseLineExtractNamed(rxp, tmpVars, line)
	}

	return parseLineMatchRegexp(l.Pattern, tmpVars, line, ignore)
}

// hasNamedGroups reports whether the compiled pattern defines any named capture groups.
func hasNamedGroups(rxp *regexp.Regexp) bool {
	for _, name := range rxp.SubexpNames() {
		if name != "" {
			return true
		}
	}
	return false
}

// parseLineExtractNamed maps named capture groups to their values. Unnamed groups are
// skipped, so a pattern can capture without binding a variable.
func parseLineExtractNamed(rxp *regexp.Regexp, tmpVars map[string]string, line string) (bool, error) {
	matches := rxp.FindStringSubmatch(line)
	if matches == nil {
		return false, nil
	}

	for i, name := range rxp.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}

		tmpVars[name] = matches[i]
	}

	return true, nil
}

func parseLineregExMatch(pattern string, value string) ([]string, error) {
	rxp, err := regexcache.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := rxp.FindStringSubmatch(value)
	if matches == nil {
		return nil, nil
	}

	return matches[1:], nil
}

//func ParseLine(pattern string, vars []string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
//	if len(vars) > 0 {
//		return parseExtract(pattern, vars, tmpVars, line)
//	}
//
//	return parseMatchRegexp(pattern, tmpVars, line, ignore)
//}

func parseLineExtract(pattern string, vars []string, tmpVars map[string]string, line string) (bool, error) {
	rxp, err := parseLineregExMatch(pattern, line)
	if err != nil {
		return false, errors.Wrap(ErrUnexpectedLine, "could not match line: %s", line)
	}

	if rxp == nil {
		return false, nil
	}

	for i, v := range vars {
		if i+1 > len(rxp) {
			return false, errors.New("too few matches returned for rxp")
		}

		tmpVars[v] = rxp[i]
	}
	return true, nil
}

func parseLineMatchRegexp(pattern string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
	var re = regexcache.MustCompile(`(?mi)` + pattern)

	groupNames := re.SubexpNames()
	for _, match := range re.FindAllStringSubmatch(line, -1) {
		for groupIdx, group := range match {
			// if line should be ignored then lets return
			if ignore {
				return true, nil
			}

			name := groupNames[groupIdx]
			if name == "" {
				name = "raw"
			}
			tmpVars[name] = group
		}
	}

	return true, nil
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

/*
  indexer definition v2
*/

func (p *IndexerIRCV2ParseMatch) ParseURLs(baseURL string, vars map[string]string, rls *Release) error {
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

	if p.DownloadURL != "" {
		downloadURL, err := parseTemplateURL(baseURL, p.DownloadURL, vars, "downloadurl")
		if err != nil {
			return err
		}

		rls.DownloadURL = downloadURL.String()
	}

	if p.MagnetURI != "" {
		magnetURI, err := parseTemplateURL("magnet:", p.MagnetURI, vars, "magneturi")
		if err != nil {
			return err
		}

		rls.MagnetURI = magnetURI.String()
	}

	return nil
}

func (p *IndexerIRCV2ParseMatch) ParseTorrentName(vars map[string]string, rls *Release) error {
	if p.ReleaseName != "" {
		// setup text template to inject variables into
		tmplName, err := template.New("releasename").Funcs(sprig.TxtFuncMap()).Parse(p.ReleaseName)
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

func ParseReleaseName(releaseNameTemplate string, vars map[string]string, rls *Release) error {
	if releaseNameTemplate == "" {
		return nil
	}

	// setup text template to inject variables into
	tmplName, err := template.New("releasename").Funcs(sprig.TxtFuncMap()).Parse(releaseNameTemplate)
	if err != nil {
		return err
	}

	var nameBytes bytes.Buffer
	if err := tmplName.Execute(&nameBytes, &vars); err != nil {
		return errors.New("could not write torrent name template output")
	}

	rls.TorrentName = nameBytes.String()

	return nil
}

func (p *IndexerIRCV2Parse) MapCustomVariables(vars map[string]string) error {
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

func (p *IndexerIRCV2Parse) Parse(def *IndexerDefinition, channelName string, vars map[string]string, rls *Release) error {
	channel, ok := def.IRC.GetChannel(channelName)
	if !ok {
		return errors.New("could not find channel: %s", channelName)
	}

	if err := p.MapCustomVariables(vars); err != nil {
		return errors.Wrap(err, "could not map custom variables for release")
	}

	if err := rls.MapVars(vars, channel.Parse.ForceSizeUnit); err != nil {
		return errors.Wrap(err, "could not map variables for release")
	}

	// merge vars from regex captures on announce and vars from settings
	mergedVars := mergeVars(vars, def.SettingsMap)

	baseUrl := def.BaseURL
	if baseUrl == "" {
		if len(def.URLS) == 0 {
			return errors.New("could not find a valid indexer baseUrl")
		}

		baseUrl = def.URLS[0]
	}

	// parse urls
	if err := channel.Parse.Match.ParseURLs(baseUrl, mergedVars, rls); err != nil {
		return errors.Wrap(err, "could not parse urls for release")
	}

	// optionally parse release name template
	if err := ParseReleaseName(channel.Parse.Match.ReleaseName, mergedVars, rls); err != nil {
		return errors.Wrap(err, "could not parse release name")
	}

	if err := p.parser.Parse(rls, vars); err != nil {
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
	ProxyID    int64  `json:"proxy_id,omitempty"`
	UseProxy   bool   `json:"use_proxy"`
}

type GetIndexerRequest struct {
	ID         int
	Identifier string
	Name       string
}
