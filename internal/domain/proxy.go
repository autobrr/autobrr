// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"net/url"

	"github.com/autobrr/autobrr/pkg/errors"
)

type ProxyRepo interface {
	Store(ctx context.Context, p *Proxy) error
	Update(ctx context.Context, p *Proxy) error
	List(ctx context.Context) ([]Proxy, error)
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*Proxy, error)
	ToggleEnabled(ctx context.Context, id int64, enabled bool) error
}

type Proxy struct {
	ID      int64     `json:"id"`
	Name    string    `json:"name"`
	Enabled bool      `json:"enabled"`
	Type    ProxyType `json:"type"`
	Addr    string    `json:"addr"`
	User    string    `json:"user"`
	Pass    string    `json:"pass"`
	Timeout int       `json:"timeout"`
}

type ProxyType string

const (
	ProxyTypeSocks5 = "SOCKS5"
)

func (p Proxy) ValidProxyType() bool {
	if p.Type == ProxyTypeSocks5 {
		return true
	}

	return false
}

func (p Proxy) Validate() error {
	if !p.ValidProxyType() {
		return errors.New("invalid proxy type: %s", p.Type)
	}

	if err := ValidateProxyAddr(p.Addr); err != nil {
		return err
	}

	if p.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

func ValidateProxyAddr(addr string) error {
	if addr == "" {
		return errors.New("addr is required")
	}

	proxyUrl, err := url.Parse(addr)
	if err != nil {
		return errors.Wrap(err, "could not parse proxy url: %s", addr)
	}

	if proxyUrl.Scheme != "socks5" && proxyUrl.Scheme != "socks5h" {
		return errors.New("proxy url scheme must be socks5 or socks5h")
	}

	return nil
}
