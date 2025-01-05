// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type IrcChannel struct {
	ID         int64  `json:"id"`
	Enabled    bool   `json:"enabled"`
	Name       string `json:"name"`
	Password   string `json:"password"`
	Detached   bool   `json:"detached"`
	Monitoring bool   `json:"monitoring"`
}

type IRCAuthMechanism string

const (
	IRCAuthMechanismNone      IRCAuthMechanism = "NONE"
	IRCAuthMechanismSASLPlain IRCAuthMechanism = "SASL_PLAIN"
	IRCAuthMechanismNickServ  IRCAuthMechanism = "NICKSERV"
)

type IRCAuth struct {
	Mechanism IRCAuthMechanism `json:"mechanism,omitempty"`
	Account   string           `json:"account,omitempty"`
	Password  string           `json:"password,omitempty"`
}

type IrcNetwork struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Enabled        bool         `json:"enabled"`
	Server         string       `json:"server"`
	Port           int          `json:"port"`
	TLS            bool         `json:"tls"`
	Pass           string       `json:"pass"`
	Nick           string       `json:"nick"`
	Auth           IRCAuth      `json:"auth,omitempty"`
	InviteCommand  string       `json:"invite_command"`
	UseBouncer     bool         `json:"use_bouncer"`
	BouncerAddr    string       `json:"bouncer_addr"`
	UseProxy       bool         `json:"use_proxy"`
	ProxyId        int64        `json:"proxy_id"`
	Proxy          *Proxy       `json:"proxy"`
	BotMode        bool         `json:"bot_mode"`
	Channels       []IrcChannel `json:"channels"`
	Connected      bool         `json:"connected"`
	ConnectedSince *time.Time   `json:"connected_since"`
}

type IrcNetworkWithHealth struct {
	ID               int64               `json:"id"`
	Name             string              `json:"name"`
	Enabled          bool                `json:"enabled"`
	Server           string              `json:"server"`
	Port             int                 `json:"port"`
	TLS              bool                `json:"tls"`
	Pass             string              `json:"pass"`
	Nick             string              `json:"nick"`
	Auth             IRCAuth             `json:"auth,omitempty"`
	InviteCommand    string              `json:"invite_command"`
	UseBouncer       bool                `json:"use_bouncer"`
	BouncerAddr      string              `json:"bouncer_addr"`
	BotMode          bool                `json:"bot_mode"`
	CurrentNick      string              `json:"current_nick"`
	PreferredNick    string              `json:"preferred_nick"`
	UseProxy         bool                `json:"use_proxy"`
	ProxyId          int64               `json:"proxy_id"`
	Proxy            *Proxy              `json:"proxy"`
	Channels         []ChannelWithHealth `json:"channels"`
	Connected        bool                `json:"connected"`
	ConnectedSince   time.Time           `json:"connected_since"`
	ConnectionErrors []string            `json:"connection_errors"`
	Healthy          bool                `json:"healthy"`
}

type ChannelWithHealth struct {
	ID              int64     `json:"id"`
	Enabled         bool      `json:"enabled"`
	Name            string    `json:"name"`
	Password        string    `json:"password"`
	Detached        bool      `json:"detached"`
	Monitoring      bool      `json:"monitoring"`
	MonitoringSince time.Time `json:"monitoring_since"`
	LastAnnounce    time.Time `json:"last_announce"`
}

type ChannelHealth struct {
	Name            string    `json:"name"`
	Monitoring      bool      `json:"monitoring"`
	MonitoringSince time.Time `json:"monitoring_since"`
	LastAnnounce    time.Time `json:"last_announce"`
}

type IRCManualProcessRequest struct {
	NetworkId int64  `json:"-"`
	Server    string `json:"server"`
	Channel   string `json:"channel"`
	Nick      string `json:"nick"`
	Message   string `json:"msg"`
}

type SendIrcCmdRequest struct {
	NetworkId int64  `json:"network_id"`
	Server    string `json:"server"`
	Channel   string `json:"channel"`
	Nick      string `json:"nick"`
	Message   string `json:"msg"`
}

type IrcMessage struct {
	Channel string    `json:"channel"`
	Nick    string    `json:"nick"`
	Message string    `json:"msg"`
	Time    time.Time `json:"time"`
}

func (m IrcMessage) ToJsonString() string {
	j, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(j)
}

func (m IrcMessage) Bytes() []byte {
	j, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return j
}

type IrcRepo interface {
	StoreNetwork(ctx context.Context, network *IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *IrcNetwork) error
	StoreChannel(ctx context.Context, networkID int64, channel *IrcChannel) error
	UpdateChannel(channel *IrcChannel) error
	UpdateInviteCommand(networkID int64, invite string) error
	StoreNetworkChannels(ctx context.Context, networkID int64, channels []IrcChannel) error
	CheckExistingNetwork(ctx context.Context, network *IrcNetwork) (*IrcNetwork, error)
	FindActiveNetworks(ctx context.Context) ([]IrcNetwork, error)
	ListNetworks(ctx context.Context) ([]IrcNetwork, error)
	ListChannels(networkID int64) ([]IrcChannel, error)
	GetNetworkByID(ctx context.Context, id int64) (*IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
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

type IRCParserOrpheus struct{}

func (p IRCParserOrpheus) replaceSeparator(s string) string {
	return strings.ReplaceAll(s, "–", "-")
}

var lastDecimalTag = regexp.MustCompile(`^\d{1,2}$|^100$`)

func (p IRCParserOrpheus) Parse(rls *Release, vars map[string]string) error {
	// OPS uses en-dashes as separators, which causes moistari/rls to not parse the torrentName properly,
	// we replace the en-dashes with hyphens here
	torrentName := p.replaceSeparator(vars["torrentName"])
	title := p.replaceSeparator(vars["title"])

	year := vars["year"]
	releaseTagsString := vars["releaseTags"]

	splittedTags := strings.Split(releaseTagsString, "/")

	// Check and replace the last tag if it's a number between 0 and 100
	if len(splittedTags) > 0 {
		lastTag := splittedTags[len(splittedTags)-1]
		match := lastDecimalTag.MatchString(lastTag)
		if match {
			splittedTags[len(splittedTags)-1] = lastTag + "%"
		}
	}

	// Join tags back into a string
	releaseTagsString = strings.Join(splittedTags, " ")

	//cleanTags := strings.ReplaceAll(releaseTagsString, "/", " ")
	cleanTags := CleanReleaseTags(releaseTagsString)

	tags := ParseReleaseTagString(cleanTags)
	rls.ReleaseTags = cleanTags

	audio := []string{}
	if tags.Source != "" {
		audio = append(audio, tags.Source)
	}
	if tags.AudioFormat != "" {
		audio = append(audio, tags.AudioFormat)
	}
	if tags.AudioBitrate != "" {
		audio = append(audio, tags.AudioBitrate)
	}
	rls.Bitrate = tags.AudioBitrate
	rls.AudioFormat = tags.AudioFormat

	// set log score even if it's not announced today
	rls.HasLog = tags.HasLog
	rls.LogScore = tags.LogScore
	rls.HasCue = tags.HasCue

	// Construct new release name so we have full control. We remove category such as EP/Single/Album because EP is being mis-parsed.
	torrentName = fmt.Sprintf("%s [%s] (%s)", title, year, strings.Join(audio, " "))

	rls.ParseString(torrentName)

	// use parsed values from raw rls.Release struct
	raw := rls.Raw(torrentName)
	rls.Artists = raw.Artist
	rls.Title = raw.Title

	return nil
}

// IRCParserRedacted parser for Redacted announces
type IRCParserRedacted struct{}

func (p IRCParserRedacted) Parse(rls *Release, vars map[string]string) error {
	title := vars["title"]
	year := vars["year"]
	releaseTagsString := vars["releaseTags"]

	cleanTags := CleanReleaseTags(releaseTagsString)

	tags := ParseReleaseTagString(cleanTags)

	audio := []string{}
	if tags.Source != "" {
		audio = append(audio, tags.Source)
	}
	if tags.AudioFormat != "" {
		audio = append(audio, tags.AudioFormat)
	}
	if tags.AudioBitrate != "" {
		audio = append(audio, tags.AudioBitrate)
	}
	rls.Bitrate = tags.AudioBitrate
	rls.AudioFormat = tags.AudioFormat

	// set log score
	rls.HasLog = tags.HasLog
	rls.LogScore = tags.LogScore
	rls.HasCue = tags.HasCue

	// Construct new release name so we have full control. We remove category such as EP/Single/Album because EP is being mis-parsed.
	name := fmt.Sprintf("%s [%s] (%s)", title, year, strings.Join(audio, " "))

	rls.ParseString(name)

	// use parsed values from raw rls.Release struct
	raw := rls.Raw(name)
	rls.Artists = raw.Artist
	rls.Title = raw.Title

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

// splitInMiddle utility for GGn that tries to split the announced release name
// torrent name consists of "This.Game-GRP in This Game Group" but titles can include "in"
// this function tries to split in the correct place
func splitInMiddle(s, sep string) (string, string) {
	parts := strings.Split(s, sep)
	l := len(parts)
	return strings.Join(parts[:l/2], sep), strings.Join(parts[l/2:], sep)
}
