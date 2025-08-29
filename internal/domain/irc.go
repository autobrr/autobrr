// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"encoding/json"
	"time"
)

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

type IrcChannel struct {
	ID         int64  `json:"id"`
	Enabled    bool   `json:"enabled"`
	Name       string `json:"name"`
	Password   string `json:"password"`
	Detached   bool   `json:"detached"`
	Monitoring bool   `json:"monitoring"`
}

func (ic IrcChannel) MarshalJSON() ([]byte, error) {
	type Alias IrcChannel
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password"`
	}{
		Alias:    (*Alias)(&ic),
		Password: RedactString(ic.Password),
	})
}

func (ic *IrcChannel) UnmarshalJSON(data []byte) error {
	type Alias IrcChannel
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ic),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(ic.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
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

func (ia IRCAuth) MarshalJSON() ([]byte, error) {
	type Alias IRCAuth
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password,omitempty"`
	}{
		Password: RedactString(ia.Password),
		Alias:    (*Alias)(&ia),
	})
}

func (ia *IRCAuth) UnmarshalJSON(data []byte) error {
	type Alias IRCAuth
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ia),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(ia.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
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

func (in IrcNetwork) MarshalJSON() ([]byte, error) {
	type Alias IrcNetwork
	return json.Marshal(&struct {
		*Alias
		Pass string `json:"pass"`
	}{
		Pass:  RedactString(in.Pass),
		Alias: (*Alias)(&in),
	})
}

func (in *IrcNetwork) UnmarshalJSON(data []byte) error {
	type Alias IrcNetwork
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(in),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the pass appears to be redacted, don't overwrite the existing value
	if isRedactedValue(in.Pass) {
		// Keep the original pass by not updating it
		return nil
	}

	return nil
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

func (in IrcNetworkWithHealth) MarshalJSON() ([]byte, error) {
	type Alias IrcNetworkWithHealth
	return json.Marshal(&struct {
		*Alias
		Pass string `json:"pass"`
	}{
		Pass:  RedactString(in.Pass),
		Alias: (*Alias)(&in),
	})
}

func (in *IrcNetworkWithHealth) UnmarshalJSON(data []byte) error {
	type Alias IrcNetworkWithHealth
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(in),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the pass appears to be redacted, don't overwrite the existing value
	if isRedactedValue(in.Pass) {
		// Keep the original pass by not updating it
		return nil
	}

	return nil
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

func (cwh ChannelWithHealth) MarshalJSON() ([]byte, error) {
	type Alias ChannelWithHealth
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password"`
	}{
		Password: RedactString(cwh.Password),
		Alias:    (*Alias)(&cwh),
	})
}

func (cwh *ChannelWithHealth) UnmarshalJSON(data []byte) error {
	type Alias ChannelWithHealth
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(cwh),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(cwh.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
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
