// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIrcNetwork_DetermineIfRestartIsRequired(t *testing.T) {
	t.Parallel()

	socks := &Proxy{ID: 1, Name: "socks", Enabled: true, Type: ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
	socksRenamed := &Proxy{ID: 1, Name: "renamed", Enabled: true, Type: ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
	socksDisabled := &Proxy{ID: 1, Name: "socks", Enabled: false, Type: ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
	socksMoved := &Proxy{ID: 1, Name: "socks", Enabled: true, Type: ProxyTypeSocks5, Addr: "socks5://10.0.0.2:1080"}

	type args struct {
		current IrcNetwork
		desired IrcNetwork
	}
	tests := []struct {
		name        string
		args        args
		wantFields  []string
		wantRestart bool
	}{
		{
			name: "same_proxy",
			args: args{
				current: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socks},
				desired: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socks},
			},
			wantFields:  nil,
			wantRestart: false,
		},
		{
			name: "proxy_renamed",
			args: args{
				current: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socks},
				desired: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socksRenamed},
			},
			wantFields:  nil,
			wantRestart: false,
		},
		{
			name: "proxy_disabled",
			args: args{
				current: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socks},
				desired: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socksDisabled},
			},
			wantFields:  []string{"proxy"},
			wantRestart: true,
		},
		{
			name: "proxy_addr_changed",
			args: args{
				current: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socks},
				desired: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socksMoved},
			},
			wantFields:  []string{"proxy"},
			wantRestart: true,
		},
		{
			name: "proxy_detached",
			args: args{
				current: IrcNetwork{UseProxy: true, ProxyId: 1, Proxy: socks},
				desired: IrcNetwork{UseProxy: false, ProxyId: 0, Proxy: nil},
			},
			wantFields:  []string{"use proxy", "proxy id", "proxy"},
			wantRestart: true,
		},
		{
			name: "leftover_proxy_id_without_use_proxy",
			args: args{
				current: IrcNetwork{ProxyId: 1},
				desired: IrcNetwork{},
			},
			wantFields:  nil,
			wantRestart: false,
		},
		{
			name: "no_proxy_on_either_side",
			args: args{
				current: IrcNetwork{},
				desired: IrcNetwork{},
			},
			wantFields:  nil,
			wantRestart: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, restart := tt.args.current.DetermineIfRestartIsRequired(&tt.args.desired)
			assert.Equal(t, tt.wantFields, fields)
			assert.Equal(t, tt.wantRestart, restart)
		})
	}
}
