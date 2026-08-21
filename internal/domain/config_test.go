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

	t.Run("auth disabled with empty cors is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "  ",
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "corsAllowedOrigins must be set to explicit origins when authentication is disabled")
	})

	t.Run("fully valid config", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32", "192.168.0.0/16"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
		}
		assert.NoError(t, c.ValidateAuthDisabledConfig())
	})
}
