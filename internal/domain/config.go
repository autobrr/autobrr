// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"net/netip"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"

	"golang.org/x/net/http/httpguts"
)

type Config struct {
	Version                 string
	ConfigPath              string
	Host                    string `toml:"host"`
	Port                    int    `toml:"port"`
	CorsAllowedOrigins      string `toml:"corsAllowedOrigins"`
	LogLevel                string `toml:"logLevel"`
	LogPath                 string `toml:"logPath"`
	LogMaxSize              int    `toml:"logMaxSize"`
	LogMaxBackups           int    `toml:"logMaxBackups"`
	BaseURL                 string `toml:"baseUrl"`
	BaseURLModeLegacy       bool   `toml:"baseUrlModeLegacy"`
	CustomDefinitions       string `toml:"customDefinitions"`
	CheckForUpdates         bool   `toml:"checkForUpdates"`
	DatabaseType            string `toml:"databaseType"`
	DatabaseDSN             string `toml:"databaseDSN"`
	DatabaseMaxBackups      int    `toml:"databaseMaxBackups"`
	DatabaseAutoMigrate     bool   `toml:"databaseAutoMigrate"`
	PostgresHost            string `toml:"postgresHost"`
	PostgresPort            int    `toml:"postgresPort"`
	PostgresDatabase        string `toml:"postgresDatabase"`
	PostgresUser            string `toml:"postgresUser"`
	PostgresPass            string `toml:"postgresPass"`
	PostgresSSLMode         string `toml:"postgresSSLMode"`
	PostgresSocket          string `toml:"postgresSocket"`
	PostgresExtraParams     string `toml:"postgresExtraParams"`
	ProfilingEnabled        bool   `toml:"profilingEnabled"`
	ProfilingHost           string `toml:"profilingHost"`
	ProfilingPort           int    `toml:"profilingPort"`
	OIDCEnabled             bool   `toml:"oidcEnabled"`
	OIDCIssuer              string `toml:"oidcIssuer"`
	OIDCClientID            string `toml:"oidcClientId"`
	OIDCClientSecret        string `toml:"oidcClientSecret"`
	OIDCRedirectURL         string `toml:"oidcRedirectUrl"`
	OIDCScopes              string `toml:"oidcScopes"`
	OIDCDisableBuiltInLogin bool   `toml:"oidcDisableBuiltInLogin"`
	MetricsEnabled          bool   `toml:"metricsEnabled"`
	MetricsHost             string `toml:"metricsHost"`
	MetricsPort             int    `toml:"metricsPort"`
	MetricsBasicAuthUsers   string `toml:"metricsBasicAuthUsers"`

	// AuthDisabled disables all authentication when AuthDisabled is set and
	// AuthDisabledAcknowledgement is set to AuthDisabledAcknowledgementValue.
	// Intended only for deployments behind a reverse proxy that handles
	// authentication itself. Use IsAuthDisabled() to check whether auth is
	// actually disabled.
	AuthDisabled                bool   `toml:"authDisabled"`
	AuthDisabledAcknowledgement string `toml:"authDisabledAcknowledgement"`

	// AuthAllowedPeerCIDRs is the allowlist consulted while authentication
	// is disabled. Entries match the immediate TCP peer autobrr sees (normally the
	// reverse proxy or loopback), never a forwarded header or the end user's browser.
	AuthAllowedPeerCIDRs []string `toml:"authAllowedPeerCIDRs"`

	// AuthClientIPHeader optionally names a header the reverse proxy overwrites on
	// every request with the end user's IP (e.g. X-Real-IP). While authentication is
	// disabled it is resolved into the request context for access logs; it never
	// affects the peer allowlist, which always checks the TCP peer.
	AuthClientIPHeader string `toml:"authClientIPHeader"`
}

// AuthDisabledAcknowledgementValue is the exact value AuthDisabledAcknowledgement
// must be set to in order to disable authentication.
const AuthDisabledAcknowledgementValue = "I_ACKNOWLEDGE_THIS_IS_A_BAD_IDEA"

// IsAuthDisabled reports whether authentication is disabled. AuthDisabled must
// be set and AuthDisabledAcknowledgement must equal AuthDisabledAcknowledgementValue,
// so an operator cannot disable auth without explicitly acknowledging the risk.
func (c *Config) IsAuthDisabled() bool {
	return c.AuthDisabled && c.AuthDisabledAcknowledgement == AuthDisabledAcknowledgementValue
}

// ParseAuthAllowedPeerCIDRs parses AuthAllowedPeerCIDRs into netip.Prefix
// values. Bare IPs are treated as single-host prefixes (/32 for IPv4, /128 for IPv6).
// IPv4-mapped IPv6 forms are normalized to IPv4 so they compare consistently with real
// peers. Entries with non-zero host bits (e.g. 192.168.1.5/24) and universal prefixes
// (0.0.0.0/0, ::/0) are rejected to avoid ambiguous or all-address ranges.
func (c *Config) ParseAuthAllowedPeerCIDRs() ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(c.AuthAllowedPeerCIDRs))

	for _, entry := range c.AuthAllowedPeerCIDRs {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		var prefix netip.Prefix

		if strings.Contains(entry, "/") {
			p, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, errors.Wrap(err, "invalid CIDR in authAllowedPeerCIDRs: %s", entry)
			}

			if p.Addr().Is4In6() {
				if p.Bits() < 96 {
					return nil, errors.New("invalid CIDR in authAllowedPeerCIDRs: %s is an ambiguous IPv4-mapped IPv6 range", entry)
				}
				p = netip.PrefixFrom(p.Addr().Unmap(), p.Bits()-96)
			}

			if p != p.Masked() {
				return nil, errors.New("invalid CIDR in authAllowedPeerCIDRs: %s has non-zero host bits", entry)
			}

			prefix = p
		} else {
			addr, err := netip.ParseAddr(entry)
			if err != nil {
				return nil, errors.Wrap(err, "invalid IP in authAllowedPeerCIDRs: %s", entry)
			}

			addr = addr.Unmap()

			bits := 32
			if addr.Is6() {
				bits = 128
			}

			prefix = netip.PrefixFrom(addr, bits)
		}

		if prefix.Bits() == 0 {
			return nil, errors.New("invalid CIDR in authAllowedPeerCIDRs: %s matches all addresses", entry)
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}

// ValidateAuthDisabledConfig validates that disabling authentication is configured
// safely. It is a no-op when auth is not disabled.
func (c *Config) ValidateAuthDisabledConfig() error {
	if !c.IsAuthDisabled() {
		return nil
	}

	if c.OIDCEnabled {
		return errors.New("authDisabled cannot be used together with oidcEnabled")
	}

	if len(c.AuthAllowedPeerCIDRs) == 0 {
		return errors.New("authAllowedPeerCIDRs is required when authentication is disabled")
	}

	prefixes, err := c.ParseAuthAllowedPeerCIDRs()
	if err != nil {
		return err
	}

	if len(prefixes) == 0 {
		return errors.New("authAllowedPeerCIDRs contains no valid entries")
	}

	// A wildcard CORS origin combined with disabled auth turns any allowlisted-IP
	// browser into a cross-origin read/write primitive against the API, so require
	// explicit origins in this mode. Tokenize the value the same way it is consumed
	// (comma-split) so a wildcard hidden inside a list is still rejected.
	hasOrigin := false
	for origin := range strings.SplitSeq(c.CorsAllowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			return errors.New("corsAllowedOrigins must not contain a wildcard \"*\" when authentication is disabled")
		}
		hasOrigin = true
	}

	if !hasOrigin {
		return errors.New("corsAllowedOrigins must be set to explicit origins when authentication is disabled")
	}

	if header := strings.TrimSpace(c.AuthClientIPHeader); header != "" && !httpguts.ValidHeaderFieldName(header) {
		return errors.New("authClientIPHeader is not a valid header name: %s", header)
	}

	return nil
}

type ConfigUpdate struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	LogLevel        *string `json:"log_level,omitempty"`
	LogPath         *string `json:"log_path,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	CheckForUpdates *bool   `json:"check_for_updates,omitempty"`
}
