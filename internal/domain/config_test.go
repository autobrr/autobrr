// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsAuthDisabled(t *testing.T) {
	tests := []struct {
		name                       string
		authDisabled               bool
		iAcknowledgeThisIsABadIdea bool
		want                       bool
	}{
		{"both false", false, false, false},
		{"only authDisabled", true, false, false},
		{"only acknowledge", false, true, false},
		{"both true", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				AuthDisabled:               tt.authDisabled,
				IAcknowledgeThisIsABadIdea: tt.iAcknowledgeThisIsABadIdea,
			}
			assert.Equal(t, tt.want, c.IsAuthDisabled())
		})
	}
}

func TestConfig_ParseAuthDisabledAllowedCIDRs(t *testing.T) {
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
		{"non-zero host bits rejected", []string{"192.168.1.5/24"}, 0, "invalid CIDR in authDisabledAllowedCIDRs: 192.168.1.5/24 has non-zero host bits"},
		{"garbage rejected", []string{"not-a-cidr"}, 0, "invalid IP in authDisabledAllowedCIDRs: not-a-cidr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{AuthDisabledAllowedCIDRs: tt.cidrs}
			prefixes, err := c.ParseAuthDisabledAllowedCIDRs()
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, prefixes, tt.wantLen)
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
			AuthDisabled:               true,
			IAcknowledgeThisIsABadIdea: true,
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "authDisabledAllowedCIDRs is required when authentication is disabled")
	})

	t.Run("auth disabled with invalid CIDR entry", func(t *testing.T) {
		c := &Config{
			AuthDisabled:               true,
			IAcknowledgeThisIsABadIdea: true,
			AuthDisabledAllowedCIDRs:   []string{"not-a-cidr"},
		}
		assert.ErrorContains(t, c.ValidateAuthDisabledConfig(), "invalid IP in authDisabledAllowedCIDRs: not-a-cidr")
	})

	t.Run("auth disabled with oidc enabled is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:               true,
			IAcknowledgeThisIsABadIdea: true,
			AuthDisabledAllowedCIDRs:   []string{"127.0.0.1/32"},
			OIDCEnabled:                true,
		}
		assert.EqualError(t, c.ValidateAuthDisabledConfig(), "authDisabled cannot be used together with oidcEnabled")
	})

	t.Run("fully valid config", func(t *testing.T) {
		c := &Config{
			AuthDisabled:               true,
			IAcknowledgeThisIsABadIdea: true,
			AuthDisabledAllowedCIDRs:   []string{"127.0.0.1/32", "192.168.0.0/16"},
		}
		assert.NoError(t, c.ValidateAuthDisabledConfig())
	})
}
