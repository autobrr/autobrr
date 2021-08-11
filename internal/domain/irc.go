package domain

import "context"

type IrcChannel struct {
	ID       int64  `json:"id"`
	Enabled  bool   `json:"enabled"`
	Detached bool   `json:"detached"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type SASL struct {
	Mechanism string `json:"mechanism,omitempty"`

	Plain struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	} `json:"plain,omitempty"`
}

type IrcNetwork struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	Enabled         bool         `json:"enabled"`
	Addr            string       `json:"addr"`
	TLS             bool         `json:"tls"`
	Nick            string       `json:"nick"`
	Pass            string       `json:"pass"`
	ConnectCommands []string     `json:"connect_commands"`
	SASL            SASL         `json:"sasl,omitempty"`
	Channels        []IrcChannel `json:"channels"`
}

type IrcRepo interface {
	Store(announce Announce) error
	StoreNetwork(network *IrcNetwork) error
	StoreChannel(networkID int64, channel *IrcChannel) error
	ListNetworks(ctx context.Context) ([]IrcNetwork, error)
	ListChannels(networkID int64) ([]IrcChannel, error)
	GetNetworkByID(id int64) (*IrcNetwork, error)
	DeleteNetwork(ctx context.Context, id int64) error
}
