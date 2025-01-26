// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

type Config struct {
	Version               string
	ConfigPath            string
	Host                  string `toml:"host"`
	Port                  int    `toml:"port"`
	LogLevel              string `toml:"logLevel"`
	LogPath               string `toml:"logPath"`
	LogMaxSize            int    `toml:"logMaxSize"`
	LogMaxBackups         int    `toml:"logMaxBackups"`
	BaseURL               string `toml:"baseUrl"`
	BaseURLModeLegacy     bool   `toml:"baseUrlModeLegacy"`
	SessionSecret         string `toml:"sessionSecret"`
	CustomDefinitions     string `toml:"customDefinitions"`
	CheckForUpdates       bool   `toml:"checkForUpdates"`
	DatabaseType          string `toml:"databaseType"`
	DatabaseMaxBackups    int    `toml:"databaseMaxBackups"`
	PostgresHost          string `toml:"postgresHost"`
	PostgresPort          int    `toml:"postgresPort"`
	PostgresDatabase      string `toml:"postgresDatabase"`
	PostgresUser          string `toml:"postgresUser"`
	PostgresPass          string `toml:"postgresPass"`
	PostgresSSLMode       string `toml:"postgresSSLMode"`
	PostgresExtraParams   string `toml:"postgresExtraParams"`
	ProfilingEnabled      bool   `toml:"profilingEnabled"`
	ProfilingHost         string `toml:"profilingHost"`
	ProfilingPort         int    `toml:"profilingPort"`
	OIDCEnabled           bool   `toml:"oidcEnabled" mapstructure:"oidc_enabled"`
	OIDCIssuer            string `toml:"oidcIssuer" mapstructure:"oidc_issuer"`
	OIDCClientID          string `toml:"oidcClientId" mapstructure:"oidc_client_id"`
	OIDCClientSecret      string `toml:"oidcClientSecret" mapstructure:"oidc_client_secret"`
	OIDCRedirectURL       string `toml:"oidcRedirectUrl" mapstructure:"oidc_redirect_url"`
	OIDCScopes            string `toml:"oidcScopes" mapstructure:"oidc_scopes"`
	DisableBuiltInLogin   bool   `toml:"disableBuiltInLogin" mapstructure:"disable_built_in_login"`
	MetricsEnabled        bool   `toml:"metricsEnabled"`
	MetricsHost           string `toml:"metricsHost"`
	MetricsPort           int    `toml:"metricsPort"`
	MetricsBasicAuthUsers string `toml:"metricsBasicAuthUsers"`
}

type ConfigUpdate struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	LogLevel        *string `json:"log_level,omitempty"`
	LogPath         *string `json:"log_path,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	CheckForUpdates *bool   `json:"check_for_updates,omitempty"`
}
