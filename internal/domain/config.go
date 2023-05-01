// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

type Config struct {
	Version           string
	ConfigPath        string
	Host              string `toml:"host"`
	Port              int    `toml:"port"`
	LogLevel          string `toml:"logLevel"`
	LogPath           string `toml:"logPath"`
	LogMaxSize        int    `toml:"logMaxSize"`
	LogMaxBackups     int    `toml:"logMaxBackups"`
	BaseURL           string `toml:"baseUrl"`
	SessionSecret     string `toml:"sessionSecret"`
	CustomDefinitions string `toml:"customDefinitions"`
	CheckForUpdates   bool   `toml:"checkForUpdates"`
	DatabaseType      string `toml:"databaseType"`
	PostgresHost      string `toml:"postgresHost"`
	PostgresPort      int    `toml:"postgresPort"`
	PostgresDatabase  string `toml:"postgresDatabase"`
	PostgresUser      string `toml:"postgresUser"`
	PostgresPass      string `toml:"postgresPass"`
}

type ConfigUpdate struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty"`
	LogLevel        *string `json:"log_level,omitempty"`
	LogPath         *string `json:"log_path,omitempty"`
	BaseURL         *string `json:"base_url,omitempty"`
	CheckForUpdates *bool   `json:"check_for_updates,omitempty"`
}
