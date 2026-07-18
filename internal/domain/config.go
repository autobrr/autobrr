// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"net/netip"
	"strings"

	"github.com/autobrr/autobrr/pkg/errors"
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

	// AuthDisabled disables all authentication when both AuthDisabled and
	// IAcknowledgeThisIsABadIdea are set. Intended only for deployments behind
	// a reverse proxy that handles authentication itself. Use IsAuthDisabled()
	// to check whether auth is actually disabled.
	AuthDisabled               bool     `toml:"authDisabled"`
	IAcknowledgeThisIsABadIdea bool     `toml:"I_ACKNOWLEDGE_THIS_IS_A_BAD_IDEA"`
	AuthDisabledAllowedCIDRs   []string `toml:"authDisabledAllowedCIDRs"`
}

// IsAuthDisabled reports whether authentication is disabled. Both AuthDisabled
// and IAcknowledgeThisIsABadIdea must be set, so an operator cannot disable
// auth without explicitly acknowledging the risk.
func (c *Config) IsAuthDisabled() bool {
	return c.AuthDisabled && c.IAcknowledgeThisIsABadIdea
}

// ParseAuthDisabledAllowedCIDRs parses AuthDisabledAllowedCIDRs into netip.Prefix values.
// Bare IPs are treated as single-host prefixes (/32 for IPv4, /128 for IPv6). Entries
// with non-zero host bits (e.g. 192.168.1.5/24) are rejected to avoid ambiguous ranges.
func (c *Config) ParseAuthDisabledAllowedCIDRs() ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(c.AuthDisabledAllowedCIDRs))

	for _, entry := range c.AuthDisabledAllowedCIDRs {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			prefix, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, errors.Wrap(err, "invalid CIDR in authDisabledAllowedCIDRs: %s", entry)
			}

			if prefix != prefix.Masked() {
				return nil, errors.New("invalid CIDR in authDisabledAllowedCIDRs: %s has non-zero host bits", entry)
			}

			prefixes = append(prefixes, prefix)
			continue
		}

		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return nil, errors.Wrap(err, "invalid IP in authDisabledAllowedCIDRs: %s", entry)
		}

		bits := 32
		if addr.Is6() {
			bits = 128
		}

		prefixes = append(prefixes, netip.PrefixFrom(addr, bits))
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

	if len(c.AuthDisabledAllowedCIDRs) == 0 {
		return errors.New("authDisabledAllowedCIDRs is required when authentication is disabled")
	}

	if _, err := c.ParseAuthDisabledAllowedCIDRs(); err != nil {
		return err
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
