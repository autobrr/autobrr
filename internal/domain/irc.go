// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"encoding/json"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

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

// IsValid reports whether the mechanism is one of the supported explicit values.
func (m IRCAuthMechanism) IsValid() bool {
	switch m {
	case IRCAuthMechanismNone, IRCAuthMechanismSASLPlain, IRCAuthMechanismNickServ:
		return true
	default:
		return false
	}
}

// Validate rejects unknown authentication mechanisms. Incomplete credentials
// remain accepted so existing networks can still be edited; they simply do not
// enable authentication at runtime.
func (ia IRCAuth) Validate() error {
	if ia.Mechanism == "" {
		return nil
	}

	if !ia.Mechanism.IsValid() {
		return errors.New("invalid IRC authentication mechanism: %q", ia.Mechanism)
	}

	return nil
}

// NickServEnabled reports whether NickServ identification is permitted. SASL
// permits the existing NickServ fallback when negotiation does not complete. An
// empty mechanism retains the historical password-only behavior for database
// rows and API clients created before mechanisms were required.
func (ia IRCAuth) NickServEnabled() bool {
	switch ia.Mechanism {
	case IRCAuthMechanismNickServ:
		return ia.Password != ""
	case IRCAuthMechanismSASLPlain:
		return ia.Account != "" && ia.Password != ""
	case "":
		return ia.Password != ""
	default:
		return false
	}
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

type IrcNetwork struct {
	ID               int64        `json:"id"`
	Name             string       `json:"name"`
	Enabled          bool         `json:"enabled"`
	Server           string       `json:"server"`
	Port             int          `json:"port"`
	TLS              bool         `json:"tls"`
	TLSSkipVerify    bool         `json:"tls_skip_verify"`
	Pass             string       `json:"pass"`
	Nick             string       `json:"nick"`
	Auth             IRCAuth      `json:"auth"`
	InviteCommand    string       `json:"invite_command"`
	UseBouncer       bool         `json:"use_bouncer"`
	BouncerAddr      string       `json:"bouncer_addr"`
	UseProxy         bool         `json:"use_proxy"`
	ProxyId          int64        `json:"proxy_id"`
	Proxy            *Proxy       `json:"proxy"`
	BotMode          bool         `json:"bot_mode"`
	Channels         []IrcChannel `json:"channels"`
	Connected        bool         `json:"connected"`
	ConnectedSince   *time.Time   `json:"connected_since"`
	ConnectionErrors []string     `json:"connection_errors"`
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

// DetermineIfRestartIsRequired diff currentState and desiredState to determine if restart is required to reach the desired state
func (in IrcNetwork) DetermineIfRestartIsRequired(desiredState *IrcNetwork) ([]string, bool) {
	var fieldsChanged []string

	if in.Server != desiredState.Server {
		fieldsChanged = append(fieldsChanged, "server")
	}
	if in.Port != desiredState.Port {
		fieldsChanged = append(fieldsChanged, "port")
	}
	if in.TLS != desiredState.TLS {
		fieldsChanged = append(fieldsChanged, "tls")
	}
	if in.TLSSkipVerify != desiredState.TLSSkipVerify {
		fieldsChanged = append(fieldsChanged, "tls skip verify")
	}
	if in.Pass != desiredState.Pass {
		fieldsChanged = append(fieldsChanged, "pass")
	}
	if in.InviteCommand != desiredState.InviteCommand {
		fieldsChanged = append(fieldsChanged, "invite command")
	}
	if in.UseBouncer != desiredState.UseBouncer {
		fieldsChanged = append(fieldsChanged, "use bouncer")
	}
	if in.BouncerAddr != desiredState.BouncerAddr {
		fieldsChanged = append(fieldsChanged, "bouncer addr")
	}
	if in.BotMode != desiredState.BotMode {
		fieldsChanged = append(fieldsChanged, "bot mode")
	}
	if in.UseProxy != desiredState.UseProxy {
		fieldsChanged = append(fieldsChanged, "use proxy")
	}
	// a leftover id on a network with the toggle off dials nothing, so the id only matters
	// when one side actually uses the proxy
	if in.ProxyId != desiredState.ProxyId && (in.UseProxy || desiredState.UseProxy) {
		fieldsChanged = append(fieldsChanged, "proxy id")
	}
	if proxyDialConfigChanged(in.Proxy, desiredState.Proxy) {
		fieldsChanged = append(fieldsChanged, "proxy")
	}
	if in.Auth.Mechanism != desiredState.Auth.Mechanism {
		fieldsChanged = append(fieldsChanged, "auth mechanism")
	}
	if in.Auth.Account != desiredState.Auth.Account {
		fieldsChanged = append(fieldsChanged, "auth account")
	}
	if in.Auth.Password != desiredState.Auth.Password {
		fieldsChanged = append(fieldsChanged, "auth password")
	}

	return fieldsChanged, len(fieldsChanged) > 0
}

// proxyDialConfigChanged reports whether the attached proxy would dial differently, ignoring
// fields such as the name that do not affect the connection.
func proxyDialConfigChanged(current, desired *Proxy) bool {
	if current == nil || desired == nil {
		return current != desired
	}

	return current.Enabled != desired.Enabled ||
		current.Type != desired.Type ||
		current.Addr != desired.Addr ||
		current.User != desired.User ||
		current.Pass != desired.Pass
}

type IrcNetworkWithHealth struct {
	ID               int64                  `json:"id"`
	Name             string                 `json:"name"`
	Enabled          bool                   `json:"enabled"`
	Server           string                 `json:"server"`
	Port             int                    `json:"port"`
	TLS              bool                   `json:"tls"`
	TLSSkipVerify    bool                   `json:"tls_skip_verify"`
	Pass             string                 `json:"pass"`
	Nick             string                 `json:"nick"`
	Auth             IRCAuth                `json:"auth"`
	InviteCommand    string                 `json:"invite_command"`
	UseBouncer       bool                   `json:"use_bouncer"`
	BouncerAddr      string                 `json:"bouncer_addr"`
	BotMode          bool                   `json:"bot_mode"`
	CurrentNick      string                 `json:"current_nick"`
	PreferredNick    string                 `json:"preferred_nick"`
	UseProxy         bool                   `json:"use_proxy"`
	ProxyId          int64                  `json:"proxy_id"`
	Proxy            *Proxy                 `json:"proxy"`
	Channels         []IrcChannelWithHealth `json:"channels"`
	Connected        bool                   `json:"connected"`
	ConnectedSince   time.Time              `json:"connected_since"`
	ConnectionErrors []string               `json:"connection_errors"`
	Healthy          bool                   `json:"healthy"`
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

type IrcChannelWithHealth struct {
	ID               int64     `json:"id"`
	Enabled          bool      `json:"enabled"`
	Name             string    `json:"name"`
	Password         string    `json:"password"`
	Detached         bool      `json:"detached"`
	State            string    `json:"state"`
	Monitoring       bool      `json:"monitoring"`
	MonitoringSince  time.Time `json:"monitoring_since"`
	LastAnnounce     time.Time `json:"last_announce"`
	ConnectionErrors []string  `json:"connection_errors"`
}

func (cwh IrcChannelWithHealth) MarshalJSON() ([]byte, error) {
	type Alias IrcChannelWithHealth
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password"`
	}{
		Password: RedactString(cwh.Password),
		Alias:    (*Alias)(&cwh),
	})
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
	Network int64     `json:"network"`
	Channel string    `json:"channel"`
	Nick    string    `json:"nick"`
	Message string    `json:"msg"`
	Type    string    `json:"type"`
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
