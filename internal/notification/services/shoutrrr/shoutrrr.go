// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package shoutrrr

import (
	"context"

	"github.com/nicholas-fedor/shoutrrr"
	"github.com/rs/zerolog"
)

type MessageSender interface {
	SendMessage(ctx context.Context, message *Message) error
}

type Config struct {
	URL  string
	Name string
}

type Message struct {
	Message string
}

type Client struct {
	log    zerolog.Logger
	config Config
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "shoutrrr").Str("name", config.Name).Logger(),
		config: config,
	}
}

func (c *Client) Name() string {
	return "shoutrrr"
}

// SendMessage delivers the message with shoutrrr, which resolves the service
// from the URL scheme itself and takes no context.
func (c *Client) SendMessage(_ context.Context, message *Message) error {
	if err := shoutrrr.Send(c.config.URL, message.Message); err != nil {
		return err
	}

	c.log.Debug().Msg("notification successfully sent via shoutrrr")

	return nil
}
