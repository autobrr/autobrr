package domain

import (
	"context"
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

type NickServ struct {
	Account  string `json:"account,omitempty"`
	Password string `json:"password,omitempty"`
}

type IrcNetwork struct {
	ID             int64        `json:"id"`
	Name           string       `json:"name"`
	Enabled        bool         `json:"enabled"`
	Server         string       `json:"server"`
	Port           int          `json:"port"`
	TLS            bool         `json:"tls"`
	Pass           string       `json:"pass"`
	InviteCommand  string       `json:"invite_command"`
	NickServ       NickServ     `json:"nickserv,omitempty"`
	Channels       []IrcChannel `json:"channels"`
	Connected      bool         `json:"connected"`
	ConnectedSince *time.Time   `json:"connected_since"`
}

type IrcNetworkWithHealth struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	Server        string   `json:"server"`
	Port          int      `json:"port"`
	TLS           bool     `json:"tls"`
	Pass          string   `json:"pass"`
	InviteCommand string   `json:"invite_command"`
	NickServ      NickServ `json:"nickserv,omitempty"`
	//Channels       []IrcChannel  `json:"channels"`
	Channels []ChannelWithHealth `json:"channels"`
	//Channels []struct {
	//	IrcChannel
	//	ChannelHealth
	//} `json:"channels"`
	Connected      bool      `json:"connected"`
	ConnectedSince time.Time `json:"connected_since"`
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

type IrcRepo interface {
	StoreNetwork(network *IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *IrcNetwork) error
	StoreChannel(networkID int64, channel *IrcChannel) error
	StoreNetworkChannels(ctx context.Context, networkID int64, channels []IrcChannel) error
	CheckExistingNetwork(ctx context.Context, network *IrcNetwork) (*IrcNetwork, error)
	FindActiveNetworks(ctx context.Context) ([]IrcNetwork, error)
	ListNetworks(ctx context.Context) ([]IrcNetwork, error)
	ListChannels(networkID int64) ([]IrcChannel, error)
	GetNetworkByID(id int64) (*IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
}
