// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsAuthDisabled(t *testing.T) {
	tests := []struct {
		name            string
		authDisabled    bool
		acknowledgement string
		want            bool
	}{
		{"both unset", false, "", false},
		{"only authDisabled", true, "", false},
		{"only acknowledgement", false, AuthDisabledAcknowledgementValue, false},
		{"both set", true, AuthDisabledAcknowledgementValue, true},
		{"wrong acknowledgement value", true, "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				AuthDisabled:                tt.authDisabled,
				AuthDisabledAcknowledgement: tt.acknowledgement,
			}
			assert.Equal(t, tt.want, c.IsAuthDisabled())
		})
	}
}

func TestConfig_ParseAuthAllowedPeerCIDRs(t *testing.T) {
	tests := []struct {
		name       string
		cidrs      []string
		wantLen    int
		wantErrMsg string
	}{
		{"valid CIDR", []string{"192.168.1.0/24"}, 1, ""},
		{"bare IPv4 becomes /32", []string{"127.0.0.1"}, 1, ""},
		{"bare IPv6 becomes /128", []string{"::1"}, 1, ""},
		{"multiple entries", []string{"10.0.0.0/8", "127.0.0.1"}, 2, ""},
		{"empty entries skipped", []string{"", " "}, 0, ""},
		{"IPv4-mapped IPv6 normalized", []string{"::ffff:192.168.1.10"}, 1, ""},
		{"non-zero host bits rejected", []string{"192.168.1.5/24"}, 0, "invalid CIDR in authAllowedPeerCIDRs: 192.168.1.5/24 has non-zero host bits"},
		{"garbage rejected", []string{"not-a-cidr"}, 0, "invalid IP in authAllowedPeerCIDRs: not-a-cidr"},
		{"universal IPv4 prefix rejected", []string{"0.0.0.0/0"}, 0, "invalid CIDR in authAllowedPeerCIDRs: 0.0.0.0/0 matches all addresses"},
		{"universal IPv6 prefix rejected", []string{"::/0"}, 0, "invalid CIDR in authAllowedPeerCIDRs: ::/0 matches all addresses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{AuthAllowedPeerCIDRs: tt.cidrs}
			prefixes, err := c.ParseAuthAllowedPeerCIDRs()
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, prefixes, tt.wantLen)
		})
	}

	t.Run("IPv4-mapped IPv6 entry matches an IPv4 peer", func(t *testing.T) {
		c := &Config{AuthAllowedPeerCIDRs: []string{"::ffff:192.168.1.10"}}
		prefixes, err := c.ParseAuthAllowedPeerCIDRs()
		assert.NoError(t, err)
		assert.Len(t, prefixes, 1)
		assert.True(t, prefixes[0].Addr().Is4())
		assert.True(t, prefixes[0].Contains(netip.MustParseAddr("192.168.1.10")))
	})
}

func TestConfig_ParseTrustedProxies(t *testing.T) {
	c := &Config{TrustedProxies: []string{"10.0.0.0/8", "::1"}}
	prefixes, err := c.ParseTrustedProxies()
	assert.NoError(t, err)
	assert.Len(t, prefixes, 2)

	c = &Config{TrustedProxies: []string{"0.0.0.0/0"}}
	_, err = c.ParseTrustedProxies()
	assert.EqualError(t, err, "invalid CIDR in trustedProxies: 0.0.0.0/0 matches all addresses")
}

func TestConfig_ParseAuthAllowedOrigins(t *testing.T) {
	tests := []struct {
		name       string
		origins    string
		want       []string
		wantErrMsg string
	}{
		{"single origin", "https://autobrr.example.com", []string{"https://autobrr.example.com"}, ""},
		{"origin with port", "http://nas:7474", []string{"http://nas:7474"}, ""},
		{"multiple origins", "https://autobrr.example.com, http://192.168.1.10:7474", []string{"https://autobrr.example.com", "http://192.168.1.10:7474"}, ""},
		{"normalized to lowercase", "https://Autobrr.Example.com", []string{"https://autobrr.example.com"}, ""},
		{"trailing slash dropped", "https://autobrr.example.com/", []string{"https://autobrr.example.com"}, ""},
		{"empty value", "", []string{}, ""},
		{"bare hostname rejected", "autobrr.example.com", nil, "invalid origin in corsAllowedOrigins: autobrr.example.com"},
		{"path rejected", "https://autobrr.example.com/api", nil, "invalid origin in corsAllowedOrigins: https://autobrr.example.com/api"},
		{"wildcard rejected", "*", nil, "invalid origin in corsAllowedOrigins: *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{CorsAllowedOrigins: tt.origins}
			origins, err := c.ParseAuthAllowedOrigins()
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, origins)
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		want    string
		wantErr bool
	}{
		{name: "hostname", host: "autobrr.example.com", want: "autobrr.example.com"},
		{name: "hostname with port", host: "autobrr.example.com:7474", want: "autobrr.example.com"},
		{name: "uppercase and trailing dot", host: "Autobrr.Example.COM.", want: "autobrr.example.com"},
		{name: "single label", host: "autobrr:7474", want: "autobrr"},
		{name: "ipv4 with port", host: "192.168.1.10:7474", want: "192.168.1.10"},
		{name: "bracketed ipv6", host: "[::1]", want: "::1"},
		{name: "bracketed ipv6 with port", host: "[::1]:7474", want: "::1"},
		{name: "ipv4-mapped ipv6", host: "[::ffff:127.0.0.1]:7474", want: "127.0.0.1"},
		{name: "empty", host: "", wantErr: true},
		{name: "empty label", host: "autobrr..example.com", wantErr: true},
		{name: "wildcard", host: "*.example.com", wantErr: true},
		{name: "url", host: "https://autobrr.example.com", wantErr: true},
		{name: "unterminated ipv6", host: "[::1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeHost(tt.host)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_ValidateAuthDisabledConfig(t *testing.T) {
	t.Run("auth not disabled is always valid", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.ValidateAuthDisabledConfig())
	})

	t.Run("auth disabled requires allowlist", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "authAllowedPeerCIDRs is required when authentication is disabled")
	})

	t.Run("auth disabled with only whitespace entries is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"   "},
			CorsAllowedOrigins:          "https://autobrr.example.com",
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "authAllowedPeerCIDRs contains no valid entries")
	})

	t.Run("auth disabled with invalid CIDR entry", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"not-a-cidr"},
		}
		assert.ErrorContains(t, c.ValidateAuthDisabledConfig(), "invalid IP in authAllowedPeerCIDRs: not-a-cidr")
	})

	t.Run("auth disabled with universal CIDR is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"0.0.0.0/0"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
		}
		assert.ErrorContains(t, c.ValidateAuthDisabledConfig(), "matches all addresses")
	})

	t.Run("auth disabled with oidc enabled is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			OIDCEnabled:                 true,
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "authDisabled cannot be used together with oidcEnabled")
	})

	t.Run("auth disabled with wildcard cors is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "*",
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "corsAllowedOrigins must not contain a wildcard \"*\" when authentication is disabled")
	})

	t.Run("auth disabled with wildcard hidden in a cors list is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "*,https://autobrr.example.com",
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "corsAllowedOrigins must not contain a wildcard \"*\" when authentication is disabled")
	})

	t.Run("auth disabled with malformed cors origin is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "autobrr.example.com",
		}
		assert.ErrorContains(t, c.ValidateAuthDisabledConfig(), "invalid origin in corsAllowedOrigins: autobrr.example.com")
	})

	t.Run("auth disabled with empty cors is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "  ",
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "corsAllowedOrigins must be set to explicit origins when authentication is disabled")
	})

	t.Run("auth disabled with invalid allowed host is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
			AuthAllowedHosts:            []string{"https://autobrr.example.com"},
		}
		assert.ErrorContains(t, c.ValidateAuthDisabledConfig(), "invalid host in authAllowedHosts: https://autobrr.example.com")
	})

	t.Run("auth disabled with invalid client ip header is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
			AuthClientIPHeader:          "X Real IP",
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "authClientIPHeader is not a valid header name: X Real IP")
	})

	t.Run("fully valid config", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32", "192.168.0.0/16"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
			AuthAllowedHosts:            []string{"autobrr", "autobrr.lan"},
			AuthClientIPHeader:          "X-Real-IP",
		}
		assert.NoError(t, c.ValidateAuthDisabledConfig())
	})
}
