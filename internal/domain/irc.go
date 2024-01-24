// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

func (p IRCParserOrpheus) Parse(rls *Release, vars map[string]string) error {
	// OPS uses en-dashes as separators, which causes moistari/rls to not parse the torrentName properly,
	// we replace the en-dashes with hyphens here
	torrentName := vars["torrentName"]
	rls.TorrentName = strings.ReplaceAll(torrentName, "–", "-")

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
