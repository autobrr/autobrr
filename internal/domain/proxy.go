// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "context"

type ProxyRepo interface {
	Store(ctx context.Context, p *Proxy) error
	Update(ctx context.Context, p *Proxy) error
	List(ctx context.Context) ([]Proxy, error)
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*Proxy, error)
	ToggleEnabled(ctx context.Context, id int64, enabled bool) error
}

type Proxy struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Addr    string `json:"addr"`
	User    string `json:"user"`
	Pass    string `json:"pass"`
	Timeout int    `json:"timeout"`
}
