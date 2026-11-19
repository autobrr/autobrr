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
		{"default https port dropped", "https://autobrr.example.com:443", []string{"https://autobrr.example.com"}, ""},
		{"default http port dropped", "http://nas:80", []string{"http://nas"}, ""},
		{"default port of other scheme kept", "https://autobrr.example.com:80", []string{"https://autobrr.example.com:80"}, ""},
		{"default port dropped from ipv6", "http://[::1]:80", []string{"http://[::1]"}, ""},
		{"ipv6 literal with port", "https://[::1]:7474", []string{"https://[::1]:7474"}, ""},
		{"ipv6 literal canonicalized", "https://[0:0:0:0:0:0:0:1]:7474", []string{"https://[::1]:7474"}, ""},
		{"ipv6 literal lowercased and compressed", "http://[FD00:0:0:0:0:0:0:A]", []string{"http://[fd00::a]"}, ""},
		{"empty value", "", []string{}, ""},
		{"bare hostname rejected", "autobrr.example.com", nil, "invalid origin in corsAllowedOrigins: autobrr.example.com"},
		{"path rejected", "https://autobrr.example.com/api", nil, "invalid origin in corsAllowedOrigins: https://autobrr.example.com/api"},
		{"userinfo rejected", "https://admin@autobrr.example.com", nil, "invalid origin in corsAllowedOrigins: https://admin@autobrr.example.com"},
		{"wildcard rejected", "*", nil, "wildcard origin in corsAllowedOrigins is not supported: *"},
		{"wildcard subdomain rejected", "https://*.example.com", nil, "wildcard origin in corsAllowedOrigins is not supported: https://*.example.com"},
		{"port out of range rejected", "https://autobrr.example.com:99999", nil, "invalid host in corsAllowedOrigins origin: https://autobrr.example.com:99999: invalid port: 99999"},
		{"non-numeric port rejected", "https://autobrr.example.com:abc", nil, "invalid origin in corsAllowedOrigins: https://autobrr.example.com:abc"},
		{"invalid hostname character rejected", "https://autobrr_example.com!", nil, "invalid host in corsAllowedOrigins origin: https://autobrr_example.com!"},
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

func TestConfig_ValidateAuthConfig(t *testing.T) {
	t.Run("builtin mode without trusted proxies is valid", func(t *testing.T) {
		c := &Config{}
		assert.NoError(t, c.ValidateAuthConfig())
	})

	t.Run("builtin mode with invalid trusted proxy is rejected", func(t *testing.T) {
		c := &Config{TrustedProxies: []string{"0.0.0.0/0"}}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "invalid trustedProxies")
	})

	t.Run("oidc mode with invalid trusted proxy is rejected", func(t *testing.T) {
		c := &Config{OIDCEnabled: true, TrustedProxies: []string{"not-a-cidr"}}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "invalid trustedProxies")
	})

	t.Run("unacknowledged auth disabled is validated as builtin", func(t *testing.T) {
		c := &Config{AuthDisabled: true}
		assert.NoError(t, c.ValidateAuthConfig())
	})

	t.Run("auth disabled ignores trusted proxies", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
			TrustedProxies:              []string{"not-a-cidr"},
		}
		assert.NoError(t, c.ValidateAuthConfig())
	})

	t.Run("auth disabled requires allowlist", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "authAllowedPeerCIDRs is required in disabled authentication mode")
	})

	t.Run("auth disabled with only whitespace entries is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"   "},
			CorsAllowedOrigins:          "https://autobrr.example.com",
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "authAllowedPeerCIDRs contains no valid entries")
	})

	t.Run("auth disabled with invalid CIDR entry", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"not-a-cidr"},
		}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "invalid IP in authAllowedPeerCIDRs: not-a-cidr")
	})

	t.Run("auth disabled with universal CIDR is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"0.0.0.0/0"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
		}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "matches all addresses")
	})

	t.Run("auth disabled with oidc enabled is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			OIDCEnabled:                 true,
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "authDisabled cannot be used together with oidcEnabled")
	})

	t.Run("auth disabled with wildcard cors is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "*",
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "wildcard origin in corsAllowedOrigins is not supported: *")
	})

	t.Run("auth disabled with wildcard hidden in a cors list is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com,*",
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "wildcard origin in corsAllowedOrigins is not supported: *")
	})

	t.Run("auth disabled with wildcard subdomain cors origin is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://*.example.com",
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "wildcard origin in corsAllowedOrigins is not supported: https://*.example.com")
	})

	t.Run("auth disabled with out-of-range cors origin port is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com:99999",
		}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "invalid host in corsAllowedOrigins origin: https://autobrr.example.com:99999")
	})

	t.Run("auth disabled with malformed cors origin is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "autobrr.example.com",
		}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "invalid origin in corsAllowedOrigins: autobrr.example.com")
	})

	t.Run("auth disabled with empty cors is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "  ",
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "corsAllowedOrigins must be set to explicit origins in disabled authentication mode")
	})

	t.Run("auth disabled with invalid allowed host is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
			AuthAllowedHosts:            []string{"https://autobrr.example.com"},
		}
		assert.ErrorContains(t, c.ValidateAuthConfig(), "invalid host in authAllowedHosts: https://autobrr.example.com")
	})

	t.Run("auth disabled with invalid client ip header is rejected", func(t *testing.T) {
		c := &Config{
			AuthDisabled:                true,
			AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue,
			AuthAllowedPeerCIDRs:        []string{"127.0.0.1/32"},
			CorsAllowedOrigins:          "https://autobrr.example.com",
			AuthClientIPHeader:          "X Real IP",
		}
		assert.EqualError(t, c.ValidateAuthConfig(), "authClientIPHeader is not a valid header name: X Real IP")
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
		assert.NoError(t, c.ValidateAuthConfig())
	})
}

func TestConfig_ValidateAuthConfigForward(t *testing.T) {
	valid := func() Config {
		return Config{
			AuthTrustedUserHeader: "Remote-User",
			AuthTrustedUser:       "ze0s",
			AuthAllowedPeerCIDRs:  []string{"127.0.0.1/32", "::1"},
			CorsAllowedOrigins:    "https://autobrr.example.com",
		}
	}

	tests := []struct {
		name    string
		modify  func(c *Config)
		wantErr string
	}{
		{name: "valid", modify: func(c *Config) {}},
		{name: "valid with logout url", modify: func(c *Config) { c.AuthLogoutURL = "https://auth.example.com/logout" }},
		{name: "missing trusted user", modify: func(c *Config) { c.AuthTrustedUser = "" }, wantErr: "authTrustedUser is required in forward authentication mode"},
		{name: "blank trusted user", modify: func(c *Config) { c.AuthTrustedUser = "  " }, wantErr: "authTrustedUser is required in forward authentication mode"},
		{name: "invalid header name", modify: func(c *Config) { c.AuthTrustedUserHeader = "Remote User" }, wantErr: "authTrustedUserHeader is not a valid header name: Remote User"},
		{name: "with auth disabled", modify: func(c *Config) { c.AuthDisabled = true }, wantErr: "authTrustedUserHeader cannot be used together with authDisabled"},
		{name: "with oidc", modify: func(c *Config) { c.OIDCEnabled = true }, wantErr: "authTrustedUserHeader cannot be used together with oidcEnabled"},
		{name: "with oidc only login", modify: func(c *Config) { c.OIDCDisableBuiltInLogin = true }, wantErr: "authTrustedUserHeader cannot be used together with oidcDisableBuiltInLogin"},
		{name: "with oidc auto redirect", modify: func(c *Config) { c.OIDCAutoRedirect = true }, wantErr: "authTrustedUserHeader cannot be used together with oidcAutoRedirect"},
		{name: "with oidc allowed subject", modify: func(c *Config) { c.OIDCAllowedSubject = "abc" }, wantErr: "authTrustedUserHeader cannot be used together with oidcAllowedSubject"},
		{name: "with oidc allowed email", modify: func(c *Config) { c.OIDCAllowedEmail = "me@example.com" }, wantErr: "authTrustedUserHeader cannot be used together with oidcAllowedEmail"},
		{name: "missing peers", modify: func(c *Config) { c.AuthAllowedPeerCIDRs = nil }, wantErr: "authAllowedPeerCIDRs is required in forward authentication mode"},
		{name: "peer range", modify: func(c *Config) { c.AuthAllowedPeerCIDRs = []string{"127.0.0.1/32", "172.18.0.0/16"} }, wantErr: "authAllowedPeerCIDRs entry 172.18.0.0/16 is a range: forward authentication mode only accepts single proxy addresses (/32 or /128)"},
		{name: "ipv6 peer range", modify: func(c *Config) { c.AuthAllowedPeerCIDRs = []string{"fd00::/64"} }, wantErr: "authAllowedPeerCIDRs entry fd00::/64 is a range"},
		{name: "missing origins", modify: func(c *Config) { c.CorsAllowedOrigins = "" }, wantErr: "corsAllowedOrigins must be set to explicit origins in forward authentication mode"},
		{name: "wildcard origin", modify: func(c *Config) { c.CorsAllowedOrigins = "*" }, wantErr: "wildcard origin in corsAllowedOrigins is not supported: *"},
		{name: "api key header", modify: func(c *Config) { c.AuthTrustedUserHeader = "x-api-token" }, wantErr: "authTrustedUserHeader cannot be X-API-Token, which carries API keys"},
		{name: "relative logout url", modify: func(c *Config) { c.AuthLogoutURL = "/logout" }, wantErr: "authLogoutURL must be an absolute http(s) URL: /logout"},
		{name: "logout url without host", modify: func(c *Config) { c.AuthLogoutURL = "https://:443/logout" }, wantErr: "invalid host in authLogoutURL: https://:443/logout"},
		{name: "logout url with out of range port", modify: func(c *Config) { c.AuthLogoutURL = "https://auth.example.com:99999/logout" }, wantErr: "invalid host in authLogoutURL: https://auth.example.com:99999/logout"},
		{name: "logout url with port", modify: func(c *Config) { c.AuthLogoutURL = "https://auth.example.com:8443/logout" }},
		{name: "javascript logout url", modify: func(c *Config) { c.AuthLogoutURL = "javascript:alert(1)" }, wantErr: "authLogoutURL must be an absolute http(s) URL: javascript:alert(1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid()
			tt.modify(&c)

			err := c.ValidateAuthConfig()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestConfig_IsTrustedUserAllowed(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		username string
		want     bool
	}{
		{name: "empty allows nobody", username: "someone", want: false},
		{name: "blank allows nobody", user: "  ", username: "", want: false},
		{name: "match", user: "ze0s", username: "ze0s", want: true},
		{name: "configured value is trimmed", user: " ze0s ", username: "ze0s", want: true},
		{name: "case variant is another account", user: "ze0s", username: "Ze0s", want: false},
		{name: "other user", user: "ze0s", username: "admin", want: false},
		{name: "no partial match", user: "ze0s", username: "ze0s2", want: false},
		{name: "a list is not split", user: "ze0s,admin", username: "admin", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{AuthTrustedUser: tt.user}
			assert.Equal(t, tt.want, c.IsTrustedUserAllowed(tt.username))
		})
	}
}

func TestConfig_AuthCapabilities(t *testing.T) {
	disabled := Config{AuthDisabled: true, AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue}

	tests := []struct {
		name     string
		config   Config
		wantMode AuthMode
		want     AuthCapabilities
	}{
		{
			name:     "builtin",
			config:   Config{},
			wantMode: AuthModeBuiltin,
			want:     AuthCapabilities{PasswordLogin: true, Onboarding: true, AccountManagement: true, APIKeyManagement: true},
		},
		{
			name:     "builtin ignores oidc only setting",
			config:   Config{OIDCDisableBuiltInLogin: true},
			wantMode: AuthModeBuiltin,
			want:     AuthCapabilities{PasswordLogin: true, Onboarding: true, AccountManagement: true, APIKeyManagement: true},
		},
		{
			name:     "unacknowledged auth disabled",
			config:   Config{AuthDisabled: true},
			wantMode: AuthModeBuiltin,
			want:     AuthCapabilities{PasswordLogin: true, Onboarding: true, AccountManagement: true, APIKeyManagement: true},
		},
		{
			name:     "oidc",
			config:   Config{OIDCEnabled: true},
			wantMode: AuthModeOIDC,
			want:     AuthCapabilities{PasswordLogin: true, OIDCLogin: true, AccountManagement: true, APIKeyManagement: true},
		},
		{
			name:     "oidc only",
			config:   Config{OIDCEnabled: true, OIDCDisableBuiltInLogin: true},
			wantMode: AuthModeOIDC,
			want:     AuthCapabilities{OIDCLogin: true, AccountManagement: true, APIKeyManagement: true},
		},
		{
			name:     "forward",
			config:   Config{AuthTrustedUserHeader: "Remote-User"},
			wantMode: AuthModeForward,
			want:     AuthCapabilities{APIKeyManagement: true, ProxyGuards: true},
		},
		{
			name:     "forward wins over disabled",
			config:   Config{AuthTrustedUserHeader: "Remote-User", AuthDisabled: true, AuthDisabledAcknowledgement: AuthDisabledAcknowledgementValue},
			wantMode: AuthModeForward,
			want:     AuthCapabilities{APIKeyManagement: true, ProxyGuards: true},
		},
		{
			name:     "disabled",
			config:   disabled,
			wantMode: AuthModeDisabled,
			want:     AuthCapabilities{ProxyGuards: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMode, tt.config.AuthMode())
			assert.Equal(t, tt.want, tt.config.AuthCapabilities())
		})
	}
}

func TestConfig_IsOIDCUserAllowed(t *testing.T) {
	tests := []struct {
		name           string
		allowedSubject string
		allowedEmail   string
		subject        string
		verifiedEmail  string
		want           bool
	}{
		{name: "unset allows anyone", subject: "anyone", want: true},
		{name: "subject match", allowedSubject: "abc-123", subject: "abc-123", want: true},
		{name: "subject is case-sensitive", allowedSubject: "ABC-123", subject: "abc-123", want: false},
		{name: "subject setting ignores matching email", allowedSubject: "me@example.com", subject: "xyz", verifiedEmail: "me@example.com", want: false},
		{name: "email match ignores case", allowedEmail: "Me@Example.com", subject: "abc-123", verifiedEmail: "me@example.com", want: true},
		{name: "email setting ignores matching subject", allowedEmail: "me@example.com", subject: "me@example.com", want: false},
		{name: "no verified email", allowedEmail: "me@example.com", subject: "abc-123", want: false},
		{name: "different user", allowedSubject: "abc-123", subject: "xyz", verifiedEmail: "other@example.com", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{OIDCAllowedSubject: tt.allowedSubject, OIDCAllowedEmail: tt.allowedEmail}
			assert.Equal(t, tt.want, c.IsOIDCUserAllowed(tt.subject, tt.verifiedEmail))
		})
	}
}
