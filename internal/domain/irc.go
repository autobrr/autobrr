// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"encoding/json"
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
