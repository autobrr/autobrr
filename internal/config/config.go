// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/autobrr/autobrr/internal/api"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var EnvVarPrefix = "AUTOBRR__"

var configTemplate = `# config.toml

# Hostname / IP
#
# Default: "localhost"
#
host = "{{ .host }}"

# Port
#
# Default: 7474
#
port = 7474

# Base url
# Set custom baseUrl eg /autobrr/ to serve in subdirectory.
# Not needed for subdomain, or by accessing with the :port directly.
#
# Optional
#
#baseUrl = "/autobrr/"

# Base url mode legacy
# This is kept for compatibility with older versions doing url rewrite on the proxy.
# If you use baseUrl you can set this to false and skip any url rewrite in your proxy.
#
# Default: true
#
baseUrlModeLegacy = true

# autobrr logs file
# If not defined, logs to stdout
# Make sure to use forward slashes and include the filename with extension. eg: "log/autobrr.log", "C:/autobrr/log/autobrr.log"
#
# Optional
#
#logPath = "log/autobrr.log"

# Log level
#
# Default: "DEBUG"
#
# Options: "ERROR", "DEBUG", "INFO", "WARN", "TRACE"
#
logLevel = "DEBUG"

# Log Max Size
#
# Default: 50
#
# Max log size in megabytes
#
#logMaxSize = 50

# Log Max Backups
#
# Default: 3
#
# Max amount of old log files
#
#logMaxBackups = 3

# Check for updates
#
checkForUpdates = true

# Session secret
#
sessionSecret = "{{ .sessionSecret }}"

# Database Max Backups
#
# Default: 5
#
#databaseMaxBackups = 5

# Golang pprof profiling and tracing
#
#profilingEnabled = false
#
#profilingHost = "127.0.0.1"
#
# Default: 6060
#profilingPort = 6060

# OpenID Connect Configuration
#
# Enable OIDC authentication
#oidcEnabled = false
#
# OIDC Issuer URL (e.g. https://auth.example.com)
#oidcIssuer = ""
#
# OIDC Client ID
#oidcClientId = ""
#
# OIDC Client Secret
#oidcClientSecret = ""
#
# OIDC Redirect URL (e.g. http://localhost:7474/api/auth/oidc/callback)
#oidcRedirectUrl = ""
#
# Disable Built In Login Form (only works when using external auth)
#oidcDisableBuiltInLogin = false

# Metrics
#
# Enable metrics endpoint
#metricsEnabled = true
#
# Metrics server host
#
#metricsHost = "127.0.0.1"
#
# Metrics server port
#
#metricsPort = 9074
#
# Metrics basic auth
#
# Comma separate list of user:password. Password must be htpasswd bcrypt hashed. Use autobrrctl to generate.
# Only enabled if correctly set with user:pass.
#
#metricsBasicAuthUsers = ""

# Custom definitions
#
#customDefinitions = "test/definitions"
`

func (c *AppConfig) writeConfig(configPath string, configFile string) error {
	cfgPath := filepath.Join(configPath, configFile)

	// check if configPath exists, if not create it
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		err := os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	// check if config exists, if not create it
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		// set default host
		host := "127.0.0.1"

		if _, err := os.Stat("/.dockerenv"); err == nil {
			// docker creates a .dockerenv file at the root
			// of the directory tree inside the container.
			// if this file exists then the viewer is running
			// from inside a docker container so return true
			host = "0.0.0.0"
		} else if _, err := os.Stat("/dev/.lxc-boot-id"); err == nil {
			// lxc creates this file containing the uuid
			// of the container in every boot.
			// if this file exists then the viewer is running
			// from inside a lxc container so return true
			host = "0.0.0.0"
		} else if os.Getpid() == 1 {
			// if we're running as pid 1, we're honoured.
			// but there's a good chance this is an isolated namespace
			// or a container.
			host = "0.0.0.0"
		} else if user := os.Getenv("USERNAME"); user == "ContainerAdministrator" || user == "ContainerUser" {
			/* this is the correct code below, but golang helpfully Panics when it can't find netapi32.dll
			   the issue was first reported 7 years ago, but is fixed in go 1.24 where the below code works.
			*/
			/*
				 u, err := user.Current(); err == nil && u != nil &&
				(u.Name == "ContainerAdministrator" || u.Name == "ContainerUser") {
				// Windows conatiners run containers as ContainerAdministrator by default */
			host = "0.0.0.0"
		} else if pd, _ := os.Open("/proc/1/cgroup"); pd != nil {
			defer pd.Close()
			b := make([]byte, 4096)
			pd.Read(b)
			if strings.Contains(string(b), "/docker") || strings.Contains(string(b), "/lxc") {
				host = "0.0.0.0"
			}
		}

		f, err := os.Create(cfgPath)
		if err != nil { // perm 0666
			// handle failed create
			log.Printf("error creating file: %q", err)
			return err
		}
		defer f.Close()

		// setup text template to inject variables into
		tmpl, err := template.New("config").Parse(configTemplate)
		if err != nil {
			return errors.Wrap(err, "could not create config template")
		}

		tmplVars := map[string]string{
			"host":          host,
			"sessionSecret": c.Config.SessionSecret,
		}

		var buffer bytes.Buffer
		if err = tmpl.Execute(&buffer, &tmplVars); err != nil {
			return errors.Wrap(err, "could not write torrent url template output")
		}

		if _, err = f.WriteString(buffer.String()); err != nil {
			log.Printf("error writing contents to file: %v %q", configPath, err)
			return err
		}

		return f.Sync()
	}

	return nil
}

type Config interface {
	UpdateConfig() error
	DynamicReload(log logger.Logger)
}

type AppConfig struct {
	Config *domain.Config
	m      *sync.Mutex
}

func New(configPath string, version string) *AppConfig {
	c := &AppConfig{
		m: new(sync.Mutex),
	}
	c.defaults()
	c.Config.Version = version
	c.Config.ConfigPath = configPath

	c.load(configPath)
	c.loadFromEnv()

	return c
}

func (c *AppConfig) defaults() {
	c.Config = &domain.Config{
		Version:               "dev",
		Host:                  "localhost",
		Port:                  7474,
		LogLevel:              "TRACE",
		LogPath:               "",
		LogMaxSize:            50,
		LogMaxBackups:         3,
		DatabaseMaxBackups:    5,
		BaseURL:               "/",
		BaseURLModeLegacy:     true,
		SessionSecret:         api.GenerateSecureToken(16),
		CustomDefinitions:     "",
		CheckForUpdates:       true,
		DatabaseType:          "sqlite",
		DatabaseDSN:           "",
		PostgresHost:          "",
		PostgresPort:          0,
		PostgresDatabase:      "",
		PostgresUser:          "",
		PostgresPass:          "",
		PostgresSSLMode:       "disable",
		PostgresExtraParams:   "",
		PostgresSocket:        "",
		ProfilingEnabled:      false,
		ProfilingHost:         "127.0.0.1",
		ProfilingPort:         6060,
		MetricsEnabled:        false,
		MetricsHost:           "127.0.0.1",
		MetricsPort:           9074,
		MetricsBasicAuthUsers: "",
	}

}

func (c *AppConfig) loadFromEnv() {
	if v := GetEnvStr("HOST"); v != "" {
		c.Config.Host = v
	}

	if v := GetEnvInt("PORT"); v > 0 {
		c.Config.Port = v
	}

	if v := GetEnvStr("BASE_URL"); v != "" {
		c.Config.BaseURL = v
	}

	if v := GetEnvStr("BASE_URL_MODE_LEGACY"); v != "" {
		c.Config.BaseURLModeLegacy = strings.EqualFold(strings.ToLower(v), "true")
	}

	if v := GetEnvStr("LOG_LEVEL"); v != "" {
		c.Config.LogLevel = v
	}

	if v := GetEnvStr("LOG_PATH"); v != "" {
		c.Config.LogPath = v
	}

	if v := GetEnvInt("LOG_MAX_SIZE"); v > 0 {
		c.Config.LogMaxSize = v
	}

	if v := GetEnvInt("LOG_MAX_BACKUPS"); v > 0 {
		c.Config.LogMaxBackups = v
	}

	if v := GetEnvStr("SESSION_SECRET"); v != "" {
		c.Config.SessionSecret = v
	}

	if v := GetEnvStr("CUSTOM_DEFINITIONS"); v != "" {
		c.Config.CustomDefinitions = v
	}

	if v := GetEnvStr("CHECK_FOR_UPDATES"); v != "" {
		c.Config.CheckForUpdates = strings.EqualFold(strings.ToLower(v), "true")
	}

	if v := GetEnvStr("DATABASE_DSN"); v != "" {
		c.Config.DatabaseDSN = v
	}

	if v := GetEnvStr("DATABASE_TYPE"); v != "" {
		if validDatabaseType(v) {
			c.Config.DatabaseType = v
		}
	}

	if v := GetEnvInt("DATABASE_MAX_BACKUPS"); v > 0 {
		c.Config.DatabaseMaxBackups = v
	}

	if v := GetEnvStr("POSTGRES_HOST"); v != "" {
		c.Config.PostgresHost = v
	}

	if v := GetEnvInt("POSTGRES_PORT"); v > 0 {
		c.Config.PostgresPort = v
	}

	if v := GetEnvStr("POSTGRES_DATABASE"); v != "" {
		c.Config.PostgresDatabase = v
	}

	if v := GetEnvStr("POSTGRES_DB"); v != "" {
		c.Config.PostgresDatabase = v
	}

	if v := GetEnvStr("POSTGRES_USER"); v != "" {
		c.Config.PostgresUser = v
	}

	if v := GetEnvStr("POSTGRES_PASS"); v != "" {
		c.Config.PostgresPass = v
	}

	if v := GetEnvStr("POSTGRES_PASSWORD"); v != "" {
		c.Config.PostgresPass = v
	}

	if v := GetEnvStr("POSTGRES_SSLMODE"); v != "" {
		c.Config.PostgresSSLMode = v
	}

	if v := GetEnvStr("POSTGRES_SOCKET"); v != "" {
		c.Config.PostgresSocket = v
	}

	if v := GetEnvStr("POSTGRES_EXTRA_PARAMS"); v != "" {
		c.Config.PostgresExtraParams = v
	}

	if v := GetEnvStr("PROFILING_ENABLED"); v != "" {
		c.Config.ProfilingEnabled = strings.EqualFold(strings.ToLower(v), "true")
	}

	if v := GetEnvStr("PROFILING_HOST"); v != "" {
		c.Config.ProfilingHost = v
	}

	if v := GetEnvInt("PROFILING_PORT"); v > 0 {
		c.Config.ProfilingPort = v
	}

	// OIDC Configuration
	if v := GetEnvStr("OIDC_ENABLED"); v != "" {
		c.Config.OIDCEnabled = strings.EqualFold(strings.ToLower(v), "true")
	}

	if v := GetEnvStr("OIDC_ISSUER"); v != "" {
		c.Config.OIDCIssuer = v
	}

	if v := GetEnvStr("OIDC_CLIENT_ID"); v != "" {
		c.Config.OIDCClientID = v
	}

	if v := GetEnvStr("OIDC_CLIENT_SECRET"); v != "" {
		c.Config.OIDCClientSecret = v
	}

	if v := GetEnvStr("OIDC_REDIRECT_URL"); v != "" {
		c.Config.OIDCRedirectURL = v
	}

	if v := GetEnvStr("OIDC_DISABLE_BUILT_IN_LOGIN"); v != "" {
		c.Config.OIDCDisableBuiltInLogin = strings.EqualFold(strings.ToLower(v), "true")
	}

	if v := GetEnvStr("METRICS_ENABLED"); v != "" {
		c.Config.MetricsEnabled = strings.EqualFold(strings.ToLower(v), "true")
	}

	if v := GetEnvStr("METRICS_HOST"); v != "" {
		c.Config.MetricsHost = v
	}

	if v := GetEnvInt("METRICS_PORT"); v > 0 {
		c.Config.MetricsPort = v
	}

	if v := GetEnvStr("METRICS_BASIC_AUTH_USERS"); v != "" {
		c.Config.MetricsBasicAuthUsers = v
	}
}

func GetEnvStr(key string) string {
	// first check if we have a variable with a _FILE ending
	// commonly used for docker secrets and similar
	if filePath := os.Getenv(EnvVarPrefix + key + "_FILE"); filePath != "" {
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Could not read file: %s err: %q", filePath, err)
			return ""
		}
		return strings.TrimSpace(string(content))
	}

	if v := os.Getenv(EnvVarPrefix + key); v != "" {
		return v
	}

	return ""
}

func GetEnvInt(key string) int {
	value := GetEnvStr(key)

	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0
	}
	return int(i)
}

func validDatabaseType(v string) bool {
	valid := []string{"sqlite", "postgres"}
	for _, s := range valid {
		if s == v {
			return true
		}
	}

	return false
}

func (c *AppConfig) load(configPath string) {
	viper.SetConfigType("toml")

	// clean trailing slash from configPath
	configPath = path.Clean(configPath)

	if configPath != "" {
		//viper.SetConfigName("config")

		// check if path and file exists
		// if not, create path and file
		if err := c.writeConfig(configPath, "config.toml"); err != nil {
			log.Printf("write error: %q", err)
		}

		viper.SetConfigFile(path.Join(configPath, "config.toml"))
	} else {
		viper.SetConfigName("config")

		// Search config in directories
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/autobrr")
		viper.AddConfigPath("$HOME/.autobrr")
	}

	// read config
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config read error: %q", err)
	}

	if err := viper.Unmarshal(c.Config); err != nil {
		log.Fatalf("Could not unmarshal config file: %v: err %q", viper.ConfigFileUsed(), err)
	}
}

func (c *AppConfig) DynamicReload(log logger.Logger) {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		c.m.Lock()
		defer c.m.Unlock()

		logLevel := viper.GetString("logLevel")
		c.Config.LogLevel = logLevel
		log.SetLogLevel(c.Config.LogLevel)

		logPath := viper.GetString("logPath")
		c.Config.LogPath = logPath

		checkUpdates := viper.GetBool("checkForUpdates")
		c.Config.CheckForUpdates = checkUpdates

		log.Debug().Msg("config file reloaded!")
	})
}

func (c *AppConfig) UpdateConfig() error {
	filePath := path.Join(c.Config.ConfigPath, "config.toml")

	f, err := os.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "could not read config file: %s", filePath)
	}

	lines := strings.Split(string(f), "\n")
	lines = c.processLines(lines)

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
		return errors.Wrap(err, "could not write config file: %s", filePath)
	}

	return nil
}

func (c *AppConfig) processLines(lines []string) []string {
	// keep track of not found values to append at bottom
	var (
		foundLineUpdate   = false
		foundLineLogLevel = false
		foundLineLogPath  = false
	)

	for i, line := range lines {
		// set checkForUpdates
		if !foundLineUpdate && strings.Contains(line, "checkForUpdates =") {
			lines[i] = fmt.Sprintf("checkForUpdates = %t", c.Config.CheckForUpdates)
			foundLineUpdate = true
		}
		if !foundLineLogLevel && strings.Contains(line, "logLevel =") {
			lines[i] = fmt.Sprintf(`logLevel = "%s"`, c.Config.LogLevel)
			foundLineLogLevel = true
		}
		if !foundLineLogPath && strings.Contains(line, "logPath =") {
			if c.Config.LogPath == "" {
				// Check if the line already has a value
				matches := strings.Split(line, "=")
				if len(matches) > 1 && strings.TrimSpace(matches[1]) != `""` {
					lines[i] = line // Preserve the existing line
				} else {
					lines[i] = `#logPath = ""`
				}
			} else {
				lines[i] = fmt.Sprintf("logPath = \"%s\"", c.Config.LogPath)
			}
			foundLineLogPath = true
		}
	}

	// append missing vars to bottom
	if !foundLineUpdate {
		lines = append(lines, "# Check for updates")
		lines = append(lines, "#")
		lines = append(lines, fmt.Sprintf("checkForUpdates = %t", c.Config.CheckForUpdates))
	}

	if !foundLineLogLevel {
		lines = append(lines, "# Log level")
		lines = append(lines, "#")
		lines = append(lines, `# Default: "DEBUG"`)
		lines = append(lines, "#")
		lines = append(lines, `# Options: "ERROR", "DEBUG", "INFO", "WARN", "TRACE"`)
		lines = append(lines, "#")
		lines = append(lines, fmt.Sprintf(`logLevel = "%s"`, c.Config.LogLevel))
	}

	if !foundLineLogPath {
		lines = append(lines, "# Log Path")
		lines = append(lines, "#")
		lines = append(lines, "# Optional")
		lines = append(lines, "#")
		if c.Config.LogPath == "" {
			lines = append(lines, `#logPath = ""`)
		} else {
			lines = append(lines, fmt.Sprintf(`logPath = "%s"`, c.Config.LogPath))
		}
	}

	return lines
}
