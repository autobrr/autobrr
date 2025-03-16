// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

type Config struct {
	Version                 string
	ConfigPath              string
	Host                    string `toml:"host"`
	LogLevel                string `toml:"logLevel"`
	LogPath                 string `toml:"logPath"`
	BaseURL                 string `toml:"baseUrl"`
	SessionSecret           string `toml:"sessionSecret"`
	CustomDefinitions       string `toml:"customDefinitions"`
	DatabaseType            string `toml:"databaseType"`
	PostgresHost            string `toml:"postgresHost"`
	PostgresDatabase        string `toml:"postgresDatabase"`
	PostgresUser            string `toml:"postgresUser"`
	PostgresPass            string `toml:"postgresPass"`
	PostgresSSLMode         string `toml:"postgresSSLMode"`
	PostgresExtraParams     string `toml:"postgresExtraParams"`
	ProfilingHost           string `toml:"profilingHost"`
	OIDCIssuer              string `toml:"oidcIssuer"`
	OIDCClientID            string `toml:"oidcClientId"`
	OIDCClientSecret        string `toml:"oidcClientSecret"`
	OIDCRedirectURL         string `toml:"oidcRedirectUrl"`
	OIDCScopes              string `toml:"oidcScopes"`
	MetricsHost             string `toml:"metricsHost"`
	MetricsBasicAuthUsers   string `toml:"metricsBasicAuthUsers"`
	Port                    int    `toml:"port"`
	LogMaxSize              int    `toml:"logMaxSize"`
	LogMaxBackups           int    `toml:"logMaxBackups"`
	DatabaseMaxBackups      int    `toml:"databaseMaxBackups"`
	PostgresPort            int    `toml:"postgresPort"`
	ProfilingPort           int    `toml:"profilingPort"`
	MetricsPort             int    `toml:"metricsPort"`
	BaseURLModeLegacy       bool   `toml:"baseUrlModeLegacy"`
	CheckForUpdates         bool   `toml:"checkForUpdates"`
	ProfilingEnabled        bool   `toml:"profilingEnabled"`
	OIDCEnabled             bool   `toml:"oidcEnabled"`
	OIDCDisableBuiltInLogin bool   `toml:"oidcDisableBuiltInLogin"`
	MetricsEnabled          bool   `toml:"metricsEnabled"`
}

type ConfigUpdate struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	LogLevel        *string `json:"log_level,omitempty"`
	LogPath         *string `json:"log_path,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	CheckForUpdates *bool   `json:"check_for_updates,omitempty"`
}
